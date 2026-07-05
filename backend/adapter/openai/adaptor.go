package openai

import (
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
	"STfreApi/service/qingyuan"
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
	// Inject reasoning_effort if set via suffix
	// The field is already in the JSON struct, so it will be marshaled automatically.
	// No additional work needed for OpenAI — the field is sent as-is.
	return request, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, channel *model.Channel, request any) (*http.Response, error) {
	openAIReq := request.(*dto.OpenAIRequest)
	var reqBody []byte
	var err error

	if len(openAIReq.RawBody) > 0 {
		reqBody = openAIReq.RawBody
	} else {
		reqBody, err = json.Marshal(request)
		if err != nil {
			return nil, err
		}
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
	} else if strings.HasPrefix(openAIReq.Model, "tts") {
		targetURL = fmt.Sprintf("%s/v1/audio/speech", baseUrl)
	} else if strings.HasPrefix(openAIReq.Model, "whisper") {
		targetURL = fmt.Sprintf("%s/v1/audio/transcriptions", baseUrl)
	} else if strings.HasPrefix(openAIReq.Model, "text-moderation") {
		targetURL = fmt.Sprintf(ModerationsURL, baseUrl)
	} else if openAIReq.Query != "" {
		targetURL = fmt.Sprintf(RerankURL, baseUrl)
	} else if openAIReq.Prompt != "" && openAIReq.Input == nil && len(openAIReq.Messages) == 0 {
		targetURL = fmt.Sprintf(CompletionsURL, baseUrl)
	} else if openAIReq.Input != nil {
		targetURL = fmt.Sprintf(EmbeddingsURL, baseUrl)
	} else {
		targetURL = fmt.Sprintf(ChatCompletionsURL, baseUrl)
	}

	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", channel.Key))
	if openAIReq.ContentType != "" {
		req.Header.Set("Content-Type", openAIReq.ContentType)
	} else {
		req.Header.Set("Content-Type", "application/json")
	}
	common.ApplyHeaders(req, channel.Headers)

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

	client := common.NewHTTPClient(channel.Proxy)
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

	// 应用响应侧上下文净化
	if policyValue, exists := c.Get("context_sanitization_policy"); exists {
		if policy, ok := policyValue.(qingyuan.ResolvedPolicy); ok {
			var rc qingyuan.ResponseContext
			if rcValue, ok := c.Get("context_sanitization_request_context"); ok {
				if reqCtx, ok := rcValue.(qingyuan.RequestContext); ok {
					rc.RequestContext = reqCtx
				}
			}
			if v, ok := c.Get("context_sanitization_user_requested_ads"); ok {
				if b, ok := v.(bool); ok {
					rc.UserRequestedAds = b
				}
			}
			// 传递请求工具列表用于响应工具调用校验
			if v, ok := c.Get("context_sanitization_request_tools"); ok {
				if tools, ok := v.([]interface{}); ok {
					rc.RequestTools = tools
				}
			}

			if result, err := qingyuan.ApplyOpenAIResponse(c.Request.Context(), body, rc, policy); err == nil && result != nil {
				if result.Blocked {
					// 响应被阻断
					c.JSON(http.StatusForbidden, dto.OpenAIErrorResponse{
						Error: dto.OpenAIError{
							Message: result.Message,
							Type:    "invalid_response_content",
							Code:    "response_sanitization_blocked",
						},
					})
					return nil, fmt.Errorf("response blocked by sanitizer")
				}
				body = result.Body
			}
		}
	}

	// Write back to client
	for k, v := range resp.Header {
		c.Writer.Header().Set(k, v[0])
	}
	c.Writer.WriteHeader(resp.StatusCode)
	c.Writer.Write(body)

	var response dto.ChatCompletionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, nil
	}

	if len(response.Choices) > 0 {
		// Extract cached tokens from prompt_tokens_details
		if response.Usage.PromptTokensDetails != nil && response.Usage.PromptTokensDetails.CachedTokens > 0 {
			response.Usage.CachedTokens = response.Usage.PromptTokensDetails.CachedTokens
		}

		// 悄悄把 AI 酱说的话塞进 context 小背篓，给玄鉴风控闻一闻有没有危险味道~ (qy_completion_text)
		// 这里覆盖了 moonshot/ollama/perplexity/siliconflow 等一大票直接委托给本 Adaptor 的小伙伴，
		// 因为大家共用同一个 gin.Context，一处塞好，处处都能取到啦
		if content := response.Choices[0].Message.Content; content != "" {
			c.Set("qy_completion_text", content)
		}
		return &response.Usage, nil
	}

	// Try Embedding Response
	var embeddingResp dto.EmbeddingResponse
	if err := json.Unmarshal(body, &embeddingResp); err == nil && len(embeddingResp.Data) > 0 {
		return &embeddingResp.Usage, nil
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

	// 获取上下文净化策略
	var policy qingyuan.ResolvedPolicy
	var requestTools []interface{}
	shouldSanitize := false

	if policyValue, exists := c.Get("context_sanitization_policy"); exists {
		if p, ok := policyValue.(qingyuan.ResolvedPolicy); ok {
			policy = p
			shouldSanitize = true
		}
	}
	if v, ok := c.Get("context_sanitization_request_tools"); ok {
		if tools, ok := v.([]interface{}); ok {
			requestTools = tools
		}
	}

	// 创建流式处理器
	var processor *qingyuan.StreamProcessor
	if shouldSanitize && qingyuan.IsEnabled(policy) {
		processor = qingyuan.NewStreamProcessor(policy, requestTools)
	}

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
	// 悄悄准备一个小背篓，把流式吐出来的字一点点攒起来给玄鉴风控闻一闻~ (qy_completion_text)
	var qyTextBuf strings.Builder

	for scanner.Scan() {
		data := scanner.Text()
		if len(data) < 6 {
			continue
		}

		// 从原始数据块里摸一点点文字攒进小背篓，摸不到就算啦，不耽误正常转发
		if payload := strings.TrimPrefix(data, "data: "); payload != "[DONE]" {
			var qyChunk dto.ChatCompletionStreamResponse
			if err := json.Unmarshal([]byte(payload), &qyChunk); err == nil {
				for _, qyChoice := range qyChunk.Choices {
					if qyChoice.Delta.Content != "" {
						qyTextBuf.WriteString(qyChoice.Delta.Content)
					}
				}
			}
		}

		// 宸汐清源: 尾部广告策略开启时，processor 会把最后一小截"扣在手里"暂不放行
		chunkToWrite := []byte(data + "\n\n")
		if processor != nil {
			if processed, err := processor.ProcessChunk([]byte(data + "\n")); err == nil {
				chunkToWrite = processed
			}
		}

		c.Writer.Write(chunkToWrite)
		c.Writer.Flush()
	}

	// 流结束后检测: 把之前"扣在手里"没放行的尾部，做完广告/注入检测再决定要不要补写出去
	if processor != nil {
		tail, findings := processor.Finalize()
		if len(tail) > 0 {
			c.Writer.Write(tail)
			c.Writer.Flush()
		}
		if len(findings) > 0 {
			// 异步记录事件
			go func() {
				var rc qingyuan.RequestContext
				if rcValue, ok := c.Get("context_sanitization_request_context"); ok {
					if reqCtx, ok := rcValue.(qingyuan.RequestContext); ok {
						rc = reqCtx
					}
				}

				maxScore := 0
				for _, f := range findings {
					if f.Score > maxScore {
						maxScore = f.Score
					}
				}

				qingyuan.RecordEventAsync(qingyuan.EventInput{
					RequestId:      rc.RequestId,
					UserId:         rc.UserId,
					TokenId:        rc.TokenId,
					TokenName:      rc.TokenName,
					ChannelId:      rc.ChannelId,
					ChannelName:    rc.ChannelName,
					RequestedModel: rc.RequestedModel,
					MappedModel:    rc.MappedModel,
					Policy:         policy,
					Direction:      "response",
					Stage:          "stream_finalize",
					Action:         "monitor",
					RiskScore:      maxScore,
					Findings:       findings,
					SnippetSource:  processor.GetTailBuffer(),
				})
			}()
		}
	}

	// 流跑完啦，把攒好的完整回答一次性放进 context 小背篓，给玄鉴风控闻一闻~ (qy_completion_text)
	if qyTextBuf.Len() > 0 {
		c.Set("qy_completion_text", qyTextBuf.String())
	}

	return usage, nil
}
