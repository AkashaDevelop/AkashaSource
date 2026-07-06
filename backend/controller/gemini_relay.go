package controller

import (
	"STfreApi/common"
	"STfreApi/middleware"
	"STfreApi/model"
	"STfreApi/service"
	"STfreApi/service/qingyuan"
	"STfreApi/service/xuanjian"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// geminiNativeResponse 只取我们要检测/清理的那部分字段，其余原样透传，
// 不引入 adapter/gemini 包的结构体，避免 controller 层跨包耦合～
type geminiNativeResponse struct {
	Candidates []struct {
		Content struct {
			Role  string `json:"role,omitempty"`
			Parts []struct {
				Text string `json:"text,omitempty"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason,omitempty"`
		Index        int    `json:"index,omitempty"`
	} `json:"candidates"`
	PromptFeedback json.RawMessage `json:"promptFeedback,omitempty"`
	UsageMetadata  json.RawMessage `json:"usageMetadata,omitempty"`
}

// RelayGeminiNative proxies requests in Gemini's native REST format
func RelayGeminiNative(c *gin.Context) {
	startTime := time.Now()
	// 1. Auth via query param or header
	apiKey := c.Query("key")
	if apiKey == "" {
		authHeader := c.GetHeader("Authorization")
		apiKey = strings.TrimPrefix(authHeader, "Bearer ")
	}
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing api key"})
		return
	}

	token, err := GetTokenByKey(apiKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
		return
	}
	if err := ValidateToken(token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 2. Extract model and action from URL path (*path = "/gemini-pro:generateContent")
	rawPath := strings.TrimPrefix(c.Param("path"), "/")
	parts := strings.SplitN(rawPath, ":", 2)
	modelName := parts[0]
	if modelName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model required"})
		return
	}
	action := "generateContent"
	if len(parts) == 2 && parts[1] != "" {
		action = parts[1]
	}

	// 3. Get user
	var user model.User
	if err := common.DB.Where("id = ?", token.UserId).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	// 4. Select Gemini channel
	usingGroup, err := service.ResolveUsingGroup(user.Group, token.Group)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if !middleware.GroupRateLimitMiddleware(service.ResolveBillingGroup(user.Group, token.Group)) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": fmt.Sprintf("分组 %s 请求过于频繁", usingGroup)})
		return
	}
	channels, mappedModels, channelGroups, err := SelectChannelWithAffinity(modelName, user.Group, usingGroup, token.CrossGroupRetry, apiKey, defaultChannelAffinityRule)
	if err != nil || len(channels) == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": fmt.Sprintf("no channel for model: %s", modelName),
		})
		return
	}

	// Find a Gemini-type channel
	selectedIndex := 0
	for i := range channels {
		if channels[i].Type == model.ChannelTypeGemini {
			selectedIndex = i
			break
		}
	}
	channel := channels[selectedIndex]
	mappedModel := modelName
	if selectedIndex < len(mappedModels) && strings.TrimSpace(mappedModels[selectedIndex]) != "" {
		mappedModel = mappedModels[selectedIndex]
	}
	billingGroup := usingGroup
	if selectedIndex < len(channelGroups) {
		billingGroup = channelGroups[selectedIndex]
	}
	channel.Key = service.GetNextKey(channel.Key)

	// 5. Read body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// 5.5 宸汐玄鉴 AI 审核（预审 / 规则初审+AI复审 / 两者）
	// 同时构造 reviewMessages 供事后分析使用
	reviewMessages := make([]interface{}, 0)
	{
		var geminiReq struct {
			Contents []struct {
				Role  string `json:"role"`
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"contents"`
		}
		if json.Unmarshal(bodyBytes, &geminiReq) == nil {
			for _, content := range geminiReq.Contents {
				var text string
				for _, part := range content.Parts {
					text += part.Text + " "
				}
				if text != "" {
					reviewMessages = append(reviewMessages, map[string]interface{}{
						"role": content.Role, "content": text,
					})
				}
			}
		}
	}
	if xuanjian.IsEnabled() && len(reviewMessages) > 0 {
		allowed, blockMsg := xuanjian.AIReviewCheck(token.Id, user.Id, reviewMessages, "")
		if !allowed {
			go RecordFailLog(c, token, modelName, blockMsg)
			c.JSON(http.StatusBadRequest, gin.H{"error": blockMsg})
			return
		}
	}

	// 6. Forward to upstream Gemini
	baseURL := channel.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	targetURL := fmt.Sprintf("%s/v1beta/models/%s:%s?key=%s",
		baseURL, mappedModel, action, channel.Key)

	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	common.ApplyHeaders(req, channel.Headers)

	client := common.NewHTTPClient(channel.Proxy)
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	// 8. 读取上游响应，命中成功状态码时先跑一遍宸汐清源检测/清理，再决定发给客户端的字节内容
	respBody, _ := io.ReadAll(resp.Body)

	policy := qingyuan.ResolvePolicy(channel.Id, modelName, mappedModel)
	reqCtx := qingyuan.RequestContext{
		RequestId:      c.GetString("request_id"),
		UserId:         user.Id,
		TokenId:        token.Id,
		TokenName:      token.Name,
		ChannelId:      channel.Id,
		ChannelName:    channel.Name,
		ChannelType:    channel.Type,
		RequestedModel: modelName,
		MappedModel:    mappedModel,
		UserGroup:      user.Group,
		ClientIP:       c.ClientIP(),
	}
	respCtx := qingyuan.ResponseContext{RequestContext: reqCtx}

	completionText := ""
	var respFindings []qingyuan.Finding
	if resp.StatusCode == http.StatusOK && qingyuan.IsEnabled(policy) {
		var geminiResp geminiNativeResponse
		if json.Unmarshal(respBody, &geminiResp) == nil {
			changed := false
			for ci := range geminiResp.Candidates {
				for pi := range geminiResp.Candidates[ci].Content.Parts {
					text := geminiResp.Candidates[ci].Content.Parts[pi].Text
					if text == "" {
						continue
					}
					completionText += text
					cleaned, _, _, findings := qingyuan.AnalyzeText(text, nil, respCtx, policy)
					respFindings = append(respFindings, findings...)
					if cleaned != text {
						geminiResp.Candidates[ci].Content.Parts[pi].Text = cleaned
						changed = true
					}
				}
			}
			if changed {
				if newBody, err := json.Marshal(geminiResp); err == nil {
					respBody = newBody
				}
			}
			if len(respFindings) > 0 {
				qingyuan.RecordResponseFindings(respCtx, policy, respFindings, changed)
			}
		}
	}

	for k, v := range resp.Header {
		c.Writer.Header().Set(k, v[0])
	}
	c.Writer.WriteHeader(resp.StatusCode)
	c.Writer.Write(respBody)

	// 9. Estimate usage for billing
	if resp.StatusCode == http.StatusOK {
		promptTokens := common.CountTokenForModel(string(bodyBytes), mappedModel)
		completionTokens := common.CountTokenForModel(string(respBody), mappedModel)
		c.Set("channel_id", channel.Id)
		c.Set("billing_group", billingGroup)
		c.Set("is_stream", false)
		c.Set("use_time", int(time.Since(startTime).Milliseconds()))
		go RecordConsumeLog(c, token, mappedModel,
			promptTokens, completionTokens)
		go upsertChannelAffinity(defaultChannelAffinityRule, user.Group, getChannelAffinityKeyFP(apiKey), channel.Id, mappedModel, promptTokens, completionTokens, 0)

		// 宸汐玄鉴：异步行为分析
		if xuanjian.IsEnabled() && len(reviewMessages) > 0 {
			promptSnippet := xuanjian.CollectPromptSnippet(reviewMessages, "", 2000)
			go xuanjian.RecordRequest(xuanjian.RequestRecord{
				TokenID:           token.Id,
				TokenName:         token.Name,
				UserID:            user.Id,
				TokenCreatedAt:    time.Unix(token.CreatedTime, 0),
				IP:                c.ClientIP(),
				UserAgent:         c.GetHeader("User-Agent"),
				Model:             mappedModel,
				PromptTokens:      promptTokens,
				CompletionTokens:  completionTokens,
				PromptSnippet:     promptSnippet,
				CompletionSnippet: xuanjian.TruncateSnippet(completionText, 500),
				Messages:          reviewMessages,
				StatusCode:        resp.StatusCode,
				QYFindings:        xuanjian.ExtractQYFindingTypes(respFindings),
			})
		}
	}
}
