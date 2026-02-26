package ali

import (
	"STfreApi/adapter/openai"
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
	openai.Adaptor
}

func (a *Adaptor) GetChannelName() string {
	return "ali"
}

func (a *Adaptor) ConvertRequest(c *gin.Context, request *dto.OpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("请求为空")
	}
	if request.TopP > 0 && request.TopP >= 1 {
		request.TopP = 0.999
	}
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
		baseUrl = "https://dashscope.aliyuncs.com/compatible-mode"
	}
	baseUrl = strings.TrimSuffix(baseUrl, "/")

	targetURL := ""
	if strings.HasPrefix(openAIReq.Model, "dall-e") {
		targetURL = baseUrl + "/v1/images/generations"
	} else if openAIReq.Input != nil {
		targetURL = baseUrl + "/v1/embeddings"
	} else {
		targetURL = baseUrl + "/v1/chat/completions"
	}

	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+channel.Key)
	if openAIReq.ContentType != "" {
		req.Header.Set("Content-Type", openAIReq.ContentType)
	} else {
		req.Header.Set("Content-Type", "application/json")
	}
	if openAIReq.Stream {
		req.Header.Set("X-DashScope-SSE", "enable")
	}
	common.ApplyHeaders(req, channel.Headers)
	client := common.NewHTTPClient(channel.Proxy)
	return client.Do(req)
}
