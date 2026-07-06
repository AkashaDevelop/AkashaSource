package custom

import (
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
	Config           *model.CustomChannelConfig
	promptTokenCount int // ConvertRequest 时预先计算，供 stream 响应使用
}

func (a *Adaptor) GetChannelName() string {
	return "custom_" + a.Config.Name
}

func (a *Adaptor) GetModelList() []string {
	return []string{} // 自定义渠道由用户自行配置模型列表
}

func (a *Adaptor) ConvertRequest(c *gin.Context, request *dto.OpenAIRequest) (any, error) {
	// 预先统计 prompt tokens，供流式响应使用
	if raw, err := json.Marshal(request.Messages); err == nil {
		a.promptTokenCount = common.CountTokenForModel(string(raw), request.Model)
	}

	result := make(map[string]any)

	// 基础字段映射
	if a.Config.FieldModel != "" && a.Config.FieldModel != "-" {
		result[a.Config.FieldModel] = request.Model
	}
	if a.Config.FieldMessages != "" && a.Config.FieldMessages != "-" {
		result[a.Config.FieldMessages] = request.Messages
	}
	if a.Config.FieldTemperature != "" && a.Config.FieldTemperature != "-" && request.Temperature != 0 {
		result[a.Config.FieldTemperature] = request.Temperature
	}
	if a.Config.FieldMaxTokens != "" && a.Config.FieldMaxTokens != "-" && request.MaxTokens != 0 {
		result[a.Config.FieldMaxTokens] = request.MaxTokens
	}
	if a.Config.FieldStream != "" && a.Config.FieldStream != "-" {
		result[a.Config.FieldStream] = request.Stream
	}
	if a.Config.FieldTopP != "" && a.Config.FieldTopP != "-" && request.TopP != 0 {
		result[a.Config.FieldTopP] = request.TopP
	}
	if a.Config.FieldStop != "" && a.Config.FieldStop != "-" && request.Stop != nil {
		result[a.Config.FieldStop] = request.Stop
	}

	return result, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, channel *model.Channel, request any) (*http.Response, error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	// 构建完整 URL
	baseUrl := strings.TrimSuffix(channel.BaseURL, "/")
	endpoint := strings.TrimPrefix(a.Config.RequestEndpoint, "/")
	targetURL := fmt.Sprintf("%s/%s", baseUrl, endpoint)

	// 创建 HTTP 请求
	req, err := http.NewRequest(a.Config.RequestMethod, targetURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	// 设置认证头
	authHeader := strings.ReplaceAll(a.Config.AuthHeaderTemplate, "{key}", channel.Key)
	req.Header.Set(a.Config.AuthHeaderName, authHeader)

	// 设置 Content-Type
	req.Header.Set("Content-Type", a.Config.RequestContentType)

	// 应用自定义 Headers
	if a.Config.CustomHeaders != "" {
		var headers []map[string]string
		if err := json.Unmarshal([]byte(a.Config.CustomHeaders), &headers); err == nil {
			for _, h := range headers {
				if key, ok := h["key"]; ok {
					if value, ok := h["value"]; ok {
						// 支持变量替换
						value = strings.ReplaceAll(value, "{key}", channel.Key)
						req.Header.Set(key, value)
					}
				}
			}
		}
	}

	// 应用渠道自定义 Headers
	common.ApplyHeaders(req, channel.Headers)

	// 创建 HTTP 客户端（带超时）
	timeout := time.Duration(a.Config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	// 发送请求
	return client.Do(req)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *model.Token) (usage *dto.Usage, err error) {
	// 检查是否为流式响应
	isStream := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")

	if isStream && a.Config.StreamEnabled == 1 {
		return a.handleStreamResponse(c, resp, meta)
	}

	return a.handleNormalResponse(c, resp, meta)
}

func (a *Adaptor) handleNormalResponse(c *gin.Context, resp *http.Response, meta *model.Token) (*dto.Usage, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 解析 JSON 响应
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// 提取内容（简单实现，使用点路径）
	content := extractByPath(result, a.Config.ResponseContentPath)

	// 提取 Usage
	var usage *dto.Usage
	if a.Config.ResponseUsagePath != "" {
		promptTokens := extractIntByPath(result, a.Config.ResponsePromptTokensPath)
		completionTokens := extractIntByPath(result, a.Config.ResponseCompletionTokensPath)
		totalTokens := extractIntByPath(result, a.Config.ResponseTotalTokensPath)

		// 如果没有 total_tokens，自动计算
		if totalTokens == 0 {
			totalTokens = promptTokens + completionTokens
		}

		usage = &dto.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
		}
	}

	// 构建 OpenAI 格式响应
	openaiResp := gin.H{
		"id":      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   meta.Name,
		"choices": []gin.H{
			{
				"index": 0,
				"message": gin.H{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": "stop",
			},
		},
	}

	if usage != nil {
		openaiResp["usage"] = usage
	}

	// 宸汐玄鉴: 把自定义渠道回复原文塞进侧信道 方便后面风控扫描喵～
	if content != "" {
		c.Set("qy_completion_text", content)
	}

	c.JSON(http.StatusOK, openaiResp)
	return usage, nil
}

// extractByPath 从嵌套 map 中提取字符串值（简单实现）
func extractByPath(data map[string]any, path string) string {
	if path == "" {
		return ""
	}

	parts := strings.Split(path, ".")
	var current any = data

	for _, part := range parts {
		if m, ok := current.(map[string]any); ok {
			current = m[part]
		} else {
			return ""
		}
	}

	if str, ok := current.(string); ok {
		return str
	}
	return fmt.Sprint(current)
}

// extractIntByPath 从嵌套 map 中提取整数值
func extractIntByPath(data map[string]any, path string) int {
	if path == "" {
		return 0
	}

	parts := strings.Split(path, ".")
	var current any = data

	for _, part := range parts {
		if m, ok := current.(map[string]any); ok {
			current = m[part]
		} else {
			return 0
		}
	}

	switch v := current.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func (a *Adaptor) handleStreamResponse(c *gin.Context, resp *http.Response, meta *model.Token) (*dto.Usage, error) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	scanner := bufio.NewScanner(resp.Body)
	defer resp.Body.Close()

	var totalContent strings.Builder
	var promptTokens, completionTokens int

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过空行
		if strings.TrimSpace(line) == "" {
			continue
		}

		// 移除数据前缀
		if a.Config.StreamDataPrefix != "" && strings.HasPrefix(line, a.Config.StreamDataPrefix) {
			line = strings.TrimPrefix(line, a.Config.StreamDataPrefix)
		}

		// 检查结束标记
		if strings.TrimSpace(line) == a.Config.StreamEndMarker {
			c.SSEvent("", " [DONE]")
			c.Writer.Flush()
			break
		}

		// 解析 JSON
		var result map[string]any
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}

		// 提取内容
		content := extractByPath(result, a.Config.StreamContentPath)
		if content != "" {
			totalContent.WriteString(content)

			// 构建 OpenAI 格式的流式响应
			chunk := gin.H{
				"id":      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
				"object":  "chat.completion.chunk",
				"created": time.Now().Unix(),
				"model":   meta.Name,
				"choices": []gin.H{
					{
						"index": 0,
						"delta": gin.H{
							"content": content,
						},
						"finish_reason": nil,
					},
				},
			}

			chunkBytes, _ := json.Marshal(chunk)
			c.SSEvent("", " "+string(chunkBytes))
			c.Writer.Flush()
		}
	}

	// 估算 token 使用：completion 用 tiktoken 精确计数，prompt 用 ConvertRequest 时预算的值
	completionTokens = common.CountToken(totalContent.String())
	promptTokens = a.promptTokenCount
	if promptTokens == 0 {
		promptTokens = completionTokens / 2 // 兜底估算
	}

	// 宸汐玄鉴: 流式拼好的完整回复也塞进侧信道喵～
	if totalContent.Len() > 0 {
		c.Set("qy_completion_text", totalContent.String())
	}

	return &dto.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}, nil
}
