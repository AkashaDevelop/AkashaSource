package claude

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

	"github.com/gin-gonic/gin"
)

type Adaptor struct{}

func (a *Adaptor) GetChannelName() string {
	return "claude"
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) ConvertRequest(c *gin.Context, request *dto.OpenAIRequest) (any, error) {
	claudeReq := ClaudeRequest{
		Model:     request.Model,
		MaxTokens: request.MaxTokens,
		Stream:    request.Stream,
	}
	if claudeReq.MaxTokens == 0 {
		claudeReq.MaxTokens = 4096
	}

	// Convert tools
	if len(request.Tools) > 0 {
		claudeReq.Tools = convertTools(request.Tools)
	}
	if request.ToolChoice != nil {
		raw, _ := json.Marshal(request.ToolChoice)
		claudeReq.ToolChoice = raw
	}

	// Convert messages
	var systemParts []string
	for _, msg := range request.Messages {
		m, ok := msg.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := m["role"].(string)
		if role == "system" {
			systemParts = append(systemParts, extractTextContent(m["content"]))
			continue
		}
		cm := ClaudeMessage{Role: role}
		cm.Content = convertMessageContent(m["content"], role)
		claudeReq.Messages = append(claudeReq.Messages, cm)
	}

	if len(systemParts) > 0 {
		sysText := strings.Join(systemParts, "\n")
		raw, _ := json.Marshal(sysText)
		claudeReq.System = raw
	}

	// Inject thinking mode from suffix
	if request.ThinkingMode && claudeReq.Thinking == nil {
		claudeReq.Thinking = &ClaudeThinking{
			Type:         "enabled",
			BudgetTokens: 10000,
		}
		// Thinking mode requires stream and no temperature
		claudeReq.Stream = true
		claudeReq.Temperature = nil
	}

	return &claudeReq, nil
}

// extractTextContent gets plain text from message content (string or array)
func extractTextContent(content interface{}) string {
	if s, ok := content.(string); ok {
		return s
	}
	if arr, ok := content.([]interface{}); ok {
		var parts []string
		for _, item := range arr {
			if m, ok := item.(map[string]interface{}); ok {
				if t, ok := m["text"].(string); ok {
					parts = append(parts, t)
				}
			}
		}
		return strings.Join(parts, "\n")
	}
	return fmt.Sprint(content)
}

// convertMessageContent converts OpenAI message content to Claude format
func convertMessageContent(content interface{}, role string) json.RawMessage {
	// Simple string content
	if s, ok := content.(string); ok {
		raw, _ := json.Marshal(s)
		return raw
	}

	// Array content (vision, tool_result, etc.)
	if arr, ok := content.([]interface{}); ok {
		var blocks []ContentBlock
		for _, item := range arr {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			typ, _ := m["type"].(string)
			switch typ {
			case "text":
				text, _ := m["text"].(string)
				blocks = append(blocks, ContentBlock{Type: "text", Text: text})
			case "image_url":
				if urlObj, ok := m["image_url"].(map[string]interface{}); ok {
					url, _ := urlObj["url"].(string)
					if strings.HasPrefix(url, "data:") {
						// data:image/jpeg;base64,xxxx
						parts := strings.SplitN(url, ",", 2)
						mediaType := "image/jpeg"
						if len(parts) == 2 {
							meta := strings.TrimPrefix(parts[0], "data:")
							meta = strings.TrimSuffix(meta, ";base64")
							mediaType = meta
							blocks = append(blocks, ContentBlock{
								Type: "image",
								Source: &ImageSource{
									Type:      "base64",
									MediaType: mediaType,
									Data:      parts[1],
								},
							})
						}
					}
				}
			case "tool_use":
				id, _ := m["id"].(string)
				name, _ := m["name"].(string)
				inputRaw, _ := json.Marshal(m["input"])
				blocks = append(blocks, ContentBlock{
					Type:  "tool_use",
					ID:    id,
					Name:  name,
					Input: inputRaw,
				})
			case "tool_result":
				toolUseID, _ := m["tool_use_id"].(string)
				contentRaw, _ := json.Marshal(m["content"])
				blocks = append(blocks, ContentBlock{
					Type:      "tool_result",
					ToolUseID: toolUseID,
					Content:   contentRaw,
				})
			}
		}
		raw, _ := json.Marshal(blocks)
		return raw
	}

	raw, _ := json.Marshal(fmt.Sprint(content))
	return raw
}

// convertTools converts OpenAI function tools to Claude tool format
func convertTools(tools []any) []ClaudeTool {
	var result []ClaudeTool
	for _, t := range tools {
		m, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		fn, ok := m["function"].(map[string]interface{})
		if !ok {
			continue
		}
		ct := ClaudeTool{
			Name:        getString(fn, "name"),
			Description: getString(fn, "description"),
		}
		if params, ok := fn["parameters"]; ok {
			ct.InputSchema, _ = json.Marshal(params)
		}
		result = append(result, ct)
	}
	return result
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func (a *Adaptor) DoRequest(c *gin.Context, channel *model.Channel, request any) (*http.Response, error) {
	claudeReq := request.(*ClaudeRequest)
	reqBody, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, err
	}

	baseUrl := channel.BaseURL
	if baseUrl == "" {
		baseUrl = BaseURL
	}
	baseUrl = strings.TrimSuffix(baseUrl, "/")

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/messages", baseUrl), bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-api-key", channel.Key)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")
	// Enable beta features for tool_use and thinking
	req.Header.Set("anthropic-beta", "tools-2024-04-04,prompt-caching-2024-07-31")
	common.ApplyHeaders(req, channel.Headers)

	client := common.NewHTTPClient(channel.Proxy)
	return client.Do(req)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *model.Token) (usage *dto.Usage, err error) {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("provider error: %s", string(body))
	}

	isStream := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")
	if isStream {
		return a.streamHandler(c, resp)
	}
	return a.normalHandler(c, resp)
}

func (a *Adaptor) normalHandler(c *gin.Context, resp *http.Response) (*dto.Usage, error) {
	var claudeResp ClaudeResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return nil, err
	}

	// Build OpenAI-format response with tool_use and thinking support
	content := ""
	var toolCalls []dto.ToolCall
	for _, block := range claudeResp.Content {
		switch block.Type {
		case "thinking":
			if common.ThinkingToContent && block.Thinking != "" {
				content += "<thinking>\n" + block.Thinking + "\n</thinking>\n"
			}
		case "text":
			content += block.Text
		case "tool_use":
			toolCalls = append(toolCalls, dto.ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: dto.FunctionCall{
					Name:      block.Name,
					Arguments: string(block.Input),
				},
			})
		}
	}

	msg := dto.ChatMessage{
		Role:    "assistant",
		Content: content,
	}
	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
	}

	openaiResp := dto.ChatCompletionResponse{
		ID:      claudeResp.ID,
		Object:  "chat.completion",
		Created: common.GetTimestamp(),
		Model:   claudeResp.Model,
		Choices: []dto.Choice{
			{
				Message:      msg,
				Index:        0,
				FinishReason: stopReasonClaude2OpenAI(claudeResp.StopReason),
			},
		},
		Usage: dto.Usage{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
			CachedTokens:     claudeResp.Usage.CacheReadInputTokens,
		},
	}

	c.JSON(http.StatusOK, openaiResp)
	return &openaiResp.Usage, nil
}

func (a *Adaptor) streamHandler(c *gin.Context, resp *http.Response) (*dto.Usage, error) {
	common.SetEventStreamHeaders(c)
	scanner := bufio.NewScanner(resp.Body)

	usage := &dto.Usage{}
	var id string
	var modelName string

	// Track tool_use and thinking state for streaming
	var currentToolID string
	var currentToolName string
	var toolArgsBuf strings.Builder
	var inThinkingBlock bool

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var event ClaudeStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "message_start":
			if event.Message != nil {
				id = event.Message.ID
				modelName = event.Message.Model
				usage.PromptTokens = event.Message.Usage.InputTokens
				usage.CachedTokens = event.Message.Usage.CacheReadInputTokens
			}
			sendStreamChunk(c, id, modelName, "", "assistant", "")

		case "content_block_start":
			if event.ContentBlock != nil {
				switch event.ContentBlock.Type {
				case "tool_use":
					currentToolID = event.ContentBlock.ID
					currentToolName = event.ContentBlock.Name
					toolArgsBuf.Reset()
				case "thinking":
					if common.ThinkingToContent {
						sendStreamChunk(c, id, modelName, "<thinking>\n", "", "")
						inThinkingBlock = true
					}
				}
			}

		case "content_block_delta":
			if event.Delta != nil {
				switch event.Delta.Type {
				case "text_delta":
					sendStreamChunk(c, id, modelName, event.Delta.Text, "", "")
				case "thinking_delta":
					if common.ThinkingToContent && event.Delta.Thinking != "" {
						sendStreamChunk(c, id, modelName, event.Delta.Thinking, "", "")
					}
				case "input_json_delta":
					toolArgsBuf.WriteString(event.Delta.PartialJSON)
				}
			}

		case "content_block_stop":
			if inThinkingBlock && common.ThinkingToContent {
				sendStreamChunk(c, id, modelName, "\n</thinking>\n", "", "")
				inThinkingBlock = false
			}
			if currentToolID != "" {
				sendStreamToolCall(c, id, modelName, currentToolID, currentToolName, toolArgsBuf.String())
				currentToolID = ""
				currentToolName = ""
				toolArgsBuf.Reset()
			}

		case "message_delta":
			if event.Usage != nil {
				usage.CompletionTokens = event.Usage.OutputTokens
			}
			if event.Delta != nil && event.Delta.StopReason != "" {
				sendStreamChunk(c, id, modelName, "", "", stopReasonClaude2OpenAI(event.Delta.StopReason))
			}

		case "message_stop":
			// handled below
		}
	}

	c.Writer.WriteString("data: [DONE]\n\n")
	c.Writer.Flush()

	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	return usage, nil
}

func sendStreamChunk(c *gin.Context, id, model, content, role, finishReason string) {
	resp := dto.ChatCompletionStreamResponse{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: common.GetTimestamp(),
		Model:   model,
		Choices: []dto.StreamChoice{
			{
				Delta: dto.StreamDelta{
					Content: content,
					Role:    role,
				},
				FinishReason: finishReason,
			},
		},
	}
	jsonBody, _ := json.Marshal(resp)
	c.Writer.WriteString("data: " + string(jsonBody) + "\n\n")
	c.Writer.Flush()
}

func sendStreamToolCall(c *gin.Context, id, model, toolID, toolName, args string) {
	resp := dto.ChatCompletionStreamResponse{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: common.GetTimestamp(),
		Model:   model,
		Choices: []dto.StreamChoice{
			{
				Delta: dto.StreamDelta{
					ToolCalls: []dto.ToolCall{
						{
							ID:   toolID,
							Type: "function",
							Function: dto.FunctionCall{
								Name:      toolName,
								Arguments: args,
							},
						},
					},
				},
			},
		},
	}
	jsonBody, _ := json.Marshal(resp)
	c.Writer.WriteString("data: " + string(jsonBody) + "\n\n")
	c.Writer.Flush()
}

func stopReasonClaude2OpenAI(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	default:
		return reason
	}
}
