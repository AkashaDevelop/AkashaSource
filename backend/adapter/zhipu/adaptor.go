package zhipu

import (
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	openaiAdapter "STfreApi/adapter/openai"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
	openai openaiAdapter.Adaptor
}

func (a *Adaptor) GetChannelName() string {
	return "zhipu"
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

// generateZhipuToken generates a JWT token for Zhipu API
// Key format: "id.secret"
func generateZhipuToken(apiKey string) (string, error) {
	parts := strings.SplitN(apiKey, ".", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid zhipu key format, expected id.secret")
	}
	id, secret := parts[0], parts[1]

	now := time.Now().UnixMilli()
	exp := now + 3600*1000 // 1 hour

	// Header
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","sign_type":"SIGN"}`))
	// Payload
	payload := base64.RawURLEncoding.EncodeToString(
		[]byte(fmt.Sprintf(`{"api_key":"%s","exp":%d,"timestamp":%d}`, id, exp, now)),
	)

	// Signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(header + "." + payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return header + "." + payload + "." + sig, nil
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
	targetURL := fmt.Sprintf("%s/chat/completions", baseUrl)

	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	token, err := generateZhipuToken(channel.Key)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	common.ApplyHeaders(req, channel.Headers)

	client := common.NewHTTPClient(channel.Proxy)
	return client.Do(req)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *model.Token) (usage *dto.Usage, err error) {
	return a.openai.DoResponse(c, resp, meta)
}
