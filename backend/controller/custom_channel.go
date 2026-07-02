package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetAllCustomConfigs 获取所有自定义渠道配置
func GetAllCustomConfigs(c *gin.Context) {
	userId := c.GetInt("id")
	role := c.GetInt("role")

	var configs []model.CustomChannelConfig
	query := common.DB.Model(&model.CustomChannelConfig{})

	// 非 Root 用户只能看到自己创建的和公开的配置
	if role != model.RoleRoot {
		query = query.Where("creator_id = ? OR is_public = 1", userId)
	}

	if err := query.Order("id desc").Find(&configs).Error; err != nil {
		common.Fail(c, common.CodeServerError, "查询失败")
		return
	}

	common.OK(c, configs)
}

// GetCustomConfig 获取单个自定义配置
func GetCustomConfig(c *gin.Context) {
	id := c.Param("id")
	var config model.CustomChannelConfig

	if err := common.DB.First(&config, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "配置不存在")
		return
	}

	common.OK(c, config)
}

// CreateCustomConfig 创建自定义配置
func CreateCustomConfig(c *gin.Context) {
	var config model.CustomChannelConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	// 设置创建者
	config.CreatorId = c.GetInt("id")
	config.CreatedAt = time.Now().Unix()
	config.UpdatedAt = time.Now().Unix()

	// 默认值
	if config.AdapterType == "" {
		config.AdapterType = "openai_compatible"
	}
	if config.RequestMethod == "" {
		config.RequestMethod = "POST"
	}
	if config.RequestContentType == "" {
		config.RequestContentType = "application/json"
	}
	if config.AuthType == "" {
		config.AuthType = "bearer"
	}
	if config.AuthHeaderName == "" {
		config.AuthHeaderName = "Authorization"
	}
	if config.AuthHeaderTemplate == "" {
		config.AuthHeaderTemplate = "Bearer {key}"
	}
	if config.Timeout == 0 {
		config.Timeout = 120
	}
	if config.RetryCount == 0 {
		config.RetryCount = 3
	}
	if config.RetryInterval == 0 {
		config.RetryInterval = 1
	}

	if err := common.DB.Create(&config).Error; err != nil {
		common.Fail(c, common.CodeServerError, "创建失败")
		return
	}

	common.OKMsg(c, "创建成功", config)
}

// UpdateCustomConfig 更新自定义配置
func UpdateCustomConfig(c *gin.Context) {
	var config model.CustomChannelConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	// 检查权限
	var existing model.CustomChannelConfig
	if err := common.DB.First(&existing, config.Id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "配置不存在")
		return
	}

	userId := c.GetInt("id")
	role := c.GetInt("role")
	if role != model.RoleRoot && existing.CreatorId != userId {
		common.Fail(c, common.CodeForbidden, "无权限修改此配置")
		return
	}

	config.UpdatedAt = time.Now().Unix()
	if err := common.DB.Model(&model.CustomChannelConfig{}).Where("id = ?", config.Id).Updates(&config).Error; err != nil {
		common.Fail(c, common.CodeServerError, "更新失败")
		return
	}

	common.OKMsg(c, "更新成功", config)
}

// DeleteCustomConfig 删除自定义配置
func DeleteCustomConfig(c *gin.Context) {
	id := c.Param("id")
	idInt, _ := strconv.Atoi(id)

	// 检查是否有渠道在使用
	var count int64
	common.DB.Model(&model.Channel{}).Where("custom_config_id = ?", idInt).Count(&count)
	if count > 0 {
		common.Failf(c, common.CodeConflict, "有 %d 个渠道正在使用此配置，无法删除", count)
		return
	}

	// 检查权限
	var config model.CustomChannelConfig
	if err := common.DB.First(&config, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "配置不存在")
		return
	}

	userId := c.GetInt("id")
	role := c.GetInt("role")
	if role != model.RoleRoot && config.CreatorId != userId {
		common.Fail(c, common.CodeForbidden, "无权限删除此配置")
		return
	}

	if err := common.DB.Delete(&model.CustomChannelConfig{}, id).Error; err != nil {
		common.Fail(c, common.CodeServerError, "删除失败")
		return
	}

	common.OKMsg(c, "删除成功", nil)
}

// TestCustomConfig 测试自定义配置
func TestCustomConfig(c *gin.Context) {
	// TODO: 实现测试逻辑
	common.OKMsg(c, "测试功能开发中", nil)
}
