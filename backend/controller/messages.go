package controller

import (
	"STfreApi/adapter"
	"STfreApi/adapter/claude"
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

// RelayMessages handles Anthropic Messages API format requests (POST /v1/messages)
// This enables Claude Code CLI, and other Anthropic-native clients to use the gateway.
func RelayMessages(c *gin.Context) {
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
	channels, mappedModels, err := SelectChannel(claudeReq.Model, user.Group)
	if err != nil {
		sendClaudeError(c, http.StatusServiceUnavailable, "api_error",
			fmt.Sprintf("no available channel for model: %s", claudeReq.Model))
		return
	}

	// 7. Try channels
	var lastError error
	for i, channel := range channels {
		mappedModel := mappedModels[i]
		claudeReq.Model = mappedModel
		channel.Key = service.GetNextKey(channel.Key)

		if channel.Type == model.ChannelTypeAnthropic {
			// Direct passthrough to Claude upstream
			usage, err := relayClaudeDirect(c, channel, &claudeReq, bodyBytes, mappedModel)
			if err != nil {
				lastError = err
				continue
			}
			if usage != nil {
				go RecordConsumeLog(c, token, mappedModel, usage.PromptTokens, usage.CompletionTokens)
			}
			return
		}

		// Non-Claude channel: convert to OpenAI format, relay, convert response back
		usage, err := relayClaudeViaOpenAI(c, channel, &claudeReq, token)
		if err != nil {
			lastError = err
			continue
		}
		if usage != nil {
			go RecordConsumeLog(c, token, mappedModel, usage.PromptTokens, usage.CompletionTokens)
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
func relayClaudeDirect(c *gin.Context, channel *model.Channel, req *claude.ClaudeRequest, rawBody []byte, mappedModel string) (*dto.Usage, error) {
	// Re-marshal with mapped model
	req.Model = mappedModel
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	baseURL := channel.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	httpReq, err := http.NewRequest("POST", baseURL+"/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
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
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upstream error %d: %s", resp.StatusCode, string(body))
	}

	// Stream passthrough or normal passthrough
	isStream := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")
	if isStream {
		return streamPassthroughClaude(c, resp)
	}
	return normalPassthroughClaude(c, resp)
}

// normalPassthroughClaude passes a non-streaming Claude response directly to the client
func normalPassthroughClaude(c *gin.Context, resp *http.Response) (*dto.Usage, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Extract usage before forwarding
	var claudeResp claude.ClaudeResponse
	usage := &dto.Usage{}
	if json.Unmarshal(body, &claudeResp) == nil {
		usage.PromptTokens = claudeResp.Usage.InputTokens
		usage.CompletionTokens = claudeResp.Usage.OutputTokens
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	// Forward response as-is
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
	return usage, nil
}

// streamPassthroughClaude passes a streaming Claude response directly to the client
func streamPassthroughClaude(c *gin.Context, resp *http.Response) (*dto.Usage, error) {
	common.SetEventStreamHeaders(c)
	scanner := bufio.NewScanner(resp.Body)
	usage := &dto.Usage{}

	for scanner.Scan() {
		line := scanner.Text()
		c.Writer.WriteString(line + "\n")
		c.Writer.Flush()

		// Extract usage from stream events
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
			}
		}
	}

	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	return usage, nil
}

// relayClaudeViaOpenAI converts Anthropic request to OpenAI format,
// sends to a non-Claude channel, then converts the response back to Anthropic format.
func relayClaudeViaOpenAI(c *gin.Context, channel *model.Channel, req *claude.ClaudeRequest, token *model.Token) (*dto.Usage, error) {
	// Convert Claude messages to OpenAI messages
	openAIReq := claudeToOpenAIRequest(req)

	// Use the adapter system
	adaptor := adapter.GetAdaptor(channel.Type)
	converted, err := adaptor.ConvertRequest(c, openAIReq)
	if err != nil {
		return nil, err
	}

	resp, err := adaptor.DoRequest(c, channel, converted)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upstream error %d: %s", resp.StatusCode, string(body))
	}

	// Read the OpenAI response and convert back to Claude format
	isStream := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")
	if isStream {
		return streamOpenAI2Claude(c, resp, req.Model)
	}
	return normalOpenAI2Claude(c, resp, req.Model)
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
func normalOpenAI2Claude(c *gin.Context, resp *http.Response, modelName string) (*dto.Usage, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var openAIResp dto.ChatCompletionResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return nil, err
	}

	// Build Claude content blocks
	var content []claude.ContentBlock
	if len(openAIResp.Choices) > 0 {
		msg := openAIResp.Choices[0].Message
		if msg.Content != "" {
			content = append(content, claude.ContentBlock{Type: "text", Text: msg.Content})
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
	return &openAIResp.Usage, nil
}

// streamOpenAI2Claude converts streaming OpenAI SSE to Claude SSE format
func streamOpenAI2Claude(c *gin.Context, resp *http.Response, modelName string) (*dto.Usage, error) {
	common.SetEventStreamHeaders(c)
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(splitSSE)

	usage := &dto.Usage{}
	msgID := fmt.Sprintf("msg_%s", common.GetUUID())
	started := false

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
			blockDelta := map[string]interface{}{
				"type":  "content_block_delta",
				"index": 0,
				"delta": map[string]string{"type": "text_delta", "text": delta.Content},
			}
			writeClaudeSSE(c, "content_block_delta", blockDelta)
		}

		if chunk.Choices[0].FinishReason != "" {
			// content_block_stop
			writeClaudeSSE(c, "content_block_stop", map[string]interface{}{
				"type": "content_block_stop", "index": 0,
			})
			// message_delta
			writeClaudeSSE(c, "message_delta", map[string]interface{}{
				"type":  "message_delta",
				"delta": map[string]string{"stop_reason": openAIStopToClaude(chunk.Choices[0].FinishReason)},
				"usage": map[string]int{"output_tokens": usage.CompletionTokens},
			})
			// message_stop
			writeClaudeSSE(c, "message_stop", map[string]interface{}{"type": "message_stop"})
		}
	}

	return usage, nil
}

func writeClaudeSSE(c *gin.Context, eventType string, data interface{}) {
	jsonBytes, _ := json.Marshal(data)
	c.Writer.WriteString("event: " + eventType + "\n")
	c.Writer.WriteString("data: " + string(jsonBytes) + "\n\n")
	c.Writer.Flush()
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
