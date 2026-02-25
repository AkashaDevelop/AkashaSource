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
		claudeReq.MaxTokens = 4096 // Default max tokens
	}

	// Messages conversion
	var systemPrompt string
	for _, msg := range request.Messages {
		if m, ok := msg.(map[string]interface{}); ok {
			role := m["role"].(string)
			content := m["content"].(string)
			if role == "system" {
				systemPrompt += content + "\n"
			} else {
				claudeReq.Messages = append(claudeReq.Messages, ClaudeMessage{
					Role:    role,
					Content: content,
				})
			}
		}
	}
	claudeReq.System = strings.TrimSpace(systemPrompt)

	return &claudeReq, nil
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
	// Ensure no trailing slash
	baseUrl = strings.TrimSuffix(baseUrl, "/")

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/messages", baseUrl), bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-api-key", channel.Key)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	client := &http.Client{}
	return client.Do(req)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *model.Token) (usage *dto.Usage, err error) {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("provider error: %s", string(body))
	}

	// Handle Stream
	// We need to check if the request was stream. But `DoResponse` signature doesn't have request info.
	// We can check Content-Type or pass request info.
	// Or we can peek the body? No.
	// In my interface design, `ConvertRequest` returned the converted request which has `Stream` field.
	// But `DoResponse` doesn't receive it.
	// Let's assume we can detect stream from response header "text/event-stream"

	isStream := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")

	if isStream {
		return a.streamHandler(c, resp)
	} else {
		return a.normalHandler(c, resp)
	}
}

func (a *Adaptor) normalHandler(c *gin.Context, resp *http.Response) (*dto.Usage, error) {
	var claudeResp ClaudeResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return nil, err
	}

	content := ""
	if len(claudeResp.Content) > 0 {
		content = claudeResp.Content[0].Text
	}

	openaiResp := dto.ChatCompletionResponse{
		ID:      claudeResp.ID,
		Object:  "chat.completion",
		Created: common.GetTimestamp(),
		Model:   claudeResp.Model,
		Choices: []struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			Index        int    `json:"index"`
			FinishReason string `json:"finish_reason"`
		}{
			{
				Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{
					Role:    "assistant",
					Content: content,
				},
				Index:        0,
				FinishReason: stopReasonClaude2OpenAI(claudeResp.StopReason),
			},
		},
		Usage: dto.Usage{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
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
			}
			// Send initial empty chunk with role? OpenAI usually sends role in first chunk
			sendStreamResponse(c, id, modelName, "", "assistant", "")

		case "content_block_delta":
			if event.Delta != nil {
				sendStreamResponse(c, id, modelName, event.Delta.Text, "", "")
			}

		case "message_delta":
			if event.Usage != nil {
				usage.CompletionTokens = event.Usage.OutputTokens
			}
			if event.Delta != nil && event.Delta.StopReason != "" {
				sendStreamResponse(c, id, modelName, "", "", stopReasonClaude2OpenAI(event.Delta.StopReason))
			}

		case "message_stop":
			sendStreamResponse(c, id, modelName, "", "", "") // [DONE] handled by caller usually, but here we should send DONE
		}
	}

	c.Writer.WriteString("data: [DONE]\n\n")
	c.Writer.Flush()

	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	return usage, nil
}

func sendStreamResponse(c *gin.Context, id, model, content, role, finishReason string) {
	resp := dto.ChatCompletionStreamResponse{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: common.GetTimestamp(),
		Model:   model,
		Choices: []struct {
			Delta struct {
				Content string `json:"content"`
				Role    string `json:"role,omitempty"`
			} `json:"delta"`
			Index        int    `json:"index"`
			FinishReason string `json:"finish_reason"`
		}{
			{
				Delta: struct {
					Content string `json:"content"`
					Role    string `json:"role,omitempty"`
				}{
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

func stopReasonClaude2OpenAI(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	default:
		return reason
	}
}
