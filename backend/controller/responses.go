package controller

import (
	"STfreApi/adapter"
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/middleware"
	"STfreApi/model"
	"STfreApi/service"
	"STfreApi/service/qingyuan"
	"STfreApi/service/realname"
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

// ResponsesRequest represents the OpenAI Responses API request format
type ResponsesRequest struct {
	Model        string          `json:"model"`
	Input        json.RawMessage `json:"input"`
	Instructions string          `json:"instructions,omitempty"`
	Stream       bool            `json:"stream,omitempty"`
	Temperature  float64         `json:"temperature,omitempty"`
	TopP         float64         `json:"top_p,omitempty"`
	MaxTokens    int             `json:"max_output_tokens,omitempty"`
	Tools        []any           `json:"tools,omitempty"`
	ToolChoice   any             `json:"tool_choice,omitempty"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
}

// RelayResponses handles OpenAI Responses API format (POST /v1/responses)
// This enables Codex CLI and other Responses API clients.
func RelayResponses(c *gin.Context) {
	startTime := time.Now()
	// 1. Auth
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		sendResponsesError(c, http.StatusUnauthorized, "invalid_api_key", "missing Authorization header")
		return
	}
	tokenKey := strings.TrimPrefix(authHeader, "Bearer ")

	token, err := GetTokenByKey(tokenKey)
	if err != nil {
		sendResponsesError(c, http.StatusUnauthorized, "invalid_api_key", "invalid api key")
		return
	}
	if err := ValidateToken(token); err != nil {
		sendResponsesError(c, http.StatusUnauthorized, "invalid_api_key", err.Error())
		return
	}

	// 实名认证检查（双盲类型场景）
	if err := realname.CheckRealnameRequirement(token.UserId, model.RealnameScenarioDoubleBlind); err != nil {
		sendResponsesError(c, http.StatusForbidden, "realname_required", err.Error())
		return
	}

	// 2. Parse request
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		sendResponsesError(c, http.StatusBadRequest, "invalid_request_error", "failed to read body")
		return
	}

	var req ResponsesRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		sendResponsesError(c, http.StatusBadRequest, "invalid_request_error", "invalid JSON")
		return
	}
	if req.Model == "" {
		sendResponsesError(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	// 3. Token model check
	if !CheckTokenModel(token, req.Model) {
		sendResponsesError(c, http.StatusForbidden, "invalid_request_error", "model not allowed")
		return
	}

	// 风控处置检查（停用/降速/固定低RPM）
	if !middleware.CheckTokenSanction(c, token.Id) {
		return
	}

	// 4. Model rate limit
	if !middleware.ModelRateLimitMiddleware(req.Model) {
		sendResponsesError(c, http.StatusTooManyRequests, "rate_limit_error",
			fmt.Sprintf("model %s rate limit exceeded", req.Model))
		return
	}

	// 5. Get user
	var user model.User
	if err := common.DB.Where("id = ?", token.UserId).First(&user).Error; err != nil {
		sendResponsesError(c, http.StatusInternalServerError, "server_error", "internal error")
		return
	}

	// 6. Convert to OpenAI chat format
	openAIReq := responsesToChatRequest(&req)

	// 6.5 Parse reasoning suffix
	reasoningParams := common.ParseReasoningSuffix(openAIReq.Model)
	openAIReq.Model = reasoningParams.CleanModel
	if reasoningParams.ReasoningEffort != "" {
		openAIReq.ReasoningEffort = reasoningParams.ReasoningEffort
	}
	if reasoningParams.ThinkingMode {
		openAIReq.ThinkingMode = true
	}

	// 7. Select channels
	usingGroup, err := service.ResolveUsingGroupFull(user.Group, user.ExtraGroups, token.Group)
	if err != nil {
		sendResponsesError(c, http.StatusForbidden, "invalid_request_error", err.Error())
		return
	}
	if !middleware.GroupRateLimitMiddleware(service.ResolveBillingGroupFull(user.Group, user.ExtraGroups, token.Group)) {
		sendResponsesError(c, http.StatusTooManyRequests, "rate_limit_error", fmt.Sprintf("分组 %s 请求过于频繁", usingGroup))
		return
	}
	channels, mappedModels, channelGroups, err := SelectChannelWithAffinity(openAIReq.Model, user.Group, usingGroup, token.CrossGroupRetry, tokenKey, defaultChannelAffinityRule)
	if err != nil {
		sendResponsesError(c, http.StatusServiceUnavailable, "server_error",
			fmt.Sprintf("no available channel for model: %s", openAIReq.Model))
		return
	}

	// 7.5 宸汐玄鉴 AI 审核（预审 / 规则初审+AI复审 / 两者）
	if xuanjian.IsEnabled() {
		allowed, blockMsg := xuanjian.AIReviewCheck(token.Id, user.Id, openAIReq.Messages, openAIReq.Prompt)
		if !allowed {
			go RecordFailLog(c, token, req.Model, blockMsg)
			sendResponsesError(c, http.StatusBadRequest, "ai_review_blocked", blockMsg)
			return
		}
	}

	// 8. Try channels
	var lastError error
	for i, channel := range channels {
		mappedModel := mappedModels[i]
		c.Set("billing_group", channelGroups[i])
		openAIReq.Model = mappedModel
		channel.Key = service.GetNextKey(channel.Key)

		// 清源净化：深拷贝 → 策略解析 → 净化执行
		attemptReq, copyErr := qingyuan.DeepCopyOpenAIRequest(openAIReq)
		if copyErr != nil {
			lastError = copyErr
			continue
		}
		policy := qingyuan.ResolvePolicy(channel.Id, req.Model, mappedModel)
		reqCtx := qingyuan.RequestContext{
			RequestId:      c.GetString("request_id"),
			UserId:         user.Id,
			TokenId:        token.Id,
			TokenName:      token.Name,
			ChannelId:      channel.Id,
			ChannelName:    channel.Name,
			ChannelType:    channel.Type,
			RequestedModel: req.Model,
			MappedModel:    mappedModel,
			UserGroup:      user.Group,
			ClientIP:       c.ClientIP(),
		}
		sanitizeResult, sanitizeErr := qingyuan.ApplyRequest(c.Request.Context(), attemptReq, reqCtx, policy)
		if sanitizeErr != nil {
			lastError = sanitizeErr
			continue
		}
		if sanitizeResult.Blocked {
			msg := sanitizeResult.Message
			if msg == "" {
				msg = "请求被安全策略拒绝，如有疑问请联系管理员"
			}
			go RecordFailLog(c, token, mappedModel, msg)
			sendResponsesError(c, http.StatusBadRequest, "context_sanitization_blocked", msg)
			return
		}

		adaptor := adapter.GetAdaptor(channel.Type, channel)
		converted, err := adaptor.ConvertRequest(c, attemptReq)
		if err != nil {
			lastError = err
			continue
		}

		resp, err := adaptor.DoRequest(c, channel, converted)
		if err != nil {
			lastError = err
			continue
		}

		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastError = fmt.Errorf("channel %s error %d: %s", channel.Name, resp.StatusCode, string(respBody))
			continue
		}
		defer resp.Body.Close()

		// Convert response to Responses API format
		isStream := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")
		responseID := fmt.Sprintf("resp_%s", common.GetUUID())

		respCtx := qingyuan.ResponseContext{
			RequestContext:   reqCtx,
			UserRequestedAds: qingyuan.IsUserRequestingAds(attemptReq),
			RequestTools:     attemptReq.Tools,
		}

		var usage *dto.Usage
		var completionText string
		if req.Stream && isStream {
			usage, completionText, err = streamChatToResponses(c, resp, responseID, req.Model, policy, respCtx)
		} else if isStream {
			usage, completionText, err = streamToNormalResponses(c, resp, responseID, req.Model, policy, respCtx)
		} else {
			usage, completionText, err = normalChatToResponses(c, resp, responseID, req.Model, policy, respCtx)
		}

		if usage != nil {
			c.Set("channel_id", channel.Id)
			c.Set("is_stream", req.Stream)
			c.Set("use_time", int(time.Since(startTime).Milliseconds()))
			go RecordConsumeLog(c, token, mappedModels[i], usage.PromptTokens, usage.CompletionTokens)
			go upsertChannelAffinity(defaultChannelAffinityRule, user.Group, getChannelAffinityKeyFP(tokenKey), channel.Id, mappedModels[i], usage.PromptTokens, usage.CompletionTokens, usage.CachedTokens)

			// 宸汐玄鉴：异步行为分析
			if xuanjian.IsEnabled() {
				promptSnippet := xuanjian.CollectPromptSnippet(attemptReq.Messages, attemptReq.Prompt, 2000)
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
					Messages:          attemptReq.Messages,
					StatusCode:        resp.StatusCode,
					QYFindings:        xuanjian.ExtractQYFindingTypes(sanitizeResult.Findings),
					QYRiskScore:       sanitizeResult.RiskScore,
				})
			}
		}
		return
	}

	errMsg := fmt.Sprintf("all channels failed, last error: %v", lastError)
	go RecordFailLog(c, token, req.Model, errMsg)
	sendResponsesError(c, http.StatusServiceUnavailable, "server_error", errMsg)
}

// RelayResponsesCompact provides compatibility for POST /v1/responses/compact.
// Current implementation keeps the same behavior as /v1/responses.
func RelayResponsesCompact(c *gin.Context) {
	RelayResponses(c)
}

func sendResponsesError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"message": message,
			"type":    "invalid_request_error",
			"code":    code,
		},
	})
}

// responsesToChatRequest converts Responses API request to Chat Completions format
func responsesToChatRequest(req *ResponsesRequest) *dto.OpenAIRequest {
	openAIReq := &dto.OpenAIRequest{
		Model:       req.Model,
		Stream:      req.Stream,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Tools:       req.Tools,
		ToolChoice:  req.ToolChoice,
	}

	// Add instructions as system message
	if req.Instructions != "" {
		openAIReq.Messages = append(openAIReq.Messages, map[string]interface{}{
			"role": "system", "content": req.Instructions,
		})
	}

	// Parse input — can be string or array of message objects
	var inputStr string
	if json.Unmarshal(req.Input, &inputStr) == nil {
		openAIReq.Messages = append(openAIReq.Messages, map[string]interface{}{
			"role": "user", "content": inputStr,
		})
		return openAIReq
	}

	var inputMsgs []map[string]interface{}
	if json.Unmarshal(req.Input, &inputMsgs) == nil {
		for _, msg := range inputMsgs {
			openAIReq.Messages = append(openAIReq.Messages, msg)
		}
	}

	return openAIReq
}

// normalChatToResponses converts a non-streaming OpenAI chat response to Responses API format
func normalChatToResponses(c *gin.Context, resp *http.Response, responseID, modelName string, policy qingyuan.ResolvedPolicy, rc qingyuan.ResponseContext) (*dto.Usage, string, error) {
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

	var chatResp dto.ChatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, "", err
	}

	// Build output items
	var outputItems []map[string]interface{}
	completionText := ""
	if len(chatResp.Choices) > 0 {
		msg := chatResp.Choices[0].Message
		completionText = msg.Content
		outputItem := map[string]interface{}{
			"type":   "message",
			"id":     fmt.Sprintf("msg_%s", common.GetUUID()),
			"status": "completed",
			"role":   "assistant",
			"content": []map[string]interface{}{
				{"type": "output_text", "text": msg.Content},
			},
		}
		outputItems = append(outputItems, outputItem)
	}

	result := gin.H{
		"id":     responseID,
		"object": "response",
		"status": "completed",
		"model":  modelName,
		"output": outputItems,
		"usage": gin.H{
			"input_tokens":  chatResp.Usage.PromptTokens,
			"output_tokens": chatResp.Usage.CompletionTokens,
			"total_tokens":  chatResp.Usage.TotalTokens,
		},
	}

	c.JSON(http.StatusOK, result)
	return &chatResp.Usage, completionText, nil
}

// streamChatToResponses converts streaming OpenAI SSE to Responses API SSE format，
// 有清源策略时把文本增量喂给 TailBufferCore，收尾事件永远等 Finalize() 放完尾巴才发，
// 避免"尾部被拦下来，但收尾帧已经先发出去了"这种协议错序～
func streamChatToResponses(c *gin.Context, resp *http.Response, responseID, modelName string, policy qingyuan.ResolvedPolicy, rc qingyuan.ResponseContext) (*dto.Usage, string, error) {
	common.SetEventStreamHeaders(c)

	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(splitSSE)

	usage := &dto.Usage{}
	started := false
	msgID := fmt.Sprintf("msg_%s", common.GetUUID())
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
			writeResponsesSSE(c, "response.created", gin.H{
				"response": gin.H{"id": responseID, "object": "response", "status": "in_progress", "model": modelName},
			})
			writeResponsesSSE(c, "response.in_progress", gin.H{
				"response": gin.H{"id": responseID, "object": "response", "status": "in_progress", "model": modelName},
			})
			writeResponsesSSE(c, "response.output_item.added", gin.H{
				"output_index": 0,
				"item":         gin.H{"type": "message", "id": msgID, "role": "assistant", "status": "in_progress"},
			})
			writeResponsesSSE(c, "response.content_part.added", gin.H{
				"output_index":  0,
				"content_index": 0,
				"part":          gin.H{"type": "output_text", "text": ""},
			})
			started = true
		}

		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta

		if delta.Content != "" {
			if core == nil {
				writeResponsesSSE(c, "response.output_text.delta", gin.H{
					"output_index":  0,
					"content_index": 0,
					"delta":         delta.Content,
				})
			} else {
				rendered := renderResponsesSSE("response.output_text.delta", gin.H{
					"output_index":  0,
					"content_index": 0,
					"delta":         delta.Content,
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
	blocked := false
	if core != nil {
		released, fullText, findings := core.Finalize(rc.RequestTools)
		for _, r := range released {
			c.Writer.Write(r.([]byte))
		}
		c.Writer.Flush()
		completionText = fullText
		blocked = core.Blocked()
		qingyuan.RecordResponseFindings(rc, policy, findings, false)
	}

	if finishReason != "" && !blocked {
		writeResponsesSSE(c, "response.output_text.done", gin.H{
			"output_index":  0,
			"content_index": 0,
			"text":          completionText,
		})
		writeResponsesSSE(c, "response.content_part.done", gin.H{
			"output_index":  0,
			"content_index": 0,
			"part":          gin.H{"type": "output_text", "text": completionText},
		})
		writeResponsesSSE(c, "response.output_item.done", gin.H{
			"output_index": 0,
			"item": gin.H{
				"type": "message", "id": msgID, "role": "assistant", "status": "completed",
				"content": []gin.H{{"type": "output_text", "text": completionText}},
			},
		})
		writeResponsesSSE(c, "response.completed", gin.H{
			"response": gin.H{
				"id": responseID, "object": "response", "status": "completed", "model": modelName,
				"output": []gin.H{{
					"type": "message", "id": msgID, "role": "assistant", "status": "completed",
					"content": []gin.H{{"type": "output_text", "text": completionText}},
				}},
				"usage": gin.H{
					"input_tokens":  usage.PromptTokens,
					"output_tokens": usage.CompletionTokens,
					"total_tokens":  usage.TotalTokens,
				},
			},
		})
	}

	return usage, completionText, nil
}

func writeResponsesSSE(c *gin.Context, eventType string, data interface{}) {
	c.Writer.Write(renderResponsesSSE(eventType, data))
	c.Writer.Flush()
}

// renderResponsesSSE 只渲染成字节，不负责写出去——供需要"先攒住再决定发不发"的场景使用～
func renderResponsesSSE(eventType string, data interface{}) []byte {
	jsonBytes, _ := json.Marshal(data)
	var buf bytes.Buffer
	buf.WriteString("event: " + eventType + "\n")
	buf.WriteString("data: " + string(jsonBytes) + "\n\n")
	return buf.Bytes()
}

// streamToNormalResponses collects a streaming OpenAI response into a single Responses API JSON response，
// 先把整段流收完整聚合，再统一跑一遍检测/清理——这是最安全的一种场景，什么都还没发给客户端～
func streamToNormalResponses(c *gin.Context, resp *http.Response, responseID, modelName string, policy qingyuan.ResolvedPolicy, rc qingyuan.ResponseContext) (*dto.Usage, string, error) {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(func(data []byte, atEOF bool) (int, []byte, error) {
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
	})

	usage := &dto.Usage{}
	var textBuf strings.Builder

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
		if len(chunk.Choices) > 0 {
			textBuf.WriteString(chunk.Choices[0].Delta.Content)
		}
	}

	completionText := textBuf.String()
	if qingyuan.IsEnabled(policy) {
		cleaned, blocked, blockMsg, findings := qingyuan.AnalyzeText(completionText, nil, rc, policy)
		if blocked {
			return nil, "", fmt.Errorf("response blocked by qingyuan: %s", blockMsg)
		}
		if len(findings) > 0 {
			qingyuan.RecordResponseFindings(rc, policy, findings, cleaned != completionText)
		}
		completionText = cleaned
	}

	msgID := fmt.Sprintf("msg_%s", common.GetUUID())
	result := gin.H{
		"id": responseID, "object": "response", "status": "completed", "model": modelName,
		"output": []gin.H{{
			"type": "message", "id": msgID, "role": "assistant", "status": "completed",
			"content": []gin.H{{"type": "output_text", "text": completionText}},
		}},
		"usage": gin.H{
			"input_tokens":  usage.PromptTokens,
			"output_tokens": usage.CompletionTokens,
			"total_tokens":  usage.TotalTokens,
		},
	}

	c.JSON(http.StatusOK, result)
	return usage, completionText, nil
}
