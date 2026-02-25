package openai

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
	return "openai"
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) ConvertRequest(c *gin.Context, request *dto.OpenAIRequest) (any, error) {
	// OpenAI to OpenAI is just a passthrough, but we might need to handle specific logic if needed.
	// For now, just return the request as is (pointer or value? interface{} handles both)
	return request, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, channel *model.Channel, request any) (*http.Response, error) {
	openAIReq := request.(*dto.OpenAIRequest)
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	baseUrl := channel.BaseURL
	if baseUrl == "" {
		baseUrl = BaseURL
	}
	// Ensure no trailing slash
	baseUrl = strings.TrimSuffix(baseUrl, "/")

	// Determine endpoint based on model or request type
	targetURL := ""
	if strings.HasPrefix(openAIReq.Model, "dall-e") {
		targetURL = fmt.Sprintf(ImagesGenerationsURL, baseUrl)
	} else {
		targetURL = fmt.Sprintf(ChatCompletionsURL, baseUrl)
	}

	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", channel.Key))
	req.Header.Set("Content-Type", "application/json")

	// Azure needs special headers (api-key), handled in separate Azure adaptor or here with checks
	if channel.Type == model.ChannelTypeAzure {
		req.Header.Set("api-key", channel.Key)
		// URL is also different for Azure
		// https://{your-resource-name}.openai.azure.com/openai/deployments/{deployment-id}/chat/completions?api-version={api-version}
		// BaseURL usually provided as full URL or resource name
		// Let's assume BaseURL is full endpoint for Custom/Azure for simplicity in this pass,
		// or user provides standard OpenAI compatible endpoint for Azure Proxy.
		// If using raw Azure, we need more logic. Let's stick to standard OpenAI for now.
	}

	client := &http.Client{}
	return client.Do(req)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *model.Token) (usage *dto.Usage, err error) {
	// Stream check
	isStream := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")

	if isStream {
		return a.streamHandler(c, resp)
	} else {
		return a.normalHandler(c, resp)
	}
}

func (a *Adaptor) normalHandler(c *gin.Context, resp *http.Response) (*dto.Usage, error) {
	body, _ := io.ReadAll(resp.Body)

	// Write back to client
	for k, v := range resp.Header {
		c.Writer.Header().Set(k, v[0])
	}
	c.Writer.WriteHeader(resp.StatusCode)
	c.Writer.Write(body)

	var response dto.ChatCompletionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, nil // Error parsing response, maybe not JSON
	}

	// If it's an image response, it might not have usage.
	// We need to estimate usage for images.
	if response.ID == "" && len(response.Choices) == 0 {
		// Try parsing as Image Response
		var imgResp dto.ImageGenerationResponse
		if err := json.Unmarshal(body, &imgResp); err == nil && len(imgResp.Data) > 0 {
			// Image generation successful
			// Calculate usage based on DALL-E pricing model (simplified)
			// DALL-E 3: Standard $0.04 / image (HD $0.08)
			// DALL-E 2: 1024x1024 $0.02
			// Since our quota system is 500000 = $1,
			// $0.04 = 20000 quota.
			// But we usually rely on Ratio.
			// Let's return a special Usage or handle it in Relay logic.
			// Adaptor should return token count equivalent.
			// If we say 1 image = N tokens.
			// $0.04 = 20000 quota = 20000 tokens (if ratio=1 and 1token=$0.000002)
			// Actually 1k tokens = $0.002. So $1 = 500k tokens.
			// $0.04 = 20k tokens.

			// We can return a fixed usage here.
			return &dto.Usage{
				PromptTokens:     0,
				CompletionTokens: 0,
				TotalTokens:      0, // Let Relay handle fixed price for images?
			}, nil
		}
	}

	return &response.Usage, nil
}

func (a *Adaptor) streamHandler(c *gin.Context, resp *http.Response) (*dto.Usage, error) {
	common.SetEventStreamHeaders(c)

	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
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
	// We need to count tokens manually for stream if usage is not provided in stream end
	// For simplicity, we just return empty usage or rely on estimation outside

	for scanner.Scan() {
		data := scanner.Text()
		if len(data) < 6 {
			continue
		}

		c.Writer.Write([]byte(data + "\n\n"))
		c.Writer.Flush()

		// Logic to extract usage from stream_options: include_usage (OpenAI feature)
		// Or count tokens from content
	}

	return usage, nil
}
