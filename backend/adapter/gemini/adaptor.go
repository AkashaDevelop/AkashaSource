package gemini

import (
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
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
	return "gemini"
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

type GeminiChatRequest struct {
	Contents          []GeminiContent   `json:"contents"`
	SystemInstruction *GeminiContent    `json:"systemInstruction,omitempty"`
	SafetySettings    []SafetySetting   `json:"safetySettings,omitempty"`
	GenerationConfig  GenerationConfig  `json:"generationConfig,omitempty"`
	Tools             []GeminiTool      `json:"tools,omitempty"`
	ThinkingConfig    *ThinkingConfig   `json:"thinkingConfig,omitempty"`

	// Internal fields
	Model  string `json:"-"`
	Stream bool   `json:"-"`
}

type GeminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text,omitempty"`
}

type SafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type GenerationConfig struct {
	Temperature     float64  `json:"temperature,omitempty"`
	TopP            float64  `json:"topP,omitempty"`
	TopK            int      `json:"topK,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

// Gemini tool types
type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations,omitempty"`
}

type GeminiFunctionDeclaration struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// ThinkingConfig for Gemini thinking mode
type ThinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget,omitempty"`
}

type GeminiChatResponse struct {
	Candidates     []GeminiCandidate `json:"candidates"`
	PromptFeedback *PromptFeedback   `json:"promptFeedback,omitempty"`
	UsageMetadata  *UsageMetadata    `json:"usageMetadata,omitempty"`
}

type GeminiCandidate struct {
	Content      GeminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
	Index        int           `json:"index"`
}

type PromptFeedback struct {
	BlockReason string `json:"blockReason"`
}

type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

func (a *Adaptor) ConvertRequest(c *gin.Context, request *dto.OpenAIRequest) (any, error) {
	geminiReq := GeminiChatRequest{
		Model:  request.Model,
		Stream: request.Stream,
		GenerationConfig: GenerationConfig{
			Temperature:     request.Temperature,
			TopP:            request.TopP,
			MaxOutputTokens: request.MaxTokens,
		},
	}

	// Safety Settings (Default: BLOCK_NONE)
	geminiReq.SafetySettings = []SafetySetting{
		{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_NONE"},
		{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_NONE"},
		{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_NONE"},
		{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_NONE"},
	}

	// Convert messages with safe type assertions
	var systemParts []string
	for _, msg := range request.Messages {
		m, ok := msg.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := m["role"].(string)
		content := extractGeminiContent(m["content"])

		if role == "system" {
			systemParts = append(systemParts, content)
			continue
		}

		geminiRole := "user"
		if role == "assistant" {
			geminiRole = "model"
		}

		geminiReq.Contents = append(geminiReq.Contents, GeminiContent{
			Role:  geminiRole,
			Parts: []GeminiPart{{Text: content}},
		})
	}

	// System instruction support
	if len(systemParts) > 0 {
		sysText := strings.Join(systemParts, "\n")
		geminiReq.SystemInstruction = &GeminiContent{
			Parts: []GeminiPart{{Text: sysText}},
		}
	}

	// Convert tools
	if len(request.Tools) > 0 {
		var decls []GeminiFunctionDeclaration
		for _, t := range request.Tools {
			tm, ok := t.(map[string]interface{})
			if !ok {
				continue
			}
			fn, ok := tm["function"].(map[string]interface{})
			if !ok {
				continue
			}
			decl := GeminiFunctionDeclaration{}
			decl.Name, _ = fn["name"].(string)
			decl.Description, _ = fn["description"].(string)
			if params, ok := fn["parameters"]; ok {
				decl.Parameters, _ = json.Marshal(params)
			}
			decls = append(decls, decl)
		}
		if len(decls) > 0 {
			geminiReq.Tools = []GeminiTool{{FunctionDeclarations: decls}}
		}
	}

	// Thinking mode support
	if request.ThinkingMode {
		geminiReq.ThinkingConfig = &ThinkingConfig{ThinkingBudget: 10000}
	}

	return &geminiReq, nil
}

// extractGeminiContent safely extracts text from message content (string or array)
func extractGeminiContent(content interface{}) string {
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

func (a *Adaptor) DoRequest(c *gin.Context, channel *model.Channel, request any) (*http.Response, error) {
	geminiReq := request.(*GeminiChatRequest)
	reqBody, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, err
	}

	baseUrl := channel.BaseURL
	if baseUrl == "" {
		baseUrl = BaseURL
	}
	baseUrl = strings.TrimSuffix(baseUrl, "/")

	action := "generateContent"
	if geminiReq.Stream {
		action = "streamGenerateContent"
	}

	// URL: https://generativelanguage.googleapis.com/v1beta/models/{model}:{action}?key={api_key}
	// Use mapped model from request
	targetURL := fmt.Sprintf("%s/v1beta/models/%s:%s?key=%s", baseUrl, geminiReq.Model, action, channel.Key)

	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	common.ApplyHeaders(req, channel.Headers)

	client := common.NewHTTPClient(channel.Proxy)
	return client.Do(req)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *model.Token) (usage *dto.Usage, err error) {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("provider error: %s", string(body))
	}

	// Determine stream by URL or Content-Type? Or rely on what we sent.
	// We can check Content-Type for "application/json" vs ...
	// Gemini stream returns JSON array progressively? Or specific format.
	// Gemini stream returns JSON array of chunks.

	// If stream, action was streamGenerateContent.
	// The response is a stream of JSON objects? Or one big JSON array?
	// It's usually a stream of JSON objects if using REST?
	// Actually Gemini REST API returns a JSON array `[{...}, {...}]`.
	// Streaming involves parsing this array incrementally.

	// Wait, if it returns a JSON array, `json.Decoder` can handle it.
	// But we need to check if it's stream first.
	// Let's assume if it's `streamGenerateContent`, we handle as stream.
	// But `DoResponse` doesn't know.
	// However, we can peek body or check header.
	// If we used `streamGenerateContent`, the response body is streamed.

	// Let's try to detect if it's stream response format.
	// For simplicity, let's assume if the request was stream (we don't know here easily), we use stream handler.
	// BUT, `ConvertRequest` knew.
	// Can we check `c.Writer` status? No.

	// Hack: Check URL in `resp.Request.URL.String()` if possible?
	if strings.Contains(resp.Request.URL.String(), "streamGenerateContent") {
		return a.streamHandler(c, resp)
	} else {
		return a.normalHandler(c, resp)
	}
}

func (a *Adaptor) normalHandler(c *gin.Context, resp *http.Response) (*dto.Usage, error) {
	var geminiResp GeminiChatResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, err
	}

	content := ""
	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		content = geminiResp.Candidates[0].Content.Parts[0].Text
	}

	usage := dto.Usage{}
	if geminiResp.UsageMetadata != nil {
		usage.PromptTokens = geminiResp.UsageMetadata.PromptTokenCount
		usage.CompletionTokens = geminiResp.UsageMetadata.CandidatesTokenCount
		usage.TotalTokens = geminiResp.UsageMetadata.TotalTokenCount
	}

	openaiResp := dto.ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%s", common.GetUUID()),
		Object:  "chat.completion",
		Created: common.GetTimestamp(),
		Model:   "gemini",
		Choices: []dto.Choice{
			{
				Message: dto.ChatMessage{
					Role:    "assistant",
					Content: content,
				},
				Index:        0,
				FinishReason: "stop",
			},
		},
		Usage: usage,
	}

	// 把 Gemini 酱说的话悄悄塞进 context 小背篓，给玄鉴风控闻一闻~ (qy_completion_text)
	if content != "" {
		c.Set("qy_completion_text", content)
	}

	c.JSON(http.StatusOK, openaiResp)
	return &usage, nil
}

func (a *Adaptor) streamHandler(c *gin.Context, resp *http.Response) (*dto.Usage, error) {
	common.SetEventStreamHeaders(c)

	// Gemini returns a JSON array `[{}, {}, ...]`.
	// We need to parse this incrementally.
	// A simple way is to read `{` to `}` blocks?
	// Or use a JSON decoder and loop `Decode()`.

	dec := json.NewDecoder(resp.Body)

	// Expect start of array
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := t.(json.Delim); !ok || delim != '[' {
		return nil, fmt.Errorf("expected start of array")
	}

	usage := &dto.Usage{}
	var id = fmt.Sprintf("chatcmpl-%s", common.GetUUID())
	// 流式小碎片汇总桶，一片一片攒起来给风控君检查呀
	var qyTextBuf strings.Builder

	for dec.More() {
		var chunk GeminiChatResponse
		if err := dec.Decode(&chunk); err != nil {
			break
		}

		content := ""
		if len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Content.Parts) > 0 {
			content = chunk.Candidates[0].Content.Parts[0].Text
		}
		if content != "" {
			qyTextBuf.WriteString(content)
		}

		if chunk.UsageMetadata != nil {
			usage.PromptTokens = chunk.UsageMetadata.PromptTokenCount
			usage.CompletionTokens = chunk.UsageMetadata.CandidatesTokenCount
			usage.TotalTokens = chunk.UsageMetadata.TotalTokenCount
		}

		sendStreamResponse(c, id, "gemini", content, "assistant", "")
	}

	// Expect end of array
	_, _ = dec.Token()

	c.Writer.WriteString("data: [DONE]\n\n")
	c.Writer.Flush()

	// 流跑完啦，把攒好的完整回答一次性放进 context 小背篓~ (qy_completion_text)
	if qyTextBuf.Len() > 0 {
		c.Set("qy_completion_text", qyTextBuf.String())
	}

	return usage, nil
}

func sendStreamResponse(c *gin.Context, id, model, content, role, finishReason string) {
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
