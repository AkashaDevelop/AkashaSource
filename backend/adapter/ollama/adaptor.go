package ollama

import (
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	openaiAdapter "STfreApi/adapter/openai"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
	openai openaiAdapter.Adaptor
}

func (a *Adaptor) GetChannelName() string {
	return "ollama"
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) ConvertRequest(c *gin.Context, request *dto.OpenAIRequest) (any, error) {
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
	baseUrl = strings.TrimSuffix(baseUrl, "/")
	targetURL := fmt.Sprintf("%s/v1/chat/completions", baseUrl)

	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	// Ollama key can be empty
	if channel.Key != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", channel.Key))
	}
	req.Header.Set("Content-Type", "application/json")
	common.ApplyHeaders(req, channel.Headers)

	client := common.NewHTTPClient(channel.Proxy)
	return client.Do(req)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *model.Token) (usage *dto.Usage, err error) {
	return a.openai.DoResponse(c, resp, meta)
}
