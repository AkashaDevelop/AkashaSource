package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetOptions(c *gin.Context) {
	var options []model.Option
	if err := common.DB.Find(&options).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch options"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": options})
}

func UpdateOption(c *gin.Context) {
	var options []model.Option
	if err := c.ShouldBindJSON(&options); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, option := range options {
		// Update DB
		if err := common.DB.Save(&option).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update option: " + option.Key})
			return
		}
		// Update Memory
		common.UpdateOptionMap(option.Key, option.Value)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Options updated successfully"})
}

// Check if system is initialized (has admin account)
func IsSystemInitialized(c *gin.Context) {
	var count int64
	common.DB.Model(&model.User{}).Where("role >= ?", model.RoleAdmin).Count(&count)

	options := make(map[string]string)
	common.OptionLock.RLock()
	options["system_name"] = common.OptionMap["system_name"]
	options["logo_url"] = common.OptionMap["logo_url"]
	options["notice"] = common.OptionMap["notice"]
	common.OptionLock.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"initialized": count > 0,
		"options":     options,
	})
}
