package payment

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ～Creem 生产/测试两个域名，根据 testMode 切换～
const (
	creemProdBase = "https://api.creem.io/v1"
	creemTestBase = "https://test-api.creem.io/v1"
)

type creemCheckoutReq struct {
	ProductId  string `json:"product_id"`
	RequestId  string `json:"request_id,omitempty"`
	SuccessUrl string `json:"success_url,omitempty"`
}

type creemCheckoutResp struct {
	CheckoutUrl string `json:"checkout_url"`
	Id          string `json:"id"`
	// ～出错时 Creem 返回 message 字段～
	Message interface{} `json:"message"`
	Status  int         `json:"status"`
}

// CreemCheckoutResult ～Creem 返回的跳转链接小票～
type CreemCheckoutResult struct {
	CheckoutId string
	PayUrl     string
}

// CreateCreemCheckout ～请 Creem 开一张付款单，换回跳转链接～
func CreateCreemCheckout(apiKey, productId, successUrl, requestId string, testMode bool) (*CreemCheckoutResult, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Creem API Key 未配置")
	}
	if productId == "" {
		return nil, fmt.Errorf("Creem Product ID 未配置")
	}

	baseUrl := creemProdBase
	if testMode {
		baseUrl = creemTestBase
	}

	body, _ := json.Marshal(creemCheckoutReq{
		ProductId:  productId,
		RequestId:  requestId,
		SuccessUrl: successUrl,
	})

	req, err := http.NewRequest(http.MethodPost, baseUrl+"/checkouts", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Creem 请求失败: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var result creemCheckoutResp
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("解析 Creem 响应失败: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Creem 返回错误 [%d]: %v", resp.StatusCode, result.Message)
	}
	if result.CheckoutUrl == "" {
		return nil, fmt.Errorf("Creem 未返回 checkout_url，原始响应: %s", string(raw))
	}

	return &CreemCheckoutResult{
		CheckoutId: result.Id,
		PayUrl:     result.CheckoutUrl,
	}, nil
}

// VerifyCreemWebhook ～用 HMAC-SHA256 验一下 Creem 发来的 Webhook 签名是不是真的～
func VerifyCreemWebhook(rawBody []byte, signature, secret string) bool {
	if secret == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(rawBody)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
