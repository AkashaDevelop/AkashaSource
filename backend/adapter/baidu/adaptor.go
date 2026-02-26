package baidu

import (
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
	isEmbedding bool
	model       string
}

var baiduTokenStore sync.Map

func (a *Adaptor) GetChannelName() string {
	return "baidu"
}

func (a *Adaptor) GetModelList() []string {
	return []string{}
}

func (a *Adaptor) ConvertRequest(c *gin.Context, request *dto.OpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("请求为空")
	}
	a.model = request.Model
	if request.Input != nil {
		a.isEmbedding = true
		return embeddingRequestOpenAI2Baidu(request.Input), nil
	}
	a.isEmbedding = false
	return requestOpenAI2Baidu(request), nil
}

func (a *Adaptor) DoRequest(c *gin.Context, channel *model.Channel, request any) (*http.Response, error) {
	baseUrl := channel.BaseURL
	if baseUrl == "" {
		baseUrl = "https://aip.baidubce.com"
	}
	baseUrl = strings.TrimSuffix(baseUrl, "/")
	suffix := getBaiduEndpoint(a.model, a.isEmbedding)
	accessToken, err := getBaiduAccessToken(channel.Key)
	if err != nil {
		return nil, err
	}
	fullURL := fmt.Sprintf("%s/rpc/2.0/ai_custom/v1/wenxinworkshop/%s?access_token=%s", baseUrl, suffix, accessToken)
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := common.NewHTTPClient(channel.Proxy)
	return client.Do(req)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *model.Token) (*dto.Usage, error) {
	if a.isEmbedding {
		return a.embeddingHandler(c, resp)
	}
	if isStreamResponse(resp) {
		return a.streamHandler(c, resp)
	}
	return a.normalHandler(c, resp)
}

func isStreamResponse(resp *http.Response) bool {
	return strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")
}

func (a *Adaptor) normalHandler(c *gin.Context, resp *http.Response) (*dto.Usage, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var baiduResp BaiduChatResponse
	if err := json.Unmarshal(body, &baiduResp); err != nil {
		c.JSON(resp.StatusCode, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "响应解析失败",
			Type:    "server_error",
		}})
		return nil, err
	}
	if baiduResp.ErrorMsg != "" {
		c.JSON(resp.StatusCode, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: baiduResp.ErrorMsg,
			Type:    "server_error",
			Code:    baiduResp.ErrorCode,
		}})
		return nil, errors.New(baiduResp.ErrorMsg)
	}

	openaiResp := responseBaidu2OpenAI(&baiduResp, a.model)
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
		var chunk BaiduChatStreamResponse
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}
		if chunk.Usage.TotalTokens != 0 {
			usage.TotalTokens = chunk.Usage.TotalTokens
			usage.PromptTokens = chunk.Usage.PromptTokens
			usage.CompletionTokens = chunk.Usage.TotalTokens - chunk.Usage.PromptTokens
		}
		streamResp := streamResponseBaidu2OpenAI(&chunk, a.model)
		data, _ := json.Marshal(streamResp)
		c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
		c.Writer.Flush()
	}
	return usage, nil
}

func (a *Adaptor) embeddingHandler(c *gin.Context, resp *http.Response) (*dto.Usage, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var baiduResp BaiduEmbeddingResponse
	if err := json.Unmarshal(body, &baiduResp); err != nil {
		c.JSON(resp.StatusCode, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "响应解析失败",
			Type:    "server_error",
		}})
		return nil, err
	}
	if baiduResp.ErrorMsg != "" {
		c.JSON(resp.StatusCode, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: baiduResp.ErrorMsg,
			Type:    "server_error",
			Code:    baiduResp.ErrorCode,
		}})
		return nil, errors.New(baiduResp.ErrorMsg)
	}

	openaiResp := embeddingResponseBaidu2OpenAI(&baiduResp, a.model)
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	_ = json.NewEncoder(c.Writer).Encode(openaiResp)
	return &openaiResp.Usage, nil
}

func requestOpenAI2Baidu(request *dto.OpenAIRequest) *BaiduChatRequest {
	msgs := make([]BaiduMessage, 0, len(request.Messages))
	system := ""
	for _, message := range request.Messages {
		role, content := parseMessage(message)
		if role == "" {
			continue
		}
		if role == "system" {
			system = content
		} else {
			msgs = append(msgs, BaiduMessage{
				Role:    role,
				Content: content,
			})
		}
	}
	baiduRequest := BaiduChatRequest{
		Messages:        msgs,
		Temperature:     &request.Temperature,
		TopP:            request.TopP,
		PenaltyScore:    request.FrequencyPenalty,
		Stream:          request.Stream,
		System:          system,
		DisableSearch:   false,
		EnableCitation:  false,
		UserId:          request.User,
		MaxOutputTokens: maxOutputTokens(request.MaxTokens),
	}
	return &baiduRequest
}

func embeddingRequestOpenAI2Baidu(input any) *BaiduEmbeddingRequest {
	return &BaiduEmbeddingRequest{
		Input: parseEmbeddingInput(input),
	}
}

func responseBaidu2OpenAI(response *BaiduChatResponse, modelName string) *dto.ChatCompletionResponse {
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
				Content: response.Result,
			},
			Index:        0,
			FinishReason: "stop",
		},
	}
	return &resp
}

func streamResponseBaidu2OpenAI(response *BaiduChatStreamResponse, modelName string) *dto.ChatCompletionStreamResponse {
	resp := dto.ChatCompletionStreamResponse{
		Object:  "chat.completion.chunk",
		Created: response.Created,
		Model:   modelName,
	}
	resp.Choices = []dto.StreamChoice{
		{
			Delta: dto.StreamDelta{
				Content: response.Result,
			},
			Index:        0,
			FinishReason: finishReason(response.IsEnd),
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

func embeddingResponseBaidu2OpenAI(response *BaiduEmbeddingResponse, modelName string) *dto.EmbeddingResponse {
	openAIEmbeddingResponse := dto.EmbeddingResponse{
		Object: "list",
		Model:  modelName,
		Usage:  response.Usage,
	}
	for _, item := range response.Data {
		openAIEmbeddingResponse.Data = append(openAIEmbeddingResponse.Data, struct {
			Object    string    `json:"object"`
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		}{
			Object:    item.Object,
			Embedding: item.Embedding,
			Index:     item.Index,
		})
	}
	return &openAIEmbeddingResponse
}

func getBaiduAccessToken(apiKey string) (string, error) {
	if val, ok := baiduTokenStore.Load(apiKey); ok {
		if accessToken, ok := val.(BaiduAccessToken); ok {
			if time.Now().Add(time.Hour).After(accessToken.ExpiresAt) {
				go func() {
					_, _ = getBaiduAccessTokenHelper(apiKey)
				}()
			}
			return accessToken.AccessToken, nil
		}
	}
	accessToken, err := getBaiduAccessTokenHelper(apiKey)
	if err != nil {
		return "", err
	}
	return accessToken.AccessToken, nil
}

func getBaiduAccessTokenHelper(apiKey string) (*BaiduAccessToken, error) {
	parts := strings.Split(apiKey, "|")
	if len(parts) != 2 {
		return nil, errors.New("百度 API Key 格式错误")
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("https://aip.baidubce.com/oauth/2.0/token?grant_type=client_credentials&client_id=%s&client_secret=%s", parts[0], parts[1]), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	client := common.NewHTTPClient("")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var accessToken BaiduAccessToken
	if err := json.NewDecoder(res.Body).Decode(&accessToken); err != nil {
		return nil, err
	}
	if accessToken.Error != "" {
		return nil, errors.New(accessToken.Error + ": " + accessToken.ErrorDescription)
	}
	if accessToken.AccessToken == "" {
		return nil, errors.New("获取百度 access_token 失败")
	}
	accessToken.ExpiresAt = time.Now().Add(time.Duration(accessToken.ExpiresIn) * time.Second)
	baiduTokenStore.Store(apiKey, accessToken)
	return &accessToken, nil
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

func maxOutputTokens(val int) *int {
	if val <= 0 {
		return nil
	}
	if val == 1 {
		v := 2
		return &v
	}
	return &val
}

func parseEmbeddingInput(input any) []string {
	switch v := input.(type) {
	case string:
		return []string{v}
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, fmt.Sprint(item))
		}
		return out
	default:
		return []string{fmt.Sprint(input)}
	}
}

func getBaiduEndpoint(modelName string, isEmbedding bool) string {
	suffix := "chat/"
	if isEmbedding {
		suffix = "embeddings/"
	}
	switch modelName {
	case "ERNIE-4.0", "ERNIE-Bot-4", "ERNIE-4.0-8K":
		suffix += "completions_pro"
	case "ERNIE-Bot":
		suffix += "completions"
	case "ERNIE-Bot-turbo", "ERNIE-Lite-8K-0922":
		suffix += "eb-instant"
	case "ERNIE-Speed", "ERNIE-Speed-8K":
		suffix += "ernie_speed"
	case "ERNIE-Speed-128K":
		suffix += "ernie-speed-128k"
	case "ERNIE-3.5-8K":
		suffix += "completions"
	case "ERNIE-3.5-8K-0205":
		suffix += "ernie-3.5-8k-0205"
	case "ERNIE-3.5-8K-1222":
		suffix += "ernie-3.5-8k-1222"
	case "ERNIE-Bot-8K":
		suffix += "ernie_bot_8k"
	case "ERNIE-3.5-4K-0205":
		suffix += "ernie-3.5-4k-0205"
	case "ERNIE-Lite-8K-0308":
		suffix += "ernie-lite-8k"
	case "ERNIE-Tiny-8K":
		suffix += "ernie-tiny-8k"
	case "BLOOMZ-7B":
		suffix += "bloomz_7b1"
	case "Embedding-V1":
		suffix += "embedding-v1"
	case "bge-large-zh":
		suffix += "bge_large_zh"
	case "bge-large-en":
		suffix += "bge_large_en"
	case "tao-8k":
		suffix += "tao_8k"
	default:
		if isEmbedding {
			suffix += "embedding-v1"
		} else {
			suffix += strings.ToLower(modelName)
		}
	}
	return suffix
}

type BaiduMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type BaiduChatRequest struct {
	Messages        []BaiduMessage `json:"messages"`
	Temperature     *float64       `json:"temperature,omitempty"`
	TopP            float64        `json:"top_p,omitempty"`
	PenaltyScore    float64        `json:"penalty_score,omitempty"`
	Stream          bool           `json:"stream,omitempty"`
	System          string         `json:"system,omitempty"`
	DisableSearch   bool           `json:"disable_search,omitempty"`
	EnableCitation  bool           `json:"enable_citation,omitempty"`
	MaxOutputTokens *int           `json:"max_output_tokens,omitempty"`
	UserId          string         `json:"user_id,omitempty"`
}

type BaiduEmbeddingRequest struct {
	Input []string `json:"input"`
}

type BaiduEmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

type BaiduEmbeddingResponse struct {
	Id      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Data    []BaiduEmbeddingData `json:"data"`
	Usage   dto.Usage            `json:"usage"`
	Error
}

type BaiduChatResponse struct {
	Id               string    `json:"id"`
	Object           string    `json:"object"`
	Created          int64     `json:"created"`
	Result           string    `json:"result"`
	IsTruncated      bool      `json:"is_truncated"`
	NeedClearHistory bool      `json:"need_clear_history"`
	Usage            dto.Usage `json:"usage"`
	Error
}

type BaiduChatStreamResponse struct {
	BaiduChatResponse
	SentenceId int  `json:"sentence_id"`
	IsEnd      bool `json:"is_end"`
}

type Error struct {
	ErrorCode int    `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
}

type BaiduAccessToken struct {
	AccessToken      string    `json:"access_token"`
	Error            string    `json:"error,omitempty"`
	ErrorDescription string    `json:"error_description,omitempty"`
	ExpiresIn        int64     `json:"expires_in,omitempty"`
	ExpiresAt        time.Time `json:"-"`
}
