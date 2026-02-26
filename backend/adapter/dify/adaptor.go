package dify

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
	return "dify"
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

// DifyChatRequest is the Dify /v1/chat-messages format
type DifyChatRequest struct {
	Inputs         map[string]interface{} `json:"inputs"`
	Query          string                 `json:"query"`
	ResponseMode   string                 `json:"response_mode"`
	ConversationId string                 `json:"conversation_id,omitempty"`
	User           string                 `json:"user"`
}

func (a *Adaptor) ConvertRequest(c *gin.Context, request *dto.OpenAIRequest) (any, error) {
	difyReq := DifyChatRequest{
		Inputs:       make(map[string]interface{}),
		ResponseMode: "blocking",
		User:         "akasha-user",
	}
	if request.Stream {
		difyReq.ResponseMode = "streaming"
	}

	// Extract last user message as query
	for i := len(request.Messages) - 1; i >= 0; i-- {
		if m, ok := request.Messages[i].(map[string]interface{}); ok {
			role, _ := m["role"].(string)
			if role == "user" {
				if content, ok := m["content"].(string); ok {
					difyReq.Query = content
				}
				break
			}
		}
	}

	return &difyReq, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, channel *model.Channel, request any) (*http.Response, error) {
	difyReq := request.(*DifyChatRequest)
	reqBody, err := json.Marshal(difyReq)
	if err != nil {
		return nil, err
	}

	baseURL := channel.BaseURL
	if baseURL == "" {
		baseURL = BaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	targetURL := fmt.Sprintf("%s/v1/chat-messages", baseURL)
	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+channel.Key)
	req.Header.Set("Content-Type", "application/json")
	common.ApplyHeaders(req, channel.Headers)

	client := common.NewHTTPClient(channel.Proxy)
	return client.Do(req)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *model.Token) (*dto.Usage, error) {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("dify error: %s", string(body))
	}

	isStream := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")
	if isStream {
		return a.streamHandler(c, resp)
	}
	return a.normalHandler(c, resp)
}

func (a *Adaptor) normalHandler(c *gin.Context, resp *http.Response) (*dto.Usage, error) {
	body, _ := io.ReadAll(resp.Body)

	var difyResp struct {
		Answer string `json:"answer"`
		Metadata struct {
			Usage struct {
				TotalTokens      int `json:"total_tokens"`
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(body, &difyResp); err != nil {
		return nil, err
	}

	openaiResp := dto.ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%s", common.GetUUID()),
		Object:  "chat.completion",
		Created: common.GetTimestamp(),
		Model:   "dify",
		Choices: []dto.Choice{{
			Message:      dto.ChatMessage{Role: "assistant", Content: difyResp.Answer},
			Index:        0,
			FinishReason: "stop",
		}},
		Usage: dto.Usage{
			PromptTokens:     difyResp.Metadata.Usage.PromptTokens,
			CompletionTokens: difyResp.Metadata.Usage.CompletionTokens,
			TotalTokens:      difyResp.Metadata.Usage.TotalTokens,
		},
	}

	c.JSON(http.StatusOK, openaiResp)
	return &openaiResp.Usage, nil
}

func (a *Adaptor) streamHandler(c *gin.Context, resp *http.Response) (*dto.Usage, error) {
	common.SetEventStreamHeaders(c)

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0), 1024*1024)

	id := fmt.Sprintf("chatcmpl-%s", common.GetUUID())
	usage := &dto.Usage{}

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var event struct {
			Event  string `json:"event"`
			Answer string `json:"answer"`
		}
		if json.Unmarshal([]byte(data), &event) != nil {
			continue
		}

		if event.Event == "message" && event.Answer != "" {
			chunk := dto.ChatCompletionStreamResponse{
				ID:      id,
				Object:  "chat.completion.chunk",
				Created: common.GetTimestamp(),
				Model:   "dify",
				Choices: []dto.StreamChoice{{
					Delta: dto.StreamDelta{Content: event.Answer},
				}},
			}
			jsonBody, _ := json.Marshal(chunk)
			c.Writer.WriteString("data: " + string(jsonBody) + "\n\n")
			c.Writer.Flush()
		}

		if event.Event == "message_end" {
			chunk := dto.ChatCompletionStreamResponse{
				ID:      id,
				Object:  "chat.completion.chunk",
				Created: common.GetTimestamp(),
				Model:   "dify",
				Choices: []dto.StreamChoice{{
					FinishReason: "stop",
				}},
			}
			jsonBody, _ := json.Marshal(chunk)
			c.Writer.WriteString("data: " + string(jsonBody) + "\n\n")
			c.Writer.Flush()
		}
	}

	c.Writer.WriteString("data: [DONE]\n\n")
	c.Writer.Flush()

	return usage, nil
}
