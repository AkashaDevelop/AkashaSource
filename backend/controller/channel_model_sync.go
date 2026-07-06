package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service"

	"github.com/gin-gonic/gin"
)

func fetchUpstreamModelList(channel model.Channel) ([]string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(channel.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	targetURL := baseURL + "/v1/models"

	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	key := service.GetNextKey(channel.Key)
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := common.NewHTTPClient(channel.Proxy).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("upstream status %d", resp.StatusCode)
	}

	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	models := make([]string, 0, len(payload.Data))
	seen := map[string]bool{}
	for _, item := range payload.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		models = append(models, id)
	}
	sort.Strings(models)
	return models, nil
}

func parseChannelIDs(c *gin.Context) ([]int, error) {
	var req struct {
		ChannelIDs []int `json:"channel_ids"`
		ChannelID  int   `json:"channel_id"`
		ID         int   `json:"id"`
	}
	_ = c.ShouldBindJSON(&req)
	ids := make([]int, 0)
	for _, id := range req.ChannelIDs {
		if id > 0 {
			ids = append(ids, id)
		}
	}
	if req.ChannelID > 0 {
		ids = append(ids, req.ChannelID)
	}
	if req.ID > 0 {
		ids = append(ids, req.ID)
	}
	uniq := map[int]bool{}
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if !uniq[id] {
			uniq[id] = true
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("缺少 channel_ids")
	}
	return out, nil
}

// FetchUpstreamModels 对齐 new-api: GET /api/channel/fetch_models/:id
func FetchUpstreamModels(c *gin.Context) {
	FetchChannelModels(c)
}

// FetchModels 对齐 new-api: POST /api/channel/fetch_models
// 兼容两种请求：
// 1) 传 channel_ids/channel_id/id（按已存在渠道 ID 拉取）
// 2) 直接传新增表单字段 type/key/base_url/proxy（未保存渠道预拉取）
func FetchModels(c *gin.Context) {
	var req struct {
		ChannelIDs []int  `json:"channel_ids"`
		ChannelID  int    `json:"channel_id"`
		ID         int    `json:"id"`
		Type       int    `json:"type"`
		Key        string `json:"key"`
		BaseURL    string `json:"base_url"`
		Proxy      string `json:"proxy"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "请求参数格式错误")
		return
	}

	// 优先按 payload 直拉（新增渠道场景）
	if strings.TrimSpace(req.Key) != "" || strings.TrimSpace(req.BaseURL) != "" || strings.TrimSpace(req.Proxy) != "" {
		ch := model.Channel{
			Type:    req.Type,
			Key:     req.Key,
			BaseURL: req.BaseURL,
			Proxy:   req.Proxy,
		}
		models, fetchErr := fetchUpstreamModelList(ch)
		if fetchErr != nil {
			common.Fail(c, common.CodeServerError, fetchErr.Error())
			return
		}
		common.OK(c, gin.H{"models": models})
		return
	}

	ids := make([]int, 0)
	for _, id := range req.ChannelIDs {
		if id > 0 {
			ids = append(ids, id)
		}
	}
	if req.ChannelID > 0 {
		ids = append(ids, req.ChannelID)
	}
	if req.ID > 0 {
		ids = append(ids, req.ID)
	}
	uniq := map[int]bool{}
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if !uniq[id] {
			uniq[id] = true
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		common.Fail(c, common.CodeParamError, "缺少 channel_ids 或可用于拉取模型的渠道参数")
		return
	}

	var channels []model.Channel
	if err := common.DB.Where("id IN ?", out).Find(&channels).Error; err != nil {
		common.Fail(c, common.CodeServerError, "查询渠道失败")
		return
	}

	results := make(map[string]interface{})
	for _, ch := range channels {
		models, fetchErr := fetchUpstreamModelList(ch)
		if fetchErr != nil {
			results[strconv.Itoa(ch.Id)] = gin.H{"error": fetchErr.Error()}
			continue
		}
		results[strconv.Itoa(ch.Id)] = gin.H{"models": models}
	}
	common.OK(c, results)
}

func detectUpstreamUpdatesByIDs(ids []int) gin.H {
	var channels []model.Channel
	_ = common.DB.Where("id IN ?", ids).Find(&channels).Error

	items := make([]gin.H, 0, len(channels))
	for _, ch := range channels {
		upstreamModels, err := fetchUpstreamModelList(ch)
		if err != nil {
			items = append(items, gin.H{"channel_id": ch.Id, "channel_name": ch.Name, "error": err.Error()})
			continue
		}
		localSet := map[string]bool{}
		for _, m := range strings.Split(ch.Models, ",") {
			m = strings.TrimSpace(m)
			if m != "" {
				localSet[m] = true
			}
		}
		upSet := map[string]bool{}
		for _, m := range upstreamModels {
			upSet[m] = true
		}

		added := make([]string, 0)
		removed := make([]string, 0)
		for m := range upSet {
			if !localSet[m] {
				added = append(added, m)
			}
		}
		for m := range localSet {
			if !upSet[m] {
				removed = append(removed, m)
			}
		}
		sort.Strings(added)
		sort.Strings(removed)

		items = append(items, gin.H{
			"channel_id":   ch.Id,
			"channel_name": ch.Name,
			"has_updates":  len(added) > 0 || len(removed) > 0,
			"added":        added,
			"removed":      removed,
		})
	}
	return gin.H{"items": items, "count": len(items)}
}

// DetectChannelUpstreamModelUpdates 对齐 new-api: POST /api/channel/upstream_updates/detect
func DetectChannelUpstreamModelUpdates(c *gin.Context) {
	ids, err := parseChannelIDs(c)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	common.OK(c, detectUpstreamUpdatesByIDs(ids))
}

// DetectAllChannelUpstreamModelUpdates 对齐 new-api: POST /api/channel/upstream_updates/detect_all
func DetectAllChannelUpstreamModelUpdates(c *gin.Context) {
	var ids []int
	_ = common.DB.Model(&model.Channel{}).Pluck("id", &ids).Error
	common.OK(c, detectUpstreamUpdatesByIDs(ids))
}

func applyUpstreamUpdatesByIDs(ids []int) gin.H {
	var channels []model.Channel
	_ = common.DB.Where("id IN ?", ids).Find(&channels).Error

	updated := 0
	items := make([]gin.H, 0, len(channels))
	for _, ch := range channels {
		upstreamModels, err := fetchUpstreamModelList(ch)
		if err != nil {
			items = append(items, gin.H{"channel_id": ch.Id, "channel_name": ch.Name, "error": err.Error()})
			continue
		}
		joined := strings.Join(upstreamModels, ",")
		if strings.TrimSpace(joined) == strings.TrimSpace(ch.Models) {
			items = append(items, gin.H{"channel_id": ch.Id, "channel_name": ch.Name, "updated": false})
			continue
		}
		if err = common.DB.Model(&model.Channel{}).Where("id = ?", ch.Id).Update("models", joined).Error; err != nil {
			items = append(items, gin.H{"channel_id": ch.Id, "channel_name": ch.Name, "error": err.Error()})
			continue
		}
		updated++
		items = append(items, gin.H{"channel_id": ch.Id, "channel_name": ch.Name, "updated": true, "model_count": len(upstreamModels)})
	}
	return gin.H{"updated": updated, "items": items}
}

// ApplyChannelUpstreamModelUpdates 对齐 new-api: POST /api/channel/upstream_updates/apply
func ApplyChannelUpstreamModelUpdates(c *gin.Context) {
	ids, err := parseChannelIDs(c)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	common.OK(c, applyUpstreamUpdatesByIDs(ids))
}

// ApplyAllChannelUpstreamModelUpdates 对齐 new-api: POST /api/channel/upstream_updates/apply_all
func ApplyAllChannelUpstreamModelUpdates(c *gin.Context) {
	var ids []int
	_ = common.DB.Model(&model.Channel{}).Pluck("id", &ids).Error
	common.OK(c, applyUpstreamUpdatesByIDs(ids))
}

func collectChannelModelNames(channels []model.Channel) []string {
	seen := make(map[string]bool)
	models := make([]string, 0)
	for _, channel := range channels {
		for _, name := range strings.Split(channel.Models, ",") {
			modelName := strings.TrimSpace(name)
			if modelName == "" || seen[modelName] {
				continue
			}
			seen[modelName] = true
			models = append(models, modelName)
		}
	}
	sort.Strings(models)
	return models
}

func queryExistingModelMetaNames(modelNames []string) map[string]bool {
	out := make(map[string]bool)
	if len(modelNames) == 0 {
		return out
	}
	var existing []string
	if err := common.DB.Model(&model.ModelMeta{}).Where("model_name IN ?", modelNames).Pluck("model_name", &existing).Error; err != nil {
		return out
	}
	for _, name := range existing {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" {
			out[trimmed] = true
		}
	}
	return out
}

func getActiveChannelsByIDs(ids []int) ([]model.Channel, error) {
	var channels []model.Channel
	db := common.DB.Model(&model.Channel{}).Where("status = ?", model.ChannelStatusActive)
	if len(ids) > 0 {
		db = db.Where("id IN ?", ids)
	}
	err := db.Find(&channels).Error
	return channels, err
}

func parseOptionalChannelIDs(c *gin.Context) []int {
	idsMap := make(map[int]bool)

	appendID := func(id int) {
		if id > 0 {
			idsMap[id] = true
		}
	}

	if raw := strings.TrimSpace(c.Query("channel_id")); raw != "" {
		if id, err := strconv.Atoi(raw); err == nil {
			appendID(id)
		}
	}
	if raw := strings.TrimSpace(c.Query("id")); raw != "" {
		if id, err := strconv.Atoi(raw); err == nil {
			appendID(id)
		}
	}
	if raw := strings.TrimSpace(c.Query("channel_ids")); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			if id, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
				appendID(id)
			}
		}
	}

	var req struct {
		ChannelIDs []int `json:"channel_ids"`
		ChannelID  int   `json:"channel_id"`
		ID         int   `json:"id"`
	}
	if c.Request != nil && c.Request.Body != nil {
		_ = c.ShouldBindJSON(&req)
		for _, id := range req.ChannelIDs {
			appendID(id)
		}
		appendID(req.ChannelID)
		appendID(req.ID)
	}

	ids := make([]int, 0, len(idsMap))
	for id := range idsMap {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// GetMissingModels 对齐 new-api: GET /api/models/missing
// 语义：返回“在启用渠道中出现，但尚未配置到 models meta 表”的模型名。
func GetMissingModels(c *gin.Context) {
	ids := parseOptionalChannelIDs(c)
	channels, err := getActiveChannelsByIDs(ids)
	if err != nil {
		common.Fail(c, common.CodeServerError, "读取渠道失败")
		return
	}

	referenced := collectChannelModelNames(channels)
	existingSet := queryExistingModelMetaNames(referenced)

	missing := make([]string, 0)
	for _, name := range referenced {
		if !existingSet[name] {
			missing = append(missing, name)
		}
	}

	common.OK(c, gin.H{
		"missing":       missing,
		"missing_count": len(missing),
		"channel_count": len(channels),
	})
}

// SyncUpstreamPreview 对齐 new-api: GET /api/models/sync_upstream/preview
// 语义：从渠道实时拉取 /v1/models，预览缺失模型（仅预览不写库）。
func SyncUpstreamPreview(c *gin.Context) {
	ids := parseOptionalChannelIDs(c)
	channels, err := getActiveChannelsByIDs(ids)
	if err != nil {
		common.Fail(c, common.CodeServerError, "读取渠道失败")
		return
	}
	if len(channels) == 0 {
		common.OK(c, gin.H{"missing": []string{}, "missing_count": 0, "channel_count": 0, "upstream_count": 0, "failed_channels": []gin.H{}})
		return
	}

	upstreamSet := make(map[string]bool)
	failedChannels := make([]gin.H, 0)
	for _, channel := range channels {
		models, fetchErr := fetchUpstreamModelList(channel)
		if fetchErr != nil {
			failedChannels = append(failedChannels, gin.H{"channel_id": channel.Id, "channel_name": channel.Name, "error": fetchErr.Error()})
			continue
		}
		for _, name := range models {
			trimmed := strings.TrimSpace(name)
			if trimmed != "" {
				upstreamSet[trimmed] = true
			}
		}
	}

	upstreamModels := make([]string, 0, len(upstreamSet))
	for name := range upstreamSet {
		upstreamModels = append(upstreamModels, name)
	}
	sort.Strings(upstreamModels)

	existingSet := queryExistingModelMetaNames(upstreamModels)
	missing := make([]string, 0)
	for _, name := range upstreamModels {
		if !existingSet[name] {
			missing = append(missing, name)
		}
	}

	common.OK(c, gin.H{
		"missing":         missing,
		"missing_count":   len(missing),
		"upstream_count":  len(upstreamModels),
		"channel_count":   len(channels),
		"failed_channels": failedChannels,
	})
}

// SyncUpstreamModels 对齐 new-api: POST /api/models/sync_upstream
// 语义：基于实时 upstream 模型列表，补齐缺失的 models meta 记录。
func SyncUpstreamModels(c *gin.Context) {
	var req struct {
		ChannelIDs []int `json:"channel_ids"`
		ChannelID  int   `json:"channel_id"`
		ID         int   `json:"id"`
		Enabled    *bool `json:"enabled"`
	}
	_ = c.ShouldBindJSON(&req)

	idsMap := make(map[int]bool)
	for _, id := range req.ChannelIDs {
		if id > 0 {
			idsMap[id] = true
		}
	}
	if req.ChannelID > 0 {
		idsMap[req.ChannelID] = true
	}
	if req.ID > 0 {
		idsMap[req.ID] = true
	}
	ids := make([]int, 0, len(idsMap))
	for id := range idsMap {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	channels, err := getActiveChannelsByIDs(ids)
	if err != nil {
		common.Fail(c, common.CodeServerError, "读取渠道失败")
		return
	}
	if len(channels) == 0 {
		common.OK(c, gin.H{"created": 0, "skipped": 0, "created_models": []string{}, "skipped_models": []string{}})
		return
	}

	upstreamSet := make(map[string]bool)
	failedChannels := make([]gin.H, 0)
	for _, channel := range channels {
		models, fetchErr := fetchUpstreamModelList(channel)
		if fetchErr != nil {
			failedChannels = append(failedChannels, gin.H{"channel_id": channel.Id, "channel_name": channel.Name, "error": fetchErr.Error()})
			continue
		}
		for _, name := range models {
			trimmed := strings.TrimSpace(name)
			if trimmed != "" {
				upstreamSet[trimmed] = true
			}
		}
	}

	upstreamModels := make([]string, 0, len(upstreamSet))
	for name := range upstreamSet {
		upstreamModels = append(upstreamModels, name)
	}
	sort.Strings(upstreamModels)

	existingSet := queryExistingModelMetaNames(upstreamModels)
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	createdModels := make([]string, 0)
	skippedModels := make([]string, 0)
	for _, name := range upstreamModels {
		if existingSet[name] {
			skippedModels = append(skippedModels, name)
			continue
		}
		item := &model.ModelMeta{
			VendorId:    0,
			ModelName:   name,
			DisplayName: name,
			ModelType:   "text",
			Enabled:     enabled,
		}
		if insertErr := item.Insert(); insertErr != nil {
			skippedModels = append(skippedModels, name)
			continue
		}
		createdModels = append(createdModels, name)
		existingSet[name] = true
	}

	common.OK(c, gin.H{
		"created":         len(createdModels),
		"skipped":         len(skippedModels),
		"created_models":  createdModels,
		"skipped_models":  skippedModels,
		"upstream_count":  len(upstreamModels),
		"channel_count":   len(channels),
		"failed_channels": failedChannels,
	})
}
