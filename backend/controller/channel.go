package controller

import (
	"STfreApi/adapter"
	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"
)

func fetchAndPersistChannelBalance(channel *model.Channel) (float64, error) {
	if channel.AccessToken != "" && adapter.SupportsAccountFeatures(channel.Type) {
		balanceInfo, err := service.RefreshChannelBalance(channel)
		if err == nil && balanceInfo != nil {
			return balanceInfo.Balance, nil
		}
	}

	baseUrl := strings.TrimSuffix(channel.BaseURL, "/")
	key := service.GetNextKey(channel.Key)
	client := common.NewHTTPClient(channel.Proxy)

	var balance float64
	switch channel.Type {
	case model.ChannelTypeSiliconFlow:
		if baseUrl == "" {
			baseUrl = "https://api.siliconflow.cn"
		}
		req, _ := http.NewRequest("GET", baseUrl+"/v1/user/info", nil)
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := client.Do(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()
		var result struct {
			Data struct {
				Balance string `json:"balance"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			fmt.Sscanf(result.Data.Balance, "%f", &balance)
		}
	default:
		if baseUrl == "" {
			baseUrl = "https://api.openai.com"
		}
		req, _ := http.NewRequest("GET", baseUrl+"/v1/dashboard/billing/credit_grants", nil)
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := client.Do(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()
		var result struct {
			TotalAvailable float64 `json:"total_available"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			balance = result.TotalAvailable
		}
	}

	if err := common.DB.Model(channel).Update("balance", balance).Error; err != nil {
		return 0, err
	}
	return balance, nil
}

func FetchChannelBalance(c *gin.Context) {
	id := c.Param("id")
	var channel model.Channel
	if err := common.DB.First(&channel, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}
	balance, err := fetchAndPersistChannelBalance(&channel)
	if err != nil {
		common.Fail(c, common.CodeServerError, "查询余额失败: "+err.Error())
		return
	}
	common.OK(c, gin.H{"balance": balance})
}

func SearchChannels(c *gin.Context) {
	keyword := c.Query("keyword")
	var channels []model.Channel
	db := common.DB.Order("priority desc")
	if keyword != "" {
		db = db.Where("name LIKE ? OR base_url LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if err := db.Find(&channels).Error; err != nil {
		common.Fail(c, common.CodeServerError, "搜索失败")
		return
	}
	common.OK(c, channels)
}

func GetAllChannels(c *gin.Context) {
	var channels []model.Channel
	db := common.DB.Order("priority desc")
	if tag := c.Query("tag"); tag != "" {
		db = db.Where("tags LIKE ?", "%"+tag+"%")
	}
	if err := db.Find(&channels).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取渠道失败")
		return
	}
	common.OK(c, channels)
}

func ChannelListModels(c *gin.Context) {
	var channels []model.Channel
	if err := common.DB.Find(&channels).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取模型列表失败")
		return
	}
	set := make(map[string]struct{})
	for _, ch := range channels {
		for _, m := range strings.Split(ch.Models, ",") {
			m = strings.TrimSpace(m)
			if m != "" {
				set[m] = struct{}{}
			}
		}
	}
	models := make([]string, 0, len(set))
	for m := range set {
		models = append(models, m)
	}
	sort.Strings(models)
	common.OK(c, models)
}

func EnabledListModels(c *gin.Context) {
	var channels []model.Channel
	if err := common.DB.Where("status = ?", model.ChannelStatusActive).Find(&channels).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取模型列表失败")
		return
	}
	set := make(map[string]struct{})
	for _, ch := range channels {
		for _, m := range strings.Split(ch.Models, ",") {
			m = strings.TrimSpace(m)
			if m != "" {
				set[m] = struct{}{}
			}
		}
	}
	models := make([]string, 0, len(set))
	for m := range set {
		models = append(models, m)
	}
	sort.Strings(models)
	common.OK(c, models)
}

func GetChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "渠道 ID 非法")
		return
	}
	var channel model.Channel
	if err := common.DB.First(&channel, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}
	common.OK(c, channel)
}

func GetChannelKey(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "渠道 ID 非法")
		return
	}
	var channel model.Channel
	if err := common.DB.Select("id", "key").First(&channel, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}
	common.OK(c, gin.H{"key": channel.Key})
}

func UpdateAllChannelsBalance(c *gin.Context) {
	var channels []model.Channel
	if err := common.DB.Where("status = ?", model.ChannelStatusActive).Find(&channels).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取渠道失败")
		return
	}
	successCount := 0
	failCount := 0
	for i := range channels {
		if _, err := fetchAndPersistChannelBalance(&channels[i]); err != nil {
			failCount++
			continue
		}
		successCount++
	}
	common.OK(c, gin.H{"success_count": successCount, "fail_count": failCount})
}

func UpdateChannelBalance(c *gin.Context) {
	FetchChannelBalance(c)
}

func syncModelConfigsFromChannelModels(modelsText string) error {
	items := strings.Split(modelsText, ",")
	toCreate := make([]model.ModelConfig, 0, len(items))
	seen := make(map[string]bool)
	for _, item := range items {
		name := strings.TrimSpace(item)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		toCreate = append(toCreate, model.ModelConfig{
			ModelName:   name,
			DisplayName: name,
			Category:    "chat",
			InputRatio:  1,
			OutputRatio: 1,
			MaxContext:  4096,
			Enabled:     true,
			CreatedAt:   common.GetTimestamp(),
		})
	}
	if len(toCreate) == 0 {
		return nil
	}
	return common.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&toCreate).Error
}

// 🌸 syncModelMetaFromChannelModels ～把渠道里配置的模型顺手补进「模型元数据(模型广场)」～
// 新增/编辑渠道并设置好模型后，这些模型自动出现在模型广场，不用再手动一个个加啦(๑˃̵ᴗ˂̵)
// 只补「还不存在」的模型名，已有的原样不动——绝不覆盖主人辛苦调好的元数据喵～
// 🎀 升级：新建时顺便从官方上游匹配端点/供应商/标签/图标/描述，上下文默认 200K(可手动改)～
func syncModelMetaFromChannelModels(modelsText string) error {
	// 拆出去重后的模型名列表
	names := make([]string, 0)
	seen := make(map[string]bool)
	for _, item := range strings.Split(modelsText, ",") {
		name := strings.TrimSpace(item)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	if len(names) == 0 {
		return nil
	}

	// 查出已存在的模型名，剩下的才需要新建～
	var existing []string
	if err := common.DB.Model(&model.ModelMeta{}).
		Where("model_name IN ?", names).
		Pluck("model_name", &existing).Error; err != nil {
		return err
	}
	existSet := make(map[string]bool, len(existing))
	for _, n := range existing {
		existSet[strings.TrimSpace(n)] = true
	}

	// 🎀 拉一份官方元数据用于匹配端点/供应商等；拉不到也没关系，用默认值兜底～
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	modelByName, vendorByName, _, _, metaErr := fetchOfficialMeta(ctx, "")
	if metaErr != nil {
		modelByName = map[string]officialModel{}
		vendorByName = map[string]officialVendor{}
	}
	vendorIDCache := make(map[string]int)
	createdVendors := 0

	const defaultContextLength = 200000 // 🌸 上下文默认 200K，用户可在元数据编辑里改～

	now := time.Now()
	toCreate := make([]model.ModelMeta, 0)
	for _, name := range names {
		if existSet[name] {
			continue
		}
		meta := model.ModelMeta{
			ModelName:     name,
			DisplayName:   name,
			ModelType:     "text",
			ContextLength: defaultContextLength,
			Enabled:       true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		// 官方有这个模型 → 带出端点/标签/图标/描述/供应商～
		if up, ok := modelByName[name]; ok {
			meta.Endpoints = officialEndpointsStr(up.Endpoints)
			meta.Tags = up.Tags
			meta.Icon = up.Icon
			meta.Description = up.Description
			meta.NameRule = up.NameRule
			if up.VendorName != "" {
				meta.VendorId = ensureOfficialVendor(up.VendorName, vendorByName, vendorIDCache, &createdVendors)
			}
		}
		toCreate = append(toCreate, meta)
	}
	if len(toCreate) == 0 {
		return nil
	}
	return common.DB.Create(&toCreate).Error
}

func AddChannel(c *gin.Context) {
	var channel model.Channel
	if err := c.ShouldBindJSON(&channel); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if channel.Name == "" || channel.Key == "" {
		common.Fail(c, common.CodeParamError, "名称和密钥不能为空")
		return
	}
	if err := common.DB.Create(&channel).Error; err != nil {
		common.Fail(c, common.CodeServerError, "创建渠道失败")
		return
	}
	if err := channel.AddAbilities(nil); err != nil {
		common.Fail(c, common.CodeServerError, "渠道已创建，但同步能力表失败")
		return
	}
	if err := syncModelConfigsFromChannelModels(channel.Models); err != nil {
		common.Fail(c, common.CodeServerError, "渠道已创建，但同步模型管理失败")
		return
	}
	// 🌸 顺手把模型补进模型广场(含官方端点匹配，可能较慢)～后台异步做，不阻塞创建响应
	channelModels := channel.Models
	go func() {
		if err := syncModelMetaFromChannelModels(channelModels); err != nil {
			log.Printf("[Channel] 渠道已创建，但同步模型元数据失败: %v", err)
		}
	}()
	common.OKMsg(c, "渠道创建成功", channel)
}

func AddChannels(c *gin.Context) {
	var channels []model.Channel
	if err := c.ShouldBindJSON(&channels); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if len(channels) == 0 {
		common.Fail(c, common.CodeParamError, "渠道列表不能为空")
		return
	}
	for _, ch := range channels {
		if ch.Name == "" || ch.Key == "" {
			common.Fail(c, common.CodeParamError, "所有渠道的名称和密钥不能为空")
			return
		}
	}
	if err := common.DB.Create(&channels).Error; err != nil {
		common.Fail(c, common.CodeServerError, "批量创建渠道失败")
		return
	}
	for _, channel := range channels {
		if err := channel.AddAbilities(nil); err != nil {
			common.Fail(c, common.CodeServerError, "批量创建成功，但同步能力表失败")
			return
		}
		if err := syncModelConfigsFromChannelModels(channel.Models); err != nil {
			common.Fail(c, common.CodeServerError, "批量创建成功，但同步模型管理失败")
			return
		}
		// 🌸 批量创建也顺手补进模型广场～后台异步补齐(含官方端点匹配)
		cm := channel.Models
		go func() {
			if err := syncModelMetaFromChannelModels(cm); err != nil {
				log.Printf("[Channel] 批量创建渠道后同步模型元数据失败: %v", err)
			}
		}()
	}
	common.OKMsg(c, "批量创建成功", gin.H{"count": len(channels)})
}

func DeleteChannel(c *gin.Context) {
	id := c.Param("id")
	channelId, _ := strconv.Atoi(id)
	if err := common.DB.Delete(&model.Channel{}, id).Error; err != nil {
		common.Fail(c, common.CodeServerError, "删除渠道失败")
		return
	}
	_ = model.DeleteAbilitiesByChannelId(channelId)
	common.OKMsg(c, "渠道已删除", nil)
}

func DeleteDisabledChannel(c *gin.Context) {
	var disabledIds []int
	common.DB.Model(&model.Channel{}).Where("status IN ?", []int{model.ChannelStatusDisabled, model.ChannelStatusAutoDisabled}).Pluck("id", &disabledIds)
	res := common.DB.Where("status IN ?", []int{model.ChannelStatusDisabled, model.ChannelStatusAutoDisabled}).Delete(&model.Channel{})
	if res.Error != nil {
		common.Fail(c, common.CodeServerError, "删除禁用渠道失败")
		return
	}
	if len(disabledIds) > 0 {
		common.DB.Where("channel_id IN ?", disabledIds).Delete(&model.Ability{})
	}
	common.OK(c, gin.H{"deleted": res.RowsAffected})
}

type channelTagReq struct {
	Tag          string  `json:"tag"`
	NewTag       *string `json:"new_tag"`
	Priority     *int64  `json:"priority"`
	Weight       *uint   `json:"weight"`
	ModelMapping *string `json:"model_mapping"`
	Models       *string `json:"models"`
}

func DisableTagChannels(c *gin.Context) {
	var req channelTagReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Tag) == "" {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	tagFilter := "%" + strings.TrimSpace(req.Tag) + "%"
	var ids []int
	common.DB.Model(&model.Channel{}).Where("tags LIKE ?", tagFilter).Pluck("id", &ids)
	if err := common.DB.Model(&model.Channel{}).Where("tags LIKE ?", tagFilter).Update("status", model.ChannelStatusDisabled).Error; err != nil {
		common.Fail(c, common.CodeServerError, "批量禁用失败")
		return
	}
	if len(ids) > 0 {
		common.DB.Model(&model.Ability{}).Where("channel_id IN ?", ids).Update("enabled", false)
	}
	common.OK(c, nil)
}

func EnableTagChannels(c *gin.Context) {
	var req channelTagReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Tag) == "" {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	tagFilter := "%" + strings.TrimSpace(req.Tag) + "%"
	var ids []int
	common.DB.Model(&model.Channel{}).Where("tags LIKE ?", tagFilter).Pluck("id", &ids)
	if err := common.DB.Model(&model.Channel{}).Where("tags LIKE ?", tagFilter).Update("status", model.ChannelStatusActive).Error; err != nil {
		common.Fail(c, common.CodeServerError, "批量启用失败")
		return
	}
	if len(ids) > 0 {
		common.DB.Model(&model.Ability{}).Where("channel_id IN ?", ids).Update("enabled", true)
	}
	common.OK(c, nil)
}

func EditTagChannels(c *gin.Context) {
	var req channelTagReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Tag) == "" {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	updates := map[string]interface{}{}
	if req.NewTag != nil {
		updates["tags"] = strings.TrimSpace(*req.NewTag)
	}
	if req.Priority != nil {
		updates["priority"] = int(*req.Priority)
	}
	if req.Weight != nil {
		updates["weight"] = int(*req.Weight)
	}
	if req.ModelMapping != nil {
		updates["model_mapping"] = strings.TrimSpace(*req.ModelMapping)
	}
	if req.Models != nil {
		updates["models"] = strings.TrimSpace(*req.Models)
	}
	if len(updates) == 0 {
		common.Fail(c, common.CodeParamError, "无可更新字段")
		return
	}
	tagFilter := "%" + strings.TrimSpace(req.Tag) + "%"
	if err := common.DB.Model(&model.Channel{}).Where("tags LIKE ?", tagFilter).Updates(updates).Error; err != nil {
		common.Fail(c, common.CodeServerError, "批量编辑失败")
		return
	}
	// priority/weight/models/model_mapping 任一变了，能力表都得跟着重新长一遍～
	var affected []model.Channel
	common.DB.Where("tags LIKE ?", tagFilter).Find(&affected)
	for i := range affected {
		_ = affected[i].UpdateAbilities(nil)
	}
	common.OK(c, nil)
}

func FixChannelsAbilities(c *gin.Context) {
	success, fails := model.FixAllAbilities()
	common.OK(c, gin.H{"success": success, "fails": fails})
}

func UpdateChannel(c *gin.Context) {
	var channel model.Channel
	if err := c.ShouldBindJSON(&channel); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if channel.Id == 0 {
		common.Fail(c, common.CodeParamError, "缺少渠道 ID")
		return
	}
	if err := common.DB.Model(&channel).Updates(channel).Error; err != nil {
		common.Fail(c, common.CodeServerError, "更新渠道失败")
		return
	}
	// GORM 的 Updates(struct) 会跳过零值字段，这里的 channel 变量并不完整，
	// 要重新查一遍完整数据才能安心地拿去重建能力表～
	var fresh model.Channel
	if err := common.DB.First(&fresh, channel.Id).Error; err == nil {
		if err := fresh.UpdateAbilities(nil); err != nil {
			common.Fail(c, common.CodeServerError, "渠道已更新，但同步能力表失败")
			return
		}
	}
	if err := syncModelConfigsFromChannelModels(channel.Models); err != nil {
		common.Fail(c, common.CodeServerError, "渠道已更新，但同步模型管理失败")
		return
	}
	// 🌸 编辑渠道后同样把模型补进模型广场～用 fresh 完整数据里的 models 更稳；后台异步补齐
	freshModels := fresh.Models
	go func() {
		if err := syncModelMetaFromChannelModels(freshModels); err != nil {
			log.Printf("[Channel] 渠道已更新，但同步模型元数据失败: %v", err)
		}
	}()
	common.OKMsg(c, "渠道更新成功", nil)
}

func ToggleChannelStatus(c *gin.Context) {
	id := c.Param("id")
	var channel model.Channel
	if err := common.DB.First(&channel, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}
	newStatus := model.ChannelStatusDisabled
	if channel.Status == model.ChannelStatusDisabled {
		newStatus = model.ChannelStatusActive
	}
	common.DB.Model(&channel).Update("status", newStatus)
	common.OK(c, gin.H{"status": newStatus})
}

func TestChannel(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
	}
	c.ShouldBindJSON(&req)

	var channel model.Channel
	if err := common.DB.First(&channel, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}
	responseTime, err := service.CheckChannelWithPrompt(&channel, req.Model, req.Prompt)
	if err != nil {
		common.OK(c, gin.H{"success": false, "msg": err.Error(), "time": 0})
		return
	}
	channel.ResponseTime = responseTime
	channel.TestTime = common.GetTimestamp()
	common.DB.Save(&channel)
	common.OK(c, gin.H{"success": true, "time": responseTime, "msg": "测试通过"})
}
