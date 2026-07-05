package controller

import (
	"STfreApi/adapter"
	"STfreApi/adapter/claude"
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/middleware"
	"STfreApi/model"
	"STfreApi/service"
	"STfreApi/service/qingyuan"
	"STfreApi/service/xuanjian"
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

// RelayMessages handles Anthropic Messages API format requests (POST /v1/messages)
// This enables Claude Code CLI, and other Anthropic-native clients to use the gateway.
func RelayMessages(c *gin.Context) {
	startTime := time.Now()
	// 1. Auth: support both x-api-key and Authorization: Bearer
	tokenKey := c.GetHeader("x-api-key")
	if tokenKey == "" {
		authHeader := c.GetHeader("Authorization")
		tokenKey = strings.TrimPrefix(authHeader, "Bearer ")
	}
	if tokenKey == "" {
		c.JSON(http.StatusUnauthorized, claude.ClaudeErrorResponse{
			Type:  "error",
			Error: claude.ClaudeError{Type: "authentication_error", Message: "missing api key"},
		})
		return
	}

	token, err := GetTokenByKey(tokenKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, claude.ClaudeErrorResponse{
			Type:  "error",
			Error: claude.ClaudeError{Type: "authentication_error", Message: "invalid api key"},
		})
		return
	}
	if err := ValidateToken(token); err != nil {
		c.JSON(http.StatusUnauthorized, claude.ClaudeErrorResponse{
			Type:  "error",
			Error: claude.ClaudeError{Type: "authentication_error", Message: err.Error()},
		})
		return
	}

	// 2. Read and parse the Anthropic-format request body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		sendClaudeError(c, http.StatusBadRequest, "invalid_request_error", "failed to read request body")
		return
	}

	var claudeReq claude.ClaudeRequest
	if err := json.Unmarshal(bodyBytes, &claudeReq); err != nil {
		sendClaudeError(c, http.StatusBadRequest, "invalid_request_error", "invalid JSON")
		return
	}

	if claudeReq.Model == "" {
		sendClaudeError(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	// 3. Token model check
	if !CheckTokenModel(token, claudeReq.Model) {
		sendClaudeError(c, http.StatusForbidden, "invalid_request_error", "model not allowed for this token")
		return
	}

	// 4. Model rate limit
	if !middleware.ModelRateLimitMiddleware(claudeReq.Model) {
		sendClaudeError(c, http.StatusTooManyRequests, "rate_limit_error",
			fmt.Sprintf("model %s rate limit exceeded", claudeReq.Model))
		return
	}

	// 5. Get user for channel selection
	var user model.User
	if err := common.DB.Where("id = ?", token.UserId).First(&user).Error; err != nil {
		sendClaudeError(c, http.StatusInternalServerError, "api_error", "internal error")
		return
	}

	// 5.5 Parse reasoning suffix
	reasoningParams := common.ParseReasoningSuffix(claudeReq.Model)
	claudeReq.Model = reasoningParams.CleanModel

	// 6. Select channels
	usingGroup, err := service.ResolveUsingGroup(user.Group, token.Group)
	if err != nil {
		sendClaudeError(c, http.StatusForbidden, "invalid_request_error", err.Error())
		return
	}
	if !middleware.GroupRateLimitMiddleware(service.ResolveBillingGroup(user.Group, token.Group)) {
		sendClaudeError(c, http.StatusTooManyRequests, "rate_limit_error", fmt.Sprintf("分组 %s 请求过于频繁", usingGroup))
		return
	}
	channels, mappedModels, channelGroups, err := SelectChannelWithAffinity(claudeReq.Model, user.Group, usingGroup, token.CrossGroupRetry, tokenKey, defaultChannelAffinityRule)
	if err != nil {
		sendClaudeError(c, http.StatusServiceUnavailable, "api_error",
			fmt.Sprintf("no available channel for model: %s", claudeReq.Model))
		return
	}

	// 6.5 宸汐玄鉴 AI 审核（预审 / 规则初审+AI复审 / 两者）
	// 同时构造 reviewMessages 供事后分析使用
	reviewMessages := make([]interface{}, 0, len(claudeReq.Messages))
	for _, msg := range claudeReq.Messages {
		var contentStr string
		if json.Unmarshal(msg.Content, &contentStr) == nil {
			reviewMessages = append(reviewMessages, map[string]interface{}{
				"role": msg.Role, "content": contentStr,
			})
		}
	}
	if xuanjian.IsEnabled() {
		allowed, blockMsg := xuanjian.AIReviewCheck(token.Id, user.Id, reviewMessages, "")
		if !allowed {
			go RecordFailLog(c, token, claudeReq.Model, blockMsg)
			sendClaudeError(c, http.StatusBadRequest, "invalid_request_error", blockMsg)
			return
		}
	}

	// 6.6 清源净化：转换为 OpenAI 格式后执行净化检查
	sanitizedOpenAIReq := claudeToOpenAIRequest(&claudeReq)
	var sanitizeResult = &qingyuan.RequestResult{}
	var qyPolicy qingyuan.ResolvedPolicy
	var qyRespCtx qingyuan.ResponseContext
	if len(channels) > 0 {
		qyPolicy = qingyuan.ResolvePolicy(channels[0].Id, claudeReq.Model, mappedModels[0])
		reqCtx := qingyuan.RequestContext{
			RequestId:      c.GetString("request_id"),
			UserId:         user.Id,
			TokenId:        token.Id,
			TokenName:      token.Name,
			ChannelId:      channels[0].Id,
			ChannelName:    channels[0].Name,
			ChannelType:    channels[0].Type,
			RequestedModel: claudeReq.Model,
			MappedModel:    mappedModels[0],
			UserGroup:      user.Group,
			ClientIP:       c.ClientIP(),
		}
		sanitizeResult, _ = qingyuan.ApplyRequest(c.Request.Context(), sanitizedOpenAIReq, reqCtx, qyPolicy)
		if sanitizeResult.Blocked {
			msg := sanitizeResult.Message
			if msg == "" {
				msg = "请求被安全策略拒绝，如有疑问请联系管理员"
			}
			go RecordFailLog(c, token, claudeReq.Model, msg)
			sendClaudeError(c, http.StatusBadRequest, "invalid_request_error", msg)
			return
		}
		// 净化阶段如果给 sanitizedOpenAIReq 注入了安全 guard(前置 system 消息)，
		// 走 Claude 原生直连分支根本不会碰 sanitizedOpenAIReq，guard 得原样搬到 claudeReq.System 上，
		// 不然 protect 模式配置的安全边界对原生 Claude 渠道形同虚设～
		if guardText := extractLeadingSystemText(sanitizedOpenAIReq.Messages); guardText != "" {
			if sysJSON, err := json.Marshal(guardText); err == nil {
				claudeReq.System = sysJSON
			}
		}
		qyRespCtx = qingyuan.ResponseContext{
			RequestContext:   reqCtx,
			UserRequestedAds: qingyuan.IsUserRequestingAds(sanitizedOpenAIReq),
			RequestTools:     sanitizedOpenAIReq.Tools,
		}
	}

	// 7. Try channels
	var lastError error
	for i, channel := range channels {
		mappedModel := mappedModels[i]
		c.Set("billing_group", channelGroups[i])
		claudeReq.Model = mappedModel
		sanitizedOpenAIReq.Model = mappedModel
		channel.Key = service.GetNextKey(channel.Key)

		if channel.Type == model.ChannelTypeAnthropic {
			// Direct passthrough to Claude upstream
			usage, completionText, err := relayClaudeDirect(c, channel, &claudeReq, bodyBytes, mappedModel, qyPolicy, qyRespCtx)
			if err != nil {
				lastError = err
				continue
			}
			if usage != nil {
				c.Set("channel_id", channel.Id)
				c.Set("is_stream", claudeReq.Stream)
				c.Set("use_time", int(time.Since(startTime).Milliseconds()))
				go RecordConsumeLog(c, token, mappedModel, usage.PromptTokens, usage.CompletionTokens)
				go upsertChannelAffinity(defaultChannelAffinityRule, user.Group, getChannelAffinityKeyFP(tokenKey), channel.Id, mappedModel, usage.PromptTokens, usage.CompletionTokens, usage.CachedTokens)
				// 宸汐玄鉴：异步行为分析
				if xuanjian.IsEnabled() {
					promptSnippet := xuanjian.CollectPromptSnippet(reviewMessages, "", 2000)
					go xuanjian.RecordRequest(xuanjian.RequestRecord{
						TokenID:           token.Id,
						TokenName:         token.Name,
						UserID:            user.Id,
						TokenCreatedAt:    time.Unix(token.CreatedTime, 0),
						IP:                c.ClientIP(),
						UserAgent:         c.GetHeader("User-Agent"),
						Model:             mappedModel,
						PromptTokens:      usage.PromptTokens,
						CompletionTokens:  usage.CompletionTokens,
						PromptSnippet:     promptSnippet,
						CompletionSnippet: xuanjian.TruncateSnippet(completionText, 500),
						Messages:          reviewMessages,
						StatusCode:        200,
						QYFindings:        xuanjian.ExtractQYFindingTypes(sanitizeResult.Findings),
						QYRiskScore:       sanitizeResult.RiskScore,
					})
				}
			}
			return
		}

		// Non-Claude channel: convert to OpenAI format, relay, convert response back
		usage, completionText, err := relayClaudeViaOpenAI(c, channel, sanitizedOpenAIReq, token, qyPolicy, qyRespCtx)
		if err != nil {
			lastError = err
			continue
		}
		if usage != nil {
			c.Set("channel_id", channel.Id)
			c.Set("is_stream", claudeReq.Stream)
			c.Set("use_time", int(time.Since(startTime).Milliseconds()))
			go RecordConsumeLog(c, token, mappedModel, usage.PromptTokens, usage.CompletionTokens)
			go upsertChannelAffinity(defaultChannelAffinityRule, user.Group, getChannelAffinityKeyFP(tokenKey), channel.Id, mappedModel, usage.PromptTokens, usage.CompletionTokens, usage.CachedTokens)
			// 宸汐玄鉴：异步行为分析
			if xuanjian.IsEnabled() {
				promptSnippet := xuanjian.CollectPromptSnippet(sanitizedOpenAIReq.Messages, sanitizedOpenAIReq.Prompt, 2000)
				go xuanjian.RecordRequest(xuanjian.RequestRecord{
					TokenID:           token.Id,
					TokenName:         token.Name,
					UserID:            user.Id,
					TokenCreatedAt:    time.Unix(token.CreatedTime, 0),
					IP:                c.ClientIP(),
					UserAgent:         c.GetHeader("User-Agent"),
					Model:             mappedModel,
					PromptTokens:      usage.PromptTokens,
					CompletionTokens:  usage.CompletionTokens,
					PromptSnippet:     promptSnippet,
					CompletionSnippet: xuanjian.TruncateSnippet(completionText, 500),
					Messages:          sanitizedOpenAIReq.Messages,
					StatusCode:        200,
					QYFindings:        xuanjian.ExtractQYFindingTypes(sanitizeResult.Findings),
					QYRiskScore:       sanitizeResult.RiskScore,
				})
			}
		}
		return
	}

	errMsg := fmt.Sprintf("all channels failed, last error: %v", lastError)
	go RecordFailLog(c, token, claudeReq.Model, errMsg)
	sendClaudeError(c, http.StatusServiceUnavailable, "api_error", errMsg)
}

func sendClaudeError(c *gin.Context, status int, errType, message string) {
	c.JSON(status, claude.ClaudeErrorResponse{
		Type:  "error",
		Error: claude.ClaudeError{Type: errType, Message: message},
	})
}

// relayClaudeDirect forwards the request directly to a Claude upstream channel
func relayClaudeDirect(c *gin.Context, channel *model.Channel, req *claude.ClaudeRequest, rawBody []byte, mappedModel string, policy qingyuan.ResolvedPolicy, rc qingyuan.ResponseContext) (*dto.Usage, string, error) {
	// Re-marshal with mapped model
	req.Model = mappedModel
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, "", err
	}

	baseURL := channel.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	httpReq, err := http.NewRequest("POST", baseURL+"/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, "", err
	}

	httpReq.Header.Set("x-api-key", channel.Key)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("content-type", "application/json")
	// Forward beta headers from client
	if beta := c.GetHeader("anthropic-beta"); beta != "" {
		httpReq.Header.Set("anthropic-beta", beta)
	}
	common.ApplyHeaders(httpReq, channel.Headers)

	client := common.NewHTTPClient(channel.Proxy)
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("upstream error %d: %s", resp.StatusCode, string(body))
	}

	// Stream passthrough or normal passthrough
	isStream := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")
	if isStream {
		return streamPassthroughClaude(c, resp, policy, rc)
	}
	return normalPassthroughClaude(c, resp, policy, rc)
}

// normalPassthroughClaude passes a non-streaming Claude response directly to the client，
// 顺手对 content 文本 block 跑一遍宸汐清源检测，命中广告/注入就地清理后再转发～
func normalPassthroughClaude(c *gin.Context, resp *http.Response, policy qingyuan.ResolvedPolicy, rc qingyuan.ResponseContext) (*dto.Usage, string, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	// Extract usage before forwarding
	var claudeResp claude.ClaudeResponse
	usage := &dto.Usage{}
	completionText := ""
	if json.Unmarshal(body, &claudeResp) == nil {
		usage.PromptTokens = claudeResp.Usage.InputTokens
		usage.CompletionTokens = claudeResp.Usage.OutputTokens
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens

		changed := false
		for i := range claudeResp.Content {
			if claudeResp.Content[i].Type != "text" || claudeResp.Content[i].Text == "" {
				continue
			}
			completionText += claudeResp.Content[i].Text
			if qingyuan.IsEnabled(policy) {
				cleaned, _, _, findings := qingyuan.AnalyzeText(claudeResp.Content[i].Text, nil, rc, policy)
				if len(findings) > 0 {
					qingyuan.RecordResponseFindings(rc, policy, findings, cleaned != claudeResp.Content[i].Text)
				}
				if cleaned != claudeResp.Content[i].Text {
					claudeResp.Content[i].Text = cleaned
					changed = true
				}
			}
		}
		if changed {
			if newBody, err := json.Marshal(claudeResp); err == nil {
				body = newBody
			}
		}
	}

	// Forward response as-is
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
	return usage, completionText, nil
}

// streamPassthroughClaude passes a streaming Claude response directly to the client，
// 有清源策略时把 content_block_delta 的文本增量喂给 TailBufferCore，尾部广告能被真正拦下来，
// 不像以前那样"边收边发、检测结果只能拿来记日志"～
func streamPassthroughClaude(c *gin.Context, resp *http.Response, policy qingyuan.ResolvedPolicy, rc qingyuan.ResponseContext) (*dto.Usage, string, error) {
	common.SetEventStreamHeaders(c)
	scanner := bufio.NewScanner(resp.Body)
	usage := &dto.Usage{}

	var core *qingyuan.TailBufferCore
	if qingyuan.IsEnabled(policy) {
		core = qingyuan.NewTailBufferCore(policy)
	}

	for scanner.Scan() {
		line := scanner.Text()
		raw := line + "\n"
		deltaText := ""

		// Extract usage / text delta from stream events
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var event claude.ClaudeStreamEvent
			if json.Unmarshal([]byte(data), &event) == nil {
				if event.Type == "message_start" && event.Message != nil {
					usage.PromptTokens = event.Message.Usage.InputTokens
				}
				if event.Type == "message_delta" && event.Usage != nil {
					usage.CompletionTokens = event.Usage.OutputTokens
				}
				if event.Type == "content_block_delta" && event.Delta != nil {
					deltaText = event.Delta.Text
				}
			}
		}

		if core == nil {
			c.Writer.WriteString(raw)
			c.Writer.Flush()
			continue
		}
		for _, released := range core.Feed(deltaText, raw) {
			c.Writer.WriteString(released.(string))
		}
		c.Writer.Flush()
	}

	completionText := ""
	if core != nil {
		released, fullText, findings := core.Finalize(rc.RequestTools)
		for _, r := range released {
			c.Writer.WriteString(r.(string))
		}
		c.Writer.Flush()
		completionText = fullText
		qingyuan.RecordResponseFindings(rc, policy, findings, false)
	}

	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	return usage, completionText, nil
}

// relayClaudeViaOpenAI sends an OpenAI-format request to a non-Claude channel,
// then converts the response back to Anthropic format.
func relayClaudeViaOpenAI(c *gin.Context, channel *model.Channel, openAIReq *dto.OpenAIRequest, token *model.Token, policy qingyuan.ResolvedPolicy, rc qingyuan.ResponseContext) (*dto.Usage, string, error) {
	// Use the adapter system
	adaptor := adapter.GetAdaptor(channel.Type, channel)
	converted, err := adaptor.ConvertRequest(c, openAIReq)
	if err != nil {
		return nil, "", err
	}

	resp, err := adaptor.DoRequest(c, channel, converted)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("upstream error %d: %s", resp.StatusCode, string(body))
	}

	// Read the OpenAI response and convert back to Claude format
	isStream := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")
	if isStream {
		return streamOpenAI2Claude(c, resp, openAIReq.Model, policy, rc)
	}
	return normalOpenAI2Claude(c, resp, openAIReq.Model, policy, rc)
}

// extractLeadingSystemText 把 messages 开头连续的 system 消息内容拼起来～
// injectGuard 是往 Messages 最前面插入 guard，原 Claude 系统提示（转换时也在最前面）会跟着排第二，
// 拼接顺序天然是"guard 在前、原系统提示在后"，符合安全边界优先的语义～
func extractLeadingSystemText(messages []interface{}) string {
	var parts []string
	for _, m := range messages {
		mm, ok := m.(map[string]interface{})
		if !ok {
			break
		}
		role, _ := mm["role"].(string)
		if role != "system" {
			break
		}
		if content, ok := mm["content"].(string); ok && content != "" {
			parts = append(parts, content)
		}
	}
	return strings.Join(parts, "\n\n")
}

// claudeToOpenAIRequest converts an Anthropic Messages request to OpenAI format
func claudeToOpenAIRequest(req *claude.ClaudeRequest) *dto.OpenAIRequest {
	openAIReq := &dto.OpenAIRequest{
		Model:     req.Model,
		Stream:    req.Stream,
		MaxTokens: req.MaxTokens,
	}

	// System message
	if len(req.System) > 0 {
		var sysText string
		if json.Unmarshal(req.System, &sysText) == nil && sysText != "" {
			openAIReq.Messages = append(openAIReq.Messages, map[string]interface{}{
				"role": "system", "content": sysText,
			})
		}
	}

	// Convert messages
	for _, msg := range req.Messages {
		var contentStr string
		if json.Unmarshal(msg.Content, &contentStr) == nil {
			openAIReq.Messages = append(openAIReq.Messages, map[string]interface{}{
				"role": msg.Role, "content": contentStr,
			})
			continue
		}
		// Array content — handle text, tool_use, tool_result, image
		var blocks []claude.ContentBlock
		if json.Unmarshal(msg.Content, &blocks) == nil {
			// Check if this message contains tool_use blocks (assistant with tool calls)
			var toolCalls []map[string]interface{}
			var textParts []string
			var toolResultParts []map[string]interface{}
			var contentParts []interface{}

			for _, b := range blocks {
				switch b.Type {
				case "text":
					textParts = append(textParts, b.Text)
				case "tool_use":
					toolCalls = append(toolCalls, map[string]interface{}{
						"id":   b.ID,
						"type": "function",
						"function": map[string]interface{}{
							"name":      b.Name,
							"arguments": string(b.Input),
						},
					})
				case "tool_result":
					// tool_result becomes a separate "tool" role message
					var resultContent string
					// Try string first
					if json.Unmarshal(b.Content, &resultContent) != nil {
						// Try array of content blocks
						var innerBlocks []claude.ContentBlock
						if json.Unmarshal(b.Content, &innerBlocks) == nil {
							for _, ib := range innerBlocks {
								if ib.Type == "text" {
									resultContent += ib.Text
								}
							}
						} else {
							resultContent = string(b.Content)
						}
					}
					toolResultParts = append(toolResultParts, map[string]interface{}{
						"role":         "tool",
						"content":      resultContent,
						"tool_call_id": b.ToolUseID,
					})
				case "image":
					if b.Source != nil {
						dataURL := fmt.Sprintf("data:%s;base64,%s", b.Source.MediaType, b.Source.Data)
						contentParts = append(contentParts, map[string]interface{}{
							"type": "image_url",
							"image_url": map[string]interface{}{
								"url": dataURL,
							},
						})
					}
				}
			}

			// Build the message(s)
			if msg.Role == "assistant" && len(toolCalls) > 0 {
				m := map[string]interface{}{
					"role":       "assistant",
					"tool_calls": toolCalls,
				}
				if len(textParts) > 0 {
					m["content"] = strings.Join(textParts, "")
				}
				openAIReq.Messages = append(openAIReq.Messages, m)
			} else if len(contentParts) > 0 {
				// Has images — use array content format
				for _, t := range textParts {
					contentParts = append([]interface{}{map[string]interface{}{
						"type": "text", "text": t,
					}}, contentParts...)
				}
				openAIReq.Messages = append(openAIReq.Messages, map[string]interface{}{
					"role": msg.Role, "content": contentParts,
				})
			} else if len(textParts) > 0 {
				openAIReq.Messages = append(openAIReq.Messages, map[string]interface{}{
					"role": msg.Role, "content": strings.Join(textParts, ""),
				})
			}

			// Append tool_result messages as separate "tool" role messages
			for _, tr := range toolResultParts {
				openAIReq.Messages = append(openAIReq.Messages, tr)
			}
		}
	}

	// Convert tools
	if len(req.Tools) > 0 {
		for _, t := range req.Tools {
			openAIReq.Tools = append(openAIReq.Tools, map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  json.RawMessage(t.InputSchema),
				},
			})
		}
	}

	return openAIReq
}

// normalOpenAI2Claude converts a non-streaming OpenAI response to Claude format
func normalOpenAI2Claude(c *gin.Context, resp *http.Response, modelName string, policy qingyuan.ResolvedPolicy, rc qingyuan.ResponseContext) (*dto.Usage, string, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	// 这一步天然就是标准 OpenAI ChatCompletion 形状，直接复用 ApplyOpenAIResponse 的检测/清理逻辑～
	if qingyuan.IsEnabled(policy) {
		if result, err := qingyuan.ApplyOpenAIResponse(c.Request.Context(), body, rc, policy); err == nil && result != nil {
			if result.Blocked {
				return nil, "", fmt.Errorf("response blocked by qingyuan: %s", result.Message)
			}
			body = result.Body
		}
	}

	var openAIResp dto.ChatCompletionResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return nil, "", err
	}

	// Build Claude content blocks
	var content []claude.ContentBlock
	completionText := ""
	if len(openAIResp.Choices) > 0 {
		msg := openAIResp.Choices[0].Message
		if msg.Content != "" {
			content = append(content, claude.ContentBlock{Type: "text", Text: msg.Content})
			completionText = msg.Content
		}
		for _, tc := range msg.ToolCalls {
			content = append(content, claude.ContentBlock{
				Type:  "tool_use",
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: json.RawMessage(tc.Function.Arguments),
			})
		}
	}

	stopReason := "end_turn"
	if len(openAIResp.Choices) > 0 {
		stopReason = openAIStopToClaude(openAIResp.Choices[0].FinishReason)
	}

	claudeResp := claude.ClaudeResponse{
		ID:         openAIResp.ID,
		Type:       "message",
		Role:       "assistant",
		Content:    content,
		Model:      modelName,
		StopReason: stopReason,
		Usage: claude.ClaudeUsage{
			InputTokens:  openAIResp.Usage.PromptTokens,
			OutputTokens: openAIResp.Usage.CompletionTokens,
		},
	}

	c.JSON(http.StatusOK, claudeResp)
	return &openAIResp.Usage, completionText, nil
}

// streamOpenAI2Claude converts streaming OpenAI SSE to Claude SSE format，
// 有清源策略时把 delta.Content 喂给 TailBufferCore，收尾事件(content_block_stop/message_stop)
// 永远等 core.Finalize() 把所有待放行内容清干净了才发，保证协议帧顺序不会因为尾部拦截被打乱～
func streamOpenAI2Claude(c *gin.Context, resp *http.Response, modelName string, policy qingyuan.ResolvedPolicy, rc qingyuan.ResponseContext) (*dto.Usage, string, error) {
	common.SetEventStreamHeaders(c)
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(splitSSE)

	usage := &dto.Usage{}
	msgID := fmt.Sprintf("msg_%s", common.GetUUID())
	started := false
	finishReason := ""

	var core *qingyuan.TailBufferCore
	if qingyuan.IsEnabled(policy) {
		core = qingyuan.NewTailBufferCore(policy)
	}

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk dto.ChatCompletionStreamResponse
		if json.Unmarshal([]byte(data), &chunk) != nil {
			continue
		}

		if !started {
			// Send message_start
			startEvent := map[string]interface{}{
				"type": "message_start",
				"message": map[string]interface{}{
					"id":      msgID,
					"type":    "message",
					"role":    "assistant",
					"content": []interface{}{},
					"model":   modelName,
					"usage":   map[string]int{"input_tokens": 0, "output_tokens": 0},
				},
			}
			writeClaudeSSE(c, "message_start", startEvent)
			// Send content_block_start
			blockStart := map[string]interface{}{
				"type":          "content_block_start",
				"index":         0,
				"content_block": map[string]string{"type": "text", "text": ""},
			}
			writeClaudeSSE(c, "content_block_start", blockStart)
			started = true
		}

		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta

		if delta.Content != "" {
			if core == nil {
				blockDelta := map[string]interface{}{
					"type":  "content_block_delta",
					"index": 0,
					"delta": map[string]string{"type": "text_delta", "text": delta.Content},
				}
				writeClaudeSSE(c, "content_block_delta", blockDelta)
			} else {
				rendered := renderClaudeSSE("content_block_delta", map[string]interface{}{
					"type":  "content_block_delta",
					"index": 0,
					"delta": map[string]string{"type": "text_delta", "text": delta.Content},
				})
				for _, released := range core.Feed(delta.Content, rendered) {
					c.Writer.Write(released.([]byte))
					c.Writer.Flush()
				}
			}
		}

		if chunk.Choices[0].FinishReason != "" {
			finishReason = chunk.Choices[0].FinishReason
		}
	}

	completionText := ""
	if core != nil {
		released, fullText, findings := core.Finalize(rc.RequestTools)
		for _, r := range released {
			c.Writer.Write(r.([]byte))
		}
		c.Writer.Flush()
		completionText = fullText
		qingyuan.RecordResponseFindings(rc, policy, findings, false)
	}

	if finishReason != "" {
		writeClaudeSSE(c, "content_block_stop", map[string]interface{}{
			"type": "content_block_stop", "index": 0,
		})
		writeClaudeSSE(c, "message_delta", map[string]interface{}{
			"type":  "message_delta",
			"delta": map[string]string{"stop_reason": openAIStopToClaude(finishReason)},
			"usage": map[string]int{"output_tokens": usage.CompletionTokens},
		})
		writeClaudeSSE(c, "message_stop", map[string]interface{}{"type": "message_stop"})
	}

	return usage, completionText, nil
}

func writeClaudeSSE(c *gin.Context, eventType string, data interface{}) {
	c.Writer.Write(renderClaudeSSE(eventType, data))
	c.Writer.Flush()
}

// renderClaudeSSE 只负责把事件渲染成字节，不负责写出去——供需要"先攒住再决定发不发"的场景使用～
func renderClaudeSSE(eventType string, data interface{}) []byte {
	jsonBytes, _ := json.Marshal(data)
	var buf bytes.Buffer
	buf.WriteString("event: " + eventType + "\n")
	buf.WriteString("data: " + string(jsonBytes) + "\n\n")
	return buf.Bytes()
}

func openAIStopToClaude(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	default:
		return "end_turn"
	}
}

func splitSSE(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
		return i + 2, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
