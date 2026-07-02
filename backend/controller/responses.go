package controller

import (
	"STfreApi/adapter"
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/middleware"
	"STfreApi/model"
	"STfreApi/service"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

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
	channels, mappedModels, err := SelectChannelWithAffinity(openAIReq.Model, user.Group, tokenKey, defaultChannelAffinityRule)
	if err != nil {
		sendResponsesError(c, http.StatusServiceUnavailable, "server_error",
			fmt.Sprintf("no available channel for model: %s", openAIReq.Model))
		return
	}

	// 8. Try channels
	var lastError error
	for i, channel := range channels {
		openAIReq.Model = mappedModels[i]
		channel.Key = service.GetNextKey(channel.Key)

		adaptor := adapter.GetAdaptor(channel.Type, channel)
		converted, err := adaptor.ConvertRequest(c, openAIReq)
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

		var usage *dto.Usage
		if req.Stream && isStream {
			usage, err = streamChatToResponses(c, resp, responseID, req.Model)
		} else if isStream {
			usage, err = streamToNormalResponses(c, resp, responseID, req.Model)
		} else {
			usage, err = normalChatToResponses(c, resp, responseID, req.Model)
		}

		if usage != nil {
			go RecordConsumeLog(c, token, mappedModels[i], usage.PromptTokens, usage.CompletionTokens)
			go upsertChannelAffinity(defaultChannelAffinityRule, user.Group, getChannelAffinityKeyFP(tokenKey), channel.Id, mappedModels[i], usage.PromptTokens, usage.CompletionTokens, usage.CachedTokens)
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
func normalChatToResponses(c *gin.Context, resp *http.Response, responseID, modelName string) (*dto.Usage, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var chatResp dto.ChatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, err
	}

	// Build output items
	var outputItems []map[string]interface{}
	if len(chatResp.Choices) > 0 {
		msg := chatResp.Choices[0].Message
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
	return &chatResp.Usage, nil
}

// streamChatToResponses converts streaming OpenAI SSE to Responses API SSE format
func streamChatToResponses(c *gin.Context, resp *http.Response, responseID, modelName string) (*dto.Usage, error) {
	common.SetEventStreamHeaders(c)

	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(splitSSE)

	usage := &dto.Usage{}
	started := false
	msgID := fmt.Sprintf("msg_%s", common.GetUUID())
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
			textBuf.WriteString(delta.Content)
			writeResponsesSSE(c, "response.output_text.delta", gin.H{
				"output_index":  0,
				"content_index": 0,
				"delta":         delta.Content,
			})
		}

		if chunk.Choices[0].FinishReason != "" {
			writeResponsesSSE(c, "response.output_text.done", gin.H{
				"output_index":  0,
				"content_index": 0,
				"text":          textBuf.String(),
			})
			writeResponsesSSE(c, "response.content_part.done", gin.H{
				"output_index":  0,
				"content_index": 0,
				"part":          gin.H{"type": "output_text", "text": textBuf.String()},
			})
			writeResponsesSSE(c, "response.output_item.done", gin.H{
				"output_index": 0,
				"item": gin.H{
					"type": "message", "id": msgID, "role": "assistant", "status": "completed",
					"content": []gin.H{{"type": "output_text", "text": textBuf.String()}},
				},
			})
			writeResponsesSSE(c, "response.completed", gin.H{
				"response": gin.H{
					"id": responseID, "object": "response", "status": "completed", "model": modelName,
					"output": []gin.H{{
						"type": "message", "id": msgID, "role": "assistant", "status": "completed",
						"content": []gin.H{{"type": "output_text", "text": textBuf.String()}},
					}},
					"usage": gin.H{
						"input_tokens":  usage.PromptTokens,
						"output_tokens": usage.CompletionTokens,
						"total_tokens":  usage.TotalTokens,
					},
				},
			})
		}
	}

	return usage, nil
}

func writeResponsesSSE(c *gin.Context, eventType string, data interface{}) {
	jsonBytes, _ := json.Marshal(data)
	c.Writer.WriteString("event: " + eventType + "\n")
	c.Writer.WriteString("data: " + string(jsonBytes) + "\n\n")
	c.Writer.Flush()
}

// streamToNormalResponses collects a streaming OpenAI response into a single Responses API JSON response
func streamToNormalResponses(c *gin.Context, resp *http.Response, responseID, modelName string) (*dto.Usage, error) {
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

	msgID := fmt.Sprintf("msg_%s", common.GetUUID())
	result := gin.H{
		"id": responseID, "object": "response", "status": "completed", "model": modelName,
		"output": []gin.H{{
			"type": "message", "id": msgID, "role": "assistant", "status": "completed",
			"content": []gin.H{{"type": "output_text", "text": textBuf.String()}},
		}},
		"usage": gin.H{
			"input_tokens":  usage.PromptTokens,
			"output_tokens": usage.CompletionTokens,
			"total_tokens":  usage.TotalTokens,
		},
	}

	c.JSON(http.StatusOK, result)
	return usage, nil
}
