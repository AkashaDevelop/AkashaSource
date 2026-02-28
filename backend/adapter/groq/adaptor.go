package groq

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

type Adaptor struct{ openai openaiAdapter.Adaptor }

func (a *Adaptor) GetChannelName() string { return "groq" }
func (a *Adaptor) GetModelList() []string { return ModelList }
func (a *Adaptor) ConvertRequest(_ *gin.Context, req *dto.OpenAIRequest) (any, error) {
	return req, nil
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
	base := strings.TrimSuffix(channel.BaseURL, "/")
	if base == "" {
		base = BaseURL
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/chat/completions", base), bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+channel.Key)
	req.Header.Set("Content-Type", "application/json")
	common.ApplyHeaders(req, channel.Headers)
	return common.NewHTTPClient(channel.Proxy).Do(req)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *model.Token) (*dto.Usage, error) {
	return a.openai.DoResponse(c, resp, meta)
}
