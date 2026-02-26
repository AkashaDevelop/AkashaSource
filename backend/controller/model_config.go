package controller

import (
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
	c.JSON(http.StatusOK, gin.H{"message": "创建成功", "data": config})
}

func UpdateModelConfig(c *gin.Context) {
	var config model.ModelConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if config.Id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID 必填"})
		return
	}
	if err := common.DB.Model(&config).Updates(config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

func DeleteModelConfig(c *gin.Context) {
	id := c.Param("id")
	if err := common.DB.Delete(&model.ModelConfig{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// SyncPricingFromModelConfig syncs model ratios from ModelConfig table to option map
func SyncPricingFromModelConfig(c *gin.Context) {
	var configs []model.ModelConfig
	if err := common.DB.Where("enabled = ?", true).Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取模型配置失败"})
		return
	}

	modelRatios := make(map[string]float64)
	completionRatios := make(map[string]float64)

	for _, cfg := range configs {
		modelRatios[cfg.ModelName] = cfg.InputRatio
		if cfg.OutputRatio > 0 {
			completionRatios[cfg.ModelName] = cfg.OutputRatio / cfg.InputRatio
		}
	}

	mrJSON, _ := common.MapToJSON(modelRatios)
	crJSON, _ := common.MapToJSON(completionRatios)

	// Save to DB
	common.DB.Model(&model.Option{}).Where("key = ?", "model_ratio").
		Update("value", mrJSON)
	common.DB.Model(&model.Option{}).Where("key = ?", "completion_ratio").
		Update("value", crJSON)

	// Update in-memory
	common.UpdateOptionMap("model_ratio", mrJSON)
	common.UpdateOptionMap("completion_ratio", crJSON)

	synced := strconv.Itoa(len(configs))
	c.JSON(http.StatusOK, gin.H{"message": "同步成功", "synced": synced})
}
