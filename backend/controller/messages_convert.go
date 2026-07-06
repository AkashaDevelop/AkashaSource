package controller

import (
	"STfreApi/adapter"
	"STfreApi/adapter/claude"
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
	"STfreApi/service/qingyuan"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

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
