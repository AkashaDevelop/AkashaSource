package controller

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"STfreApi/common"
	"STfreApi/model"

	"github.com/gin-gonic/gin"
)

func GetAllModelConfigs(c *gin.Context) {
	var configs []model.ModelConfig
	db := common.DB
	if category := c.Query("category"); category != "" {
		db = db.Where("category = ?", category)
	}
	if err := db.Order("model_name asc").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取模型配置失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs})
}

func AddModelConfig(c *gin.Context) {
	var config model.ModelConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.CreatedAt = time.Now().Unix()
	if err := common.DB.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建模型配置失败"})
		return
	}
	if err := syncAllPricingFromModelConfig(); err != nil {
		log.Printf("[ModelConfig] 定价同步失败: %v", err)
	}
	c.JSON(http.StatusOK, gin.H{"message": "创建成功", "data": config})
}

// UpdateModelConfig ～只更新请求体里实际带上的字段，而不是整个结构体全字段覆盖～
// 这样"模型管理"页（只送结构字段）和"模型定价"页（只送定价字段）
// 编辑同一条记录时不会互相清空对方没碰过的字段喵
func UpdateModelConfig(c *gin.Context) {
	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	idVal, ok := raw["id"]
	idFloat, isFloat := idVal.(float64)
	if !ok || !isFloat || idFloat == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID 必填"})
		return
	}
	id := int(idFloat)
	delete(raw, "id") // id 是查询条件，不放进 SET 子句里

	if len(raw) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有可更新的字段"})
		return
	}

	if err := common.DB.Model(&model.ModelConfig{}).Where("id = ?", id).Updates(raw).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	if err := syncAllPricingFromModelConfig(); err != nil {
		log.Printf("[ModelConfig] 定价同步失败: %v", err)
	}
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

func DeleteModelConfig(c *gin.Context) {
	id := c.Param("id")
	if err := common.DB.Delete(&model.ModelConfig{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if err := syncAllPricingFromModelConfig(); err != nil {
		log.Printf("[ModelConfig] 定价同步失败: %v", err)
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// BatchUpdateModelRatio 批量更新模型倍率
func BatchUpdateModelRatio(c *gin.Context) {
	var req struct {
		IDs         []int   `json:"ids" binding:"required"`
		InputRatio  float64 `json:"input_ratio"`
		OutputRatio float64 `json:"output_ratio"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}

	updates := make(map[string]interface{})
	if req.InputRatio > 0 {
		updates["input_ratio"] = req.InputRatio
	}
	if req.OutputRatio > 0 {
		updates["output_ratio"] = req.OutputRatio
	}

	if len(updates) == 0 {
		common.Fail(c, common.CodeParamError, "请提供要更新的倍率")
		return
	}

	if err := common.DB.Model(&model.ModelConfig{}).Where("id IN ?", req.IDs).Updates(updates).Error; err != nil {
		common.Fail(c, common.CodeServerError, "批量更新失败")
		return
	}
	if err := syncAllPricingFromModelConfig(); err != nil {
		log.Printf("[ModelConfig] 定价同步失败: %v", err)
	}

	common.OK(c, gin.H{"updated": len(req.IDs)})
}

// BatchApplyModelPricing ～把某个模型的全部定价维度一键复制给其他勾选的模型～
// 只搬定价字段（倍率/缓存/图像/音频/按次计费），不动 model_name/category/max_context/enabled 这些结构字段
func BatchApplyModelPricing(c *gin.Context) {
	var req struct {
		SourceId  int   `json:"source_id" binding:"required"`
		TargetIds []int `json:"target_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}

	var source model.ModelConfig
	if err := common.DB.First(&source, req.SourceId).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "源模型不存在")
		return
	}

	updates := map[string]interface{}{
		"input_ratio":            source.InputRatio,
		"output_ratio":           source.OutputRatio,
		"cache_ratio":            source.CacheRatio,
		"image_ratio":            source.ImageRatio,
		"audio_ratio":            source.AudioRatio,
		"audio_completion_ratio": source.AudioCompletionRatio,
		"is_fixed_price":         source.IsFixedPrice,
		"fixed_price":            source.FixedPrice,
	}

	if err := common.DB.Model(&model.ModelConfig{}).Where("id IN ?", req.TargetIds).Updates(updates).Error; err != nil {
		common.Fail(c, common.CodeServerError, "批量应用失败")
		return
	}
	if err := syncAllPricingFromModelConfig(); err != nil {
		log.Printf("[ModelConfig] 定价同步失败: %v", err)
	}

	common.OK(c, gin.H{"applied": len(req.TargetIds)})
}

// SyncPricingFromModelConfig 手动触发一次全量定价同步（保留给前端"强制重新同步"按钮用）
func SyncPricingFromModelConfig(c *gin.Context) {
	if err := syncAllPricingFromModelConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "同步失败"})
		return
	}
	var count int64
	common.DB.Model(&model.ModelConfig{}).Where("enabled = ?", true).Count(&count)
	c.JSON(http.StatusOK, gin.H{"message": "同步成功", "synced": strconv.FormatInt(count, 10)})
}

// syncAllPricingFromModelConfig ～从 ModelConfig 全表重新生成所有定价维度的 Option JSON，
// 写库后立刻热更新内存态，这样管理员编辑完模型定价不用再手动点同步按钮啦～
func syncAllPricingFromModelConfig() error {
	var configs []model.ModelConfig
	if err := common.DB.Where("enabled = ?", true).Find(&configs).Error; err != nil {
		return err
	}

	modelRatios := make(map[string]float64)
	completionRatios := make(map[string]float64)
	cacheRatios := make(map[string]float64)
	imageRatios := make(map[string]float64)
	audioRatios := make(map[string]float64)
	audioCompletionRatios := make(map[string]float64)
	modelPrices := make(map[string]float64)

	for _, cfg := range configs {
		modelRatios[cfg.ModelName] = cfg.InputRatio
		if cfg.InputRatio > 0 {
			completionRatios[cfg.ModelName] = cfg.OutputRatio / cfg.InputRatio
		}
		cacheRatios[cfg.ModelName] = cfg.CacheRatio
		imageRatios[cfg.ModelName] = cfg.ImageRatio
		audioRatios[cfg.ModelName] = cfg.AudioRatio
		audioCompletionRatios[cfg.ModelName] = cfg.AudioCompletionRatio
		if cfg.IsFixedPrice {
			modelPrices[cfg.ModelName] = cfg.FixedPrice
		}
	}

	pairs := []struct {
		key string
		m   map[string]float64
	}{
		{model.OptionKeyModelRatio, modelRatios},
		{model.OptionKeyCompletionRatio, completionRatios},
		{model.OptionKeyCacheRatio, cacheRatios},
		{model.OptionKeyImageRatio, imageRatios},
		{model.OptionKeyAudioRatio, audioRatios},
		{model.OptionKeyAudioCompletionRatio, audioCompletionRatios},
		{model.OptionKeyModelPrice, modelPrices},
	}

	for _, p := range pairs {
		jsonStr, err := common.MapToJSON(p.m)
		if err != nil {
			continue
		}
		// upsert：这几个 key（尤其是新增的 image/audio/cache）在 Option 表里可能压根还没有行
		opt := model.Option{Key: p.key}
		common.DB.Where(model.Option{Key: p.key}).Assign(model.Option{Value: jsonStr}).FirstOrCreate(&opt)
		common.UpdateOptionMap(p.key, jsonStr)
	}

	return nil
}
