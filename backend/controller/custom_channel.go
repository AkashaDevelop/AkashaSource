package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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
	id := c.Param("id")
	var config model.CustomChannelConfig
	if err := common.DB.First(&config, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "配置不存在")
		return
	}

	// 允许请求体中临时覆盖 key 和 base_url（用于保存前预测试）
	var override struct {
		Key     string `json:"key"`
		BaseURL string `json:"base_url"`
		Model   string `json:"model"`
		Message string `json:"message"`
	}
	_ = c.ShouldBindJSON(&override)

	testKey := strings.TrimSpace(override.Key)
	testBaseURL := strings.TrimSpace(override.BaseURL)
	testModel := strings.TrimSpace(override.Model)
	testMessage := strings.TrimSpace(override.Message)
	if testMessage == "" {
		testMessage = "Hello, please reply with a single word 'OK'."
	}
	if testModel == "" {
		testModel = "test"
	}

	baseURL := testBaseURL
	if baseURL == "" {
		// 从已绑定的渠道取 base_url，取第一个
		var ch model.Channel
		if err := common.DB.Where("custom_config_id = ?", config.Id).First(&ch).Error; err == nil {
			baseURL = strings.TrimRight(strings.TrimSpace(ch.BaseURL), "/")
			if testKey == "" {
				testKey = ch.Key
			}
		}
	}
	if baseURL == "" {
		common.Fail(c, common.CodeParamError, "无法确定测试目标地址，请传入 base_url 或先绑定渠道")
		return
	}
	if testKey == "" {
		common.Fail(c, common.CodeParamError, "无法确定 API Key，请传入 key 或先绑定渠道")
		return
	}

	// 构造最小化请求体
	fieldModel := config.FieldModel
	if fieldModel == "" || fieldModel == "-" {
		fieldModel = "model"
	}
	fieldMessages := config.FieldMessages
	if fieldMessages == "" || fieldMessages == "-" {
		fieldMessages = "messages"
	}
	reqBody := map[string]any{
		fieldModel: testModel,
		fieldMessages: []map[string]string{
			{"role": "user", "content": testMessage},
		},
		"stream": false,
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		common.Fail(c, common.CodeServerError, "构建请求失败")
		return
	}

	endpoint := strings.TrimLeft(strings.TrimSpace(config.RequestEndpoint), "/")
	targetURL := fmt.Sprintf("%s/%s", baseURL, endpoint)
	method := config.RequestMethod
	if method == "" {
		method = "POST"
	}

	req, err := http.NewRequest(method, targetURL, bytes.NewBuffer(raw))
	if err != nil {
		common.Fail(c, common.CodeServerError, fmt.Sprintf("创建请求失败: %v", err))
		return
	}

	authName := config.AuthHeaderName
	if authName == "" {
		authName = "Authorization"
	}
	authValue := strings.ReplaceAll(config.AuthHeaderTemplate, "{key}", testKey)
	if authValue == "" {
		authValue = "Bearer " + testKey
	}
	req.Header.Set(authName, authValue)

	contentType := config.RequestContentType
	if contentType == "" {
		contentType = "application/json"
	}
	req.Header.Set("Content-Type", contentType)

	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		common.OK(c, gin.H{
			"success":    false,
			"latency_ms": elapsed,
			"error":      err.Error(),
			"target_url": targetURL,
		})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	common.OK(c, gin.H{
		"success":     success,
		"status_code": resp.StatusCode,
		"latency_ms":  elapsed,
		"response":    string(body),
		"target_url":  targetURL,
	})
}
