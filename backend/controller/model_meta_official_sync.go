package controller

// 🌸 官方元数据仓库同步～对齐 new-api 的 SyncUpstreamModels 设计
// 从 basellm.github.io/llm-metadata 拉取完整的模型元信息
// （描述、图标、标签、端点、匹配规则、供应商），补齐缺失模型并支持冲突覆盖

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"STfreApi/common"
	"STfreApi/model"

	"github.com/gin-gonic/gin"
)

const officialMetaBase = "https://basellm.github.io/llm-metadata"

// ～根据语言拼出上游元数据地址～
// 实测上游仓库的多语言目录是 zh / en / ja（zh-CN 会 404 哦）
func officialMetaURLs(locale string) (modelsURL, vendorsURL string) {
	base := strings.TrimRight(officialMetaBase, "/")
	l := strings.ToLower(strings.TrimSpace(locale))
	switch l {
	case "zh", "zh-cn", "zh_cn":
		l = "zh"
	case "en", "ja":
		// 保持不变
	default:
		return base + "/api/newapi/models.json", base + "/api/newapi/vendors.json"
	}
	return fmt.Sprintf("%s/api/i18n/%s/newapi/models.json", base, l),
		fmt.Sprintf("%s/api/i18n/%s/newapi/vendors.json", base, l)
}

type officialEnvelope[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    []T    `json:"data"`
}

type officialModel struct {
	Description string          `json:"description"`
	Endpoints   json.RawMessage `json:"endpoints"`
	Icon        string          `json:"icon"`
	ModelName   string          `json:"model_name"`
	NameRule    int             `json:"name_rule"`
	Status      int             `json:"status"`
	Tags        string          `json:"tags"`
	VendorName  string          `json:"vendor_name"`
	InputPrice  *float64        `json:"price_per_m_input"`  // 🎀 $/1M tokens～
	OutputPrice *float64        `json:"price_per_m_output"` // 🎀 $/1M tokens～
}

type officialVendor struct {
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Name        string `json:"name"`
	Status      int    `json:"status"`
}

// ～简单的 ETag/内容缓存，减少重复拉取～
var (
	officialEtagCache = make(map[string]string)
	officialBodyCache = make(map[string][]byte)
	officialCacheMu   sync.RWMutex
)

func fetchOfficialJSON[T any](ctx context.Context, url string, out *officialEnvelope[T]) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	officialCacheMu.RLock()
	if et := officialEtagCache[url]; et != "" {
		req.Header.Set("If-None-Match", et)
	}
	officialCacheMu.RUnlock()

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var buf []byte
	switch resp.StatusCode {
	case http.StatusOK:
		buf, err = io.ReadAll(io.LimitReader(resp.Body, 20<<20))
		if err != nil {
			return err
		}
		officialCacheMu.Lock()
		if et := resp.Header.Get("ETag"); et != "" {
			officialEtagCache[url] = et
		}
		officialBodyCache[url] = buf
		officialCacheMu.Unlock()
	case http.StatusNotModified:
		officialCacheMu.RLock()
		buf = officialBodyCache[url]
		officialCacheMu.RUnlock()
		if len(buf) == 0 {
			return errors.New("缓存缺失（304 但无本地缓存）")
		}
	default:
		return errors.New("上游返回 " + resp.Status)
	}

	// 兼容 envelope 和纯数组两种格式
	if err := json.Unmarshal(buf, out); err != nil {
		var arr []T
		if err2 := json.Unmarshal(buf, &arr); err2 != nil {
			return err
		}
		out.Success = true
		out.Data = arr
	} else if !out.Success && len(out.Data) == 0 && out.Message == "" {
		out.Success = true
	}
	return nil
}

// ～并发拉取 models + vendors～
func fetchOfficialMeta(ctx context.Context, locale string) (map[string]officialModel, map[string]officialVendor, string, string, error) {
	modelsURL, vendorsURL := officialMetaURLs(locale)

	var modelsEnv officialEnvelope[officialModel]
	var vendorsEnv officialEnvelope[officialVendor]
	var fetchErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		// vendors 拉取失败不阻塞（供应商信息缺了还能同步模型）
		_ = fetchOfficialJSON(ctx, vendorsURL, &vendorsEnv)
	}()
	go func() {
		defer wg.Done()
		if err := fetchOfficialJSON(ctx, modelsURL, &modelsEnv); err != nil {
			fetchErr = err
		}
	}()
	wg.Wait()
	if fetchErr != nil {
		return nil, nil, modelsURL, vendorsURL, fetchErr
	}

	modelByName := make(map[string]officialModel, len(modelsEnv.Data))
	for _, m := range modelsEnv.Data {
		if m.ModelName != "" {
			modelByName[m.ModelName] = m
		}
	}
	vendorByName := make(map[string]officialVendor, len(vendorsEnv.Data))
	for _, v := range vendorsEnv.Data {
		if v.Name != "" {
			vendorByName[v.Name] = v
		}
	}
	return modelByName, vendorByName, modelsURL, vendorsURL, nil
}

// ～确保供应商存在，不存在就用上游信息创建～
func ensureOfficialVendor(vendorName string, vendorByName map[string]officialVendor, cache map[string]int, createdVendors *int) int {
	if vendorName == "" {
		return 0
	}
	if id, ok := cache[vendorName]; ok {
		return id
	}
	var existing model.Vendor
	if err := common.DB.Where("name = ?", vendorName).First(&existing).Error; err == nil {
		// 顺手补图标～
		if existing.Icon == "" {
			if uv, ok := vendorByName[vendorName]; ok && uv.Icon != "" {
				_ = common.DB.Model(&model.Vendor{}).Where("id = ?", existing.Id).Update("icon", uv.Icon).Error
			}
		}
		cache[vendorName] = existing.Id
		return existing.Id
	}
	uv := vendorByName[vendorName]
	v := &model.Vendor{
		Name:        vendorName,
		Code:        strings.ToLower(strings.ReplaceAll(vendorName, " ", "-")),
		Icon:        uv.Icon,
		Description: uv.Description,
		Status:      model.VendorStatusEnabled,
	}
	if err := v.Insert(); err == nil {
		*createdVendors++
		cache[vendorName] = v.Id
		return v.Id
	}
	cache[vendorName] = 0
	return 0
}

// ～endpoints 原始 JSON 可能是 null，要过滤掉哦～
func officialEndpointsStr(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return ""
	}
	return s
}

// ～收集本地缺失（渠道在用但元数据表没有）的模型名～
func officialMissingModelNames() []string {
	channels, err := getActiveChannelsByIDs(nil)
	if err != nil {
		return nil
	}
	referenced := collectChannelModelNames(channels)
	existingSet := queryExistingModelMetaNames(referenced)
	missing := make([]string, 0)
	for _, name := range referenced {
		if !existingSet[name] {
			missing = append(missing, name)
		}
	}
	return missing
}

// SyncOfficialPreview 对齐 new-api: GET /api/models/sync_official/preview
// 返回：可从官方仓库补齐的缺失模型 + 本地与上游的字段冲突列表
func SyncOfficialPreview(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()

	locale := c.Query("locale")
	modelByName, vendorByName, modelsURL, vendorsURL, err := fetchOfficialMeta(ctx, locale)
	if err != nil {
		common.Fail(c, common.CodeServerError, "获取官方元数据失败: "+err.Error())
		return
	}
	_ = vendorByName

	// 缺失且上游有元数据的模型
	missingAll := officialMissingModelNames()
	missing := make([]string, 0)
	for _, name := range missingAll {
		if _, ok := modelByName[name]; ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)

	// 本地已有且开启官方同步的模型 → 冲突检测
	upstreamNames := make([]string, 0, len(modelByName))
	for name := range modelByName {
		upstreamNames = append(upstreamNames, name)
	}

	var locals []model.ModelMeta
	if len(upstreamNames) > 0 {
		_ = common.DB.Where("model_name IN ? AND sync_official = ?", upstreamNames, true).Find(&locals).Error
	}

	// 本地 vendor id → name
	vendorIDSet := make(map[int]struct{})
	for _, m := range locals {
		if m.VendorId != 0 {
			vendorIDSet[m.VendorId] = struct{}{}
		}
	}
	vendorIDs := make([]int, 0, len(vendorIDSet))
	for id := range vendorIDSet {
		vendorIDs = append(vendorIDs, id)
	}
	idToVendorName := make(map[int]string)
	if len(vendorIDs) > 0 {
		var dbVendors []model.Vendor
		_ = common.DB.Where("id IN ?", vendorIDs).Find(&dbVendors).Error
		for _, v := range dbVendors {
			idToVendorName[v.Id] = v.Name
		}
	}

	type conflictField struct {
		Field    string      `json:"field"`
		Local    interface{} `json:"local"`
		Upstream interface{} `json:"upstream"`
	}
	type conflictItem struct {
		ModelName string          `json:"model_name"`
		Fields    []conflictField `json:"fields"`
	}

	conflicts := make([]conflictItem, 0)
	for _, local := range locals {
		up, ok := modelByName[local.ModelName]
		if !ok {
			continue
		}
		fields := make([]conflictField, 0, 6)
		if strings.TrimSpace(local.Description) != strings.TrimSpace(up.Description) {
			fields = append(fields, conflictField{Field: "description", Local: local.Description, Upstream: up.Description})
		}
		if strings.TrimSpace(local.Icon) != strings.TrimSpace(up.Icon) {
			fields = append(fields, conflictField{Field: "icon", Local: local.Icon, Upstream: up.Icon})
		}
		if strings.TrimSpace(local.Tags) != strings.TrimSpace(up.Tags) {
			fields = append(fields, conflictField{Field: "tags", Local: local.Tags, Upstream: up.Tags})
		}
		localVendor := idToVendorName[local.VendorId]
		if strings.TrimSpace(localVendor) != strings.TrimSpace(up.VendorName) {
			fields = append(fields, conflictField{Field: "vendor", Local: localVendor, Upstream: up.VendorName})
		}
		if local.NameRule != up.NameRule {
			fields = append(fields, conflictField{Field: "name_rule", Local: local.NameRule, Upstream: up.NameRule})
		}
		if len(fields) > 0 {
			conflicts = append(conflicts, conflictItem{ModelName: local.ModelName, Fields: fields})
		}
	}

	common.OK(c, gin.H{
		"missing":   missing,
		"conflicts": conflicts,
		"source": gin.H{
			"locale":      locale,
			"models_url":  modelsURL,
			"vendors_url": vendorsURL,
		},
	})
}

type officialOverwriteField struct {
	ModelName string   `json:"model_name"`
	Fields    []string `json:"fields"`
}

type officialSyncRequest struct {
	Overwrite []officialOverwriteField `json:"overwrite"`
	Locale    string                   `json:"locale"`
}

// SyncOfficialModels 对齐 new-api: POST /api/models/sync_official
// - 补齐缺失模型（带完整元信息：描述/图标/标签/端点/匹配规则/供应商）
// - 已存在的模型：也回填空缺字段（描述、图标、标签、端点、供应商）
// - 可选 overwrite：按用户批准的冲突字段强制覆盖
func SyncOfficialModels(c *gin.Context) {
	var req officialSyncRequest
	_ = c.ShouldBindJSON(&req)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	modelByName, vendorByName, modelsURL, vendorsURL, err := fetchOfficialMeta(ctx, req.Locale)
	if err != nil {
		common.Fail(c, common.CodeServerError, "获取官方元数据失败: "+err.Error())
		return
	}

	createdModels := 0
	createdVendors := 0
	updatedModels := 0
	enrichedModels := 0
	skipped := make([]string, 0)
	createdList := make([]string, 0)
	updatedList := make([]string, 0)

	vendorIDCache := make(map[string]int)

	// 1) 补齐缺失模型（带完整元信息）
	for _, name := range officialMissingModelNames() {
		up, ok := modelByName[name]
		if !ok {
			skipped = append(skipped, name)
			continue
		}
		vendorID := ensureOfficialVendor(up.VendorName, vendorByName, vendorIDCache, &createdVendors)

		item := &model.ModelMeta{
			VendorId:     vendorID,
			ModelName:    name,
			DisplayName:  name,
			Description:  up.Description,
			Icon:         up.Icon,
			Tags:         up.Tags,
			Endpoints:    officialEndpointsStr(up.Endpoints),
			NameRule:     up.NameRule,
			ModelType:    "text",
			Enabled:      true,
			SyncOfficial: true,
		}
		if up.InputPrice != nil {
			item.InputPrice = *up.InputPrice
		}
		if up.OutputPrice != nil {
			item.OutputPrice = *up.OutputPrice
		}
		if insertErr := item.Insert(); insertErr != nil {
			skipped = append(skipped, name)
			continue
		}
		createdModels++
		createdList = append(createdList, name)
	}

	// 2) 已有模型回填空缺字段（不覆盖用户已填的内容）
	upstreamNames := make([]string, 0, len(modelByName))
	for name := range modelByName {
		upstreamNames = append(upstreamNames, name)
	}
	var locals []model.ModelMeta
	if len(upstreamNames) > 0 {
		_ = common.DB.Where("model_name IN ? AND sync_official = ?", upstreamNames, true).Find(&locals).Error
	}
	for _, local := range locals {
		up := modelByName[local.ModelName]
		updates := map[string]interface{}{}
		if strings.TrimSpace(local.Description) == "" && up.Description != "" {
			updates["description"] = up.Description
		}
		if strings.TrimSpace(local.Icon) == "" && up.Icon != "" {
			updates["icon"] = up.Icon
		}
		if strings.TrimSpace(local.Tags) == "" && up.Tags != "" {
			updates["tags"] = up.Tags
		}
		if strings.TrimSpace(local.Endpoints) == "" {
			if eps := officialEndpointsStr(up.Endpoints); eps != "" {
				updates["endpoints"] = eps
			}
		}
		if local.VendorId == 0 && up.VendorName != "" {
			if vid := ensureOfficialVendor(up.VendorName, vendorByName, vendorIDCache, &createdVendors); vid > 0 {
				updates["vendor_id"] = vid
			}
		}
		// 🎀 价格为 0 时用官方价格补上～
		if local.InputPrice == 0 && up.InputPrice != nil && *up.InputPrice > 0 {
			updates["input_price"] = *up.InputPrice
		}
		if local.OutputPrice == 0 && up.OutputPrice != nil && *up.OutputPrice > 0 {
			updates["output_price"] = *up.OutputPrice
		}
		if len(updates) > 0 {
			if err := common.DB.Model(&model.ModelMeta{}).Where("id = ?", local.Id).Updates(updates).Error; err == nil {
				enrichedModels++
			}
		}
	}

	// 3) 用户批准的冲突覆盖
	for _, ow := range req.Overwrite {
		up, ok := modelByName[ow.ModelName]
		if !ok {
			continue
		}
		var local model.ModelMeta
		if err := common.DB.Where("model_name = ?", ow.ModelName).First(&local).Error; err != nil {
			continue
		}
		if !local.SyncOfficial {
			continue
		}
		updates := map[string]interface{}{}
		for _, f := range ow.Fields {
			switch strings.ToLower(strings.TrimSpace(f)) {
			case "description":
				updates["description"] = up.Description
			case "icon":
				updates["icon"] = up.Icon
			case "tags":
				updates["tags"] = up.Tags
			case "endpoints":
				updates["endpoints"] = officialEndpointsStr(up.Endpoints)
			case "name_rule":
				updates["name_rule"] = up.NameRule
			case "vendor":
				if vid := ensureOfficialVendor(up.VendorName, vendorByName, vendorIDCache, &createdVendors); vid > 0 {
					updates["vendor_id"] = vid
				}
			}
		}
		if len(updates) > 0 {
			if err := common.DB.Model(&model.ModelMeta{}).Where("id = ?", local.Id).Updates(updates).Error; err == nil {
				updatedModels++
				updatedList = append(updatedList, ow.ModelName)
			}
		}
	}

	common.OK(c, gin.H{
		"created_models":  createdModels,
		"created_vendors": createdVendors,
		"updated_models":  updatedModels,
		"enriched_models": enrichedModels,
		"skipped_models":  skipped,
		"created_list":    createdList,
		"updated_list":    updatedList,
		"source": gin.H{
			"locale":      req.Locale,
			"models_url":  modelsURL,
			"vendors_url": vendorsURL,
		},
	})
}
