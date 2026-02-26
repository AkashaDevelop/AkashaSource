package xunfei

import (
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Adaptor struct {
	Request   *dto.OpenAIRequest
	Model     string
	AppId     string
	ApiSecret string
	ApiKey    string
}

func (a *Adaptor) GetChannelName() string {
	return "xunfei"
}

func (a *Adaptor) GetModelList() []string {
	return []string{}
}

func (a *Adaptor) ConvertRequest(c *gin.Context, request *dto.OpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("请求为空")
	}
	a.Request = request
	a.Model = request.Model
	return requestOpenAI2Xunfei(request), nil
}

func (a *Adaptor) DoRequest(c *gin.Context, channel *model.Channel, request any) (*http.Response, error) {
	parts := strings.Split(channel.Key, "|")
	if len(parts) != 3 {
		return nil, errors.New("讯飞 API Key 格式错误")
	}
	a.AppId = parts[0]
	a.ApiSecret = parts[1]
	a.ApiKey = parts[2]
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
	}
	resp.Header.Set("Content-Type", "application/json")
	return resp, nil
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *model.Token) (*dto.Usage, error) {
	request := a.Request
	if request == nil {
		return nil, errors.New("请求为空")
	}
	version := getXunfeiAPIVersion(request.Model)
	domain := getXunfeiDomain(version)
	authURL, err := getXunfeiAuthUrl(a.ApiKey, a.ApiSecret, version)
	if err != nil {
		return nil, err
	}
	wsConn, _, err := websocket.DefaultDialer.Dial(authURL, nil)
	if err != nil {
		return nil, err
	}
	defer wsConn.Close()

	xunfeiReq := requestOpenAI2Xunfei(request)
	xunfeiReq.Payload.Message.Text = append([]XunfeiMessage{
		{Role: "system", Content: "你是一个有帮助的助手"},
	}, xunfeiReq.Payload.Message.Text...)
	xunfeiReq.Header.AppId = a.AppId
	xunfeiReq.Parameter.Chat.Domain = domain

	data, err := json.Marshal(xunfeiReq)
	if err != nil {
		return nil, err
	}
	if err := wsConn.WriteMessage(websocket.TextMessage, data); err != nil {
		return nil, err
	}

	if request.Stream {
		return xunfeiStreamResponse(c, wsConn, request.Model)
	}
	return xunfeiNormalResponse(c, wsConn, request.Model)
}

func requestOpenAI2Xunfei(request *dto.OpenAIRequest) *XunfeiChatRequest {
	msgs := make([]XunfeiMessage, 0, len(request.Messages))
	for _, message := range request.Messages {
		role, content := parseMessage(message)
		if role == "" {
			continue
		}
		msgs = append(msgs, XunfeiMessage{
			Role:    role,
			Content: content,
		})
	}
	req := XunfeiChatRequest{}
	req.Payload.Message.Text = msgs
	req.Parameter.Chat.Temperature = request.Temperature
	req.Parameter.Chat.TopK = int(request.TopP * 6)
	if req.Parameter.Chat.TopK == 0 {
		req.Parameter.Chat.TopK = 1
	}
	if request.MaxTokens > 0 {
		req.Parameter.Chat.MaxTokens = request.MaxTokens
	}
	return &req
}

func xunfeiStreamResponse(c *gin.Context, conn *websocket.Conn, modelName string) (*dto.Usage, error) {
	common.SetEventStreamHeaders(c)
	usage := &dto.Usage{}
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return usage, err
		}
		var resp XunfeiChatResponse
		if err := json.Unmarshal(message, &resp); err != nil {
			continue
		}
		if resp.Header.Code != 0 {
			return usage, errors.New(resp.Header.Message)
		}
		content := ""
		if len(resp.Payload.Choices.Text) > 0 {
			content = resp.Payload.Choices.Text[0].Content
		}
		streamResp := dto.ChatCompletionStreamResponse{
			Object:  "chat.completion.chunk",
			Created: common.GetTimestamp(),
			Model:   modelName,
		}
		streamResp.Choices = []dto.StreamChoice{
			{
				Delta: dto.StreamDelta{
					Content: content,
				},
				Index:        0,
				FinishReason: finishReason(resp.Payload.Choices.Status == 2),
			},
		}
		data, _ := json.Marshal(streamResp)
		c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
		c.Writer.Flush()
		if resp.Payload.Usage.Text.TotalTokens != 0 {
			usage.TotalTokens = resp.Payload.Usage.Text.TotalTokens
			usage.PromptTokens = resp.Payload.Usage.Text.PromptTokens
			usage.CompletionTokens = usage.TotalTokens - usage.PromptTokens
		}
		if resp.Payload.Choices.Status == 2 {
			c.Writer.Write([]byte("data: [DONE]\n\n"))
			c.Writer.Flush()
			break
		}
	}
	return usage, nil
}

func xunfeiNormalResponse(c *gin.Context, conn *websocket.Conn, modelName string) (*dto.Usage, error) {
	fullText := strings.Builder{}
	usage := &dto.Usage{}
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return usage, err
		}
		var resp XunfeiChatResponse
		if err := json.Unmarshal(message, &resp); err != nil {
			continue
		}
		if resp.Header.Code != 0 {
			return usage, errors.New(resp.Header.Message)
		}
		if len(resp.Payload.Choices.Text) > 0 {
			fullText.WriteString(resp.Payload.Choices.Text[0].Content)
		}
		if resp.Payload.Usage.Text.TotalTokens != 0 {
			usage.TotalTokens = resp.Payload.Usage.Text.TotalTokens
			usage.PromptTokens = resp.Payload.Usage.Text.PromptTokens
			usage.CompletionTokens = usage.TotalTokens - usage.PromptTokens
		}
		if resp.Payload.Choices.Status == 2 {
			break
		}
	}
	openaiResp := dto.ChatCompletionResponse{
		ID:      fmt.Sprintf("xunfei-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: common.GetTimestamp(),
		Model:   modelName,
		Usage:   *usage,
	}
	openaiResp.Choices = []dto.Choice{
		{
			Message: dto.ChatMessage{
				Role:    "assistant",
				Content: fullText.String(),
			},
			Index:        0,
			FinishReason: "stop",
		},
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(c.Writer).Encode(openaiResp)
	return usage, nil
}

func getXunfeiAPIVersion(model string) string {
	parts := strings.Split(model, "-")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return "v1.1"
}

func getXunfeiDomain(version string) string {
	switch version {
	case "v1.1":
		return "lite"
	case "v2.1":
		return "generalv2"
	case "v3.1":
		return "generalv3"
	case "v3.5":
		return "generalv3.5"
	case "v4.0":
		return "4.0Ultra"
	default:
		return "generalv3"
	}
}

func getXunfeiAuthUrl(apiKey, apiSecret, version string) (string, error) {
	baseURL := getXunfeiBaseUrl(version)
	host := strings.TrimPrefix(strings.TrimPrefix(baseURL, "wss://"), "https://")
	date := time.Now().UTC().Format(time.RFC1123)
	signatureOrigin := "host: " + host + "\n"
	signatureOrigin += "date: " + date + "\n"
	signatureOrigin += "GET /" + version + "/chat HTTP/1.1"
	h := hmac.New(sha256.New, []byte(apiSecret))
	h.Write([]byte(signatureOrigin))
	signatureSha := base64.StdEncoding.EncodeToString(h.Sum(nil))
	authOrigin := fmt.Sprintf("api_key=\"%s\", algorithm=\"hmac-sha256\", headers=\"host date request-line\", signature=\"%s\"", apiKey, signatureSha)
	authorization := base64.StdEncoding.EncodeToString([]byte(authOrigin))
	v := url.Values{}
	v.Add("authorization", authorization)
	v.Add("date", date)
	v.Add("host", host)
	return baseURL + "?" + v.Encode(), nil
}

func getXunfeiBaseUrl(version string) string {
	return fmt.Sprintf("wss://spark-api.xf-yun.com/%s/chat", version)
}

func parseMessage(message any) (string, string) {
	switch m := message.(type) {
	case map[string]interface{}:
		role := fmt.Sprint(m["role"])
		content := fmt.Sprint(m["content"])
		return role, content
	default:
		return "", ""
	}
}

func finishReason(done bool) string {
	if done {
		return "stop"
	}
	return ""
}

type XunfeiChatRequest struct {
	Header struct {
		AppId string `json:"app_id"`
	} `json:"header"`
	Parameter struct {
		Chat struct {
			Domain      string  `json:"domain"`
			Temperature float64 `json:"temperature,omitempty"`
			TopK        int     `json:"top_k,omitempty"`
			MaxTokens   int     `json:"max_tokens,omitempty"`
		} `json:"chat"`
	} `json:"parameter"`
	Payload struct {
		Message struct {
			Text []XunfeiMessage `json:"text"`
		} `json:"message"`
	} `json:"payload"`
}

type XunfeiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type XunfeiChatResponse struct {
	Header struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Sid     string `json:"sid"`
		Status  int    `json:"status"`
	} `json:"header"`
	Payload struct {
		Choices struct {
			Status int            `json:"status"`
			Text   []XunfeiChoice `json:"text"`
		} `json:"choices"`
		Usage struct {
			Text struct {
				PromptTokens int `json:"prompt_tokens"`
				TotalTokens  int `json:"total_tokens"`
			} `json:"text"`
		} `json:"usage"`
	} `json:"payload"`
}

type XunfeiChoice struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}
