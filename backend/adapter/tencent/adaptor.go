package tencent

import (
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
	Action    string
	Version   string
	Timestamp int64
	Sign      string
	Model     string
	Stream    bool
}

func (a *Adaptor) GetChannelName() string {
	return "tencent"
}

func (a *Adaptor) GetModelList() []string {
	return []string{}
}

func (a *Adaptor) ConvertRequest(c *gin.Context, request *dto.OpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("请求为空")
	}
	a.Action = "ChatCompletions"
	a.Version = "2023-09-01"
	a.Timestamp = time.Now().Unix()
	a.Model = request.Model
	a.Stream = request.Stream
	return requestOpenAI2Tencent(request), nil
}

func (a *Adaptor) DoRequest(c *gin.Context, channel *model.Channel, request any) (*http.Response, error) {
	baseUrl := channel.BaseURL
	if baseUrl == "" {
		baseUrl = "https://hunyuan.tencentcloudapi.com"
	}
	baseUrl = strings.TrimSuffix(baseUrl, "/")
	fullUrl := baseUrl + "/"
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	appId, secretId, secretKey, err := parseTencentKey(channel.Key)
	if err != nil {
		return nil, err
	}
	_ = appId
	host := "hunyuan.tencentcloudapi.com"
	if u, err := url.Parse(baseUrl); err == nil && u.Host != "" {
		host = u.Host
	}
	signature := getTencentSign(host, a.Action, reqBody, a.Timestamp, secretId, secretKey)
	a.Sign = signature

	req, err := http.NewRequest("POST", fullUrl, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", signature)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", host)
	req.Header.Set("X-TC-Action", a.Action)
	req.Header.Set("X-TC-Timestamp", strconv.FormatInt(a.Timestamp, 10))
	req.Header.Set("X-TC-Version", a.Version)

	client := common.NewHTTPClient(channel.Proxy)
	return client.Do(req)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *model.Token) (*dto.Usage, error) {
	if a.Stream {
		return a.streamHandler(c, resp)
	}
	return a.normalHandler(c, resp)
}

func (a *Adaptor) normalHandler(c *gin.Context, resp *http.Response) (*dto.Usage, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tencentResp TencentChatResponseSB
	if err := json.Unmarshal(body, &tencentResp); err != nil {
		c.JSON(resp.StatusCode, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "响应解析失败",
			Type:    "server_error",
		}})
		return nil, err
	}
	if tencentResp.Response.Error.Code != 0 {
		c.JSON(resp.StatusCode, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: tencentResp.Response.Error.Message,
			Type:    "server_error",
			Code:    tencentResp.Response.Error.Code,
		}})
		return nil, errors.New(tencentResp.Response.Error.Message)
	}

	openaiResp := responseTencent2OpenAI(&tencentResp.Response, a.Model)
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_ = json.NewEncoder(c.Writer).Encode(openaiResp)
	return &openaiResp.Usage, nil
}

func (a *Adaptor) streamHandler(c *gin.Context, resp *http.Response) (*dto.Usage, error) {
	common.SetEventStreamHeaders(c)
	defer resp.Body.Close()

	usage := &dto.Usage{}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, "data:") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
		if line == "[DONE]" || line == "data: [DONE]" {
			c.Writer.Write([]byte("data: [DONE]\n\n"))
			c.Writer.Flush()
			break
		}
		var tencentResp TencentChatResponse
		if err := json.Unmarshal([]byte(line), &tencentResp); err != nil {
			continue
		}
		if tencentResp.Usage.TotalTokens != 0 {
			usage = &tencentResp.Usage
		}
		streamResp := streamResponseTencent2OpenAI(&tencentResp, a.Model)
		data, _ := json.Marshal(streamResp)
		c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
		c.Writer.Flush()
	}
	return usage, nil
}

func parseTencentKey(key string) (string, string, string, error) {
	parts := strings.Split(key, "|")
	if len(parts) != 3 {
		return "", "", "", errors.New("腾讯云 API Key 格式错误")
	}
	return parts[0], parts[1], parts[2], nil
}

func requestOpenAI2Tencent(request *dto.OpenAIRequest) *TencentChatRequest {
	msgs := make([]TencentMessage, 0, len(request.Messages))
	for _, message := range request.Messages {
		role, content := parseMessage(message)
		if role == "" {
			continue
		}
		msgs = append(msgs, TencentMessage{
			Role:    role,
			Content: content,
		})
	}
	stream := request.Stream
	req := TencentChatRequest{
		Messages:     msgs,
		Temperature:  request.Temperature,
		TopP:         request.TopP,
		MaxTokens:    request.MaxTokens,
		Model:        request.Model,
		Stream:       &stream,
		PresencePenalty:  request.PresencePenalty,
		FrequencyPenalty: request.FrequencyPenalty,
	}
	return &req
}

func responseTencent2OpenAI(response *TencentChatResponse, modelName string) *dto.ChatCompletionResponse {
	resp := dto.ChatCompletionResponse{
		ID:      response.Id,
		Object:  "chat.completion",
		Created: response.Created,
		Model:   modelName,
		Usage:   response.Usage,
	}
	resp.Choices = []dto.Choice{
		{
			Message: dto.ChatMessage{
				Role:    "assistant",
				Content: response.Choices[0].Message.Content,
			},
			Index:        0,
			FinishReason: "stop",
		},
	}
	return &resp
}

func streamResponseTencent2OpenAI(response *TencentChatResponse, modelName string) *dto.ChatCompletionStreamResponse {
	resp := dto.ChatCompletionStreamResponse{
		Object:  "chat.completion.chunk",
		Created: response.Created,
		Model:   modelName,
	}
	content := ""
	if len(response.Choices) > 0 {
		content = response.Choices[0].Delta.Content
	}
	resp.Choices = []dto.StreamChoice{
		{
			Delta: dto.StreamDelta{
				Content: content,
			},
			Index:        0,
			FinishReason: finishReason(len(response.Choices) > 0 && response.Choices[0].FinishReason != ""),
		},
	}
	return &resp
}

func finishReason(done bool) string {
	if done {
		return "stop"
	}
	return ""
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

func getTencentSign(host string, action string, payload []byte, timestamp int64, secretId string, secretKey string) string {
	action = strings.ToLower(action)
	algorithm := "TC3-HMAC-SHA256"
	service := "hunyuan"
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)

	canonicalHeaders := fmt.Sprintf("content-type:application/json\nhost:%s\n", host)
	signedHeaders := "content-type;host"
	hashedRequestPayload := sha256hex(payload)
	canonicalRequest := fmt.Sprintf("POST\n/\n\n%s\n%s\n%s", canonicalHeaders, signedHeaders, hashedRequestPayload)
	hashedCanonicalRequest := sha256hex([]byte(canonicalRequest))
	stringToSign := fmt.Sprintf("%s\n%d\n%s\n%s", algorithm, timestamp, credentialScope, hashedCanonicalRequest)

	secretDate := hmacSha256([]byte("TC3"+secretKey), date)
	secretService := hmacSha256(secretDate, service)
	secretSigning := hmacSha256(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSha256(secretSigning, stringToSign))
	return fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s", algorithm, secretId, credentialScope, signedHeaders, signature)
}

func sha256hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hmacSha256(key []byte, msg string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(msg))
	return mac.Sum(nil)
}

type TencentMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type TencentChatRequest struct {
	Messages         []TencentMessage `json:"Messages"`
	Temperature      float64          `json:"Temperature,omitempty"`
	TopP             float64          `json:"TopP,omitempty"`
	MaxTokens        int              `json:"MaxTokens,omitempty"`
	Model            string           `json:"Model"`
	Stream           *bool            `json:"Stream,omitempty"`
	PresencePenalty  float64          `json:"PresencePenalty,omitempty"`
	FrequencyPenalty float64          `json:"FrequencyPenalty,omitempty"`
}

type TencentChatResponseSB struct {
	Response TencentChatResponse `json:"Response"`
}

type TencentChatResponse struct {
	Id      string                  `json:"Id"`
	Object  string                  `json:"Object"`
	Created int64                   `json:"Created"`
	Model   string                  `json:"Model"`
	Choices []TencentChatChoice     `json:"Choices"`
	Usage   dto.Usage               `json:"Usage"`
	Error   TencentChatError        `json:"Error"`
}

type TencentChatChoice struct {
	Index        int                 `json:"Index"`
	Message      TencentChatMessage  `json:"Message"`
	Delta        TencentChatMessage  `json:"Delta"`
	FinishReason string              `json:"FinishReason"`
}

type TencentChatMessage struct {
	Role    string `json:"Role"`
	Content string `json:"Content"`
}

type TencentChatError struct {
	Code    int    `json:"Code"`
	Message string `json:"Message"`
}
