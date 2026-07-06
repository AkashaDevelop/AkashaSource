package controller

import (
	ollamaAdapter "STfreApi/adapter/ollama"
	"STfreApi/common"
	"STfreApi/model"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func resolveOllamaChannel(channelID int) (*model.Channel, string, string, error) {
	var ch model.Channel
	if err := common.DB.First(&ch, channelID).Error; err != nil {
		return nil, "", "", fmt.Errorf("渠道不存在")
	}
	if ch.Type != model.ChannelTypeOllama {
		return nil, "", "", fmt.Errorf("该操作仅支持 Ollama 渠道")
	}

	baseURL := strings.TrimSpace(ch.BaseURL)
	if baseURL == "" {
		baseURL = ollamaAdapter.BaseURL
	}

	key := ""
	keys := splitLinesNonEmpty(ch.Key)
	if len(keys) > 0 {
		key = keys[0]
	}
	return &ch, baseURL, key, nil
}

func OllamaPullModel(c *gin.Context) {
	var req struct {
		ChannelID int    `json:"channel_id"`
		ModelName string `json:"model_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "请求参数错误")
		return
	}
	if req.ChannelID == 0 || strings.TrimSpace(req.ModelName) == "" {
		common.Fail(c, common.CodeParamError, "channel_id 和 model_name 不能为空")
		return
	}

	_, baseURL, key, err := resolveOllamaChannel(req.ChannelID)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	if err := ollamaAdapter.PullOllamaModel(baseURL, key, strings.TrimSpace(req.ModelName)); err != nil {
		common.Fail(c, common.CodeServerError, fmt.Sprintf("拉取模型失败: %s", err.Error()))
		return
	}

	common.OKMsg(c, fmt.Sprintf("模型 %s 拉取成功", req.ModelName), nil)
}

func OllamaPullModelStream(c *gin.Context) {
	var req struct {
		ChannelID int    `json:"channel_id"`
		ModelName string `json:"model_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "请求参数错误")
		return
	}
	if req.ChannelID == 0 || strings.TrimSpace(req.ModelName) == "" {
		common.Fail(c, common.CodeParamError, "channel_id 和 model_name 不能为空")
		return
	}

	_, baseURL, key, err := resolveOllamaChannel(req.ChannelID)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	progressCallback := func(progress ollamaAdapter.OllamaPullResponse) {
		data, _ := json.Marshal(progress)
		_, _ = c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
		c.Writer.Flush()
	}

	err = ollamaAdapter.PullOllamaModelStream(baseURL, key, strings.TrimSpace(req.ModelName), progressCallback)
	if err != nil {
		errorData, _ := json.Marshal(gin.H{"error": err.Error()})
		_, _ = c.Writer.Write([]byte("data: " + string(errorData) + "\n\n"))
	} else {
		successData, _ := json.Marshal(gin.H{"message": fmt.Sprintf("模型 %s 拉取成功", req.ModelName)})
		_, _ = c.Writer.Write([]byte("data: " + string(successData) + "\n\n"))
	}
	_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
	c.Writer.Flush()
}

func OllamaDeleteModel(c *gin.Context) {
	var req struct {
		ChannelID int    `json:"channel_id"`
		ModelName string `json:"model_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "请求参数错误")
		return
	}
	if req.ChannelID == 0 || strings.TrimSpace(req.ModelName) == "" {
		common.Fail(c, common.CodeParamError, "channel_id 和 model_name 不能为空")
		return
	}

	_, baseURL, key, err := resolveOllamaChannel(req.ChannelID)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	if err := ollamaAdapter.DeleteOllamaModel(baseURL, key, strings.TrimSpace(req.ModelName)); err != nil {
		common.Fail(c, common.CodeServerError, fmt.Sprintf("删除模型失败: %s", err.Error()))
		return
	}

	common.OKMsg(c, fmt.Sprintf("模型 %s 删除成功", req.ModelName), nil)
}

func OllamaVersion(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "渠道 ID 非法")
		return
	}

	_, baseURL, key, err := resolveOllamaChannel(id)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	version, err := ollamaAdapter.FetchOllamaVersion(baseURL, key)
	if err != nil {
		common.Fail(c, common.CodeServerError, fmt.Sprintf("获取 Ollama 版本失败: %s", err.Error()))
		return
	}

	common.OK(c, gin.H{"success": true, "data": gin.H{"version": version}})
}

func MarshalJSONOrEmpty(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
