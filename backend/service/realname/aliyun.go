package realname

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// AliyunClient 阿里云实人认证客户端（RPC 风格签名，无需额外 SDK 依赖）
type AliyunClient struct {
	AccessKeyId     string
	AccessKeySecret string
	Region          string // 如 cn-hangzhou
	SceneId         string // 认证场景ID
	endpoint        string
}

func NewAliyunClient(accessKeyId, accessKeySecret, region, sceneId string) *AliyunClient {
	if region == "" {
		region = "cn-hangzhou"
	}
	return &AliyunClient{
		AccessKeyId:     accessKeyId,
		AccessKeySecret: accessKeySecret,
		Region:          region,
		SceneId:         sceneId,
		endpoint:        fmt.Sprintf("cloudauth.%s.aliyuncs.com", region),
	}
}

// InitFaceVerifyResult 发起认证返回结果
type InitFaceVerifyResult struct {
	CertifyId    string `json:"CertifyId"`     // 认证流水号
	CertifyUrl   string `json:"CertifyUrl"`    // H5 认证链接
	RequestId    string `json:"RequestId"`
	Code         string `json:"Code"`          // 200 表示成功
	Message      string `json:"Message"`
}

// DescribeFaceVerifyResult 查询认证结果
type DescribeFaceVerifyResult struct {
	Passed       string `json:"Passed"`        // "T" 通过 "F" 未通过
	SubCode      string `json:"SubCode"`       // 错误子码
	Reason       string `json:"Reason"`        // 未通过原因
	RequestId    string `json:"RequestId"`
	Code         string `json:"Code"`          // 200 表示成功
	Message      string `json:"Message"`
}

// InitFaceVerify 发起人脸核身认证
// metaInfo 由前端 SDK 获取（如 ZIMJSBridge），纯 API 模式可传空
func (c *AliyunClient) InitFaceVerify(outerOrderNo, realName, idCardNo, metaInfo string) (*InitFaceVerifyResult, error) {
	params := map[string]string{
		"Action":           "InitFaceVerify",
		"Version":          "2022-10-18",
		"SceneId":          c.SceneId,
		"OuterOrderNo":     outerOrderNo,
		"ProductCode":      "ID_PRO",
		"CertType":         "IDENTITY_CARD",
		"CertName":         realName,
		"CertNo":           idCardNo,
		"MetaInfo":         metaInfo,
		"Mobile":           "",
		"Ip":               "",
		"UserId":           "",
		"ReturnUrl":        "",
		"RedirectUrl":      "",
		"FaceNotifyUrl":    "",
		"PrivacyNotices":   "",
	}

	resp, err := c.doRPC(params)
	if err != nil {
		return nil, err
	}

	var result InitFaceVerifyResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, body: %s", err, string(resp))
	}
	return &result, nil
}

// DescribeFaceVerify 查询认证结果
func (c *AliyunClient) DescribeFaceVerify(certifyId string) (*DescribeFaceVerifyResult, error) {
	params := map[string]string{
		"Action":        "DescribeFaceVerify",
		"Version":       "2022-10-18",
		"SceneId":       c.SceneId,
		"CertifyId":     certifyId,
	}

	resp, err := c.doRPC(params)
	if err != nil {
		return nil, err
	}

	var result DescribeFaceVerifyResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, body: %s", err, string(resp))
	}
	return &result, nil
}

// doRPC 执行阿里云 RPC 风格 API 调用
func (c *AliyunClient) doRPC(businessParams map[string]string) ([]byte, error) {
	// 1. 构建公共参数
	params := map[string]string{
		"Format":           "JSON",
		"Version":          "2022-10-18",
		"AccessKeyId":      c.AccessKeyId,
		"SignatureMethod":  "HMAC-SHA1",
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"SignatureVersion": "1.0",
		"SignatureNonce":   generateNonce(),
	}

	// 2. 合并业务参数
	for k, v := range businessParams {
		params[k] = v
	}

	// 3. 计算签名
	signature := c.sign(params, "POST")
	params["Signature"] = signature

	// 4. 发送 POST 请求
	formValues := url.Values{}
	for k, v := range params {
		formValues.Set(k, v)
	}

	apiUrl := fmt.Sprintf("https://%s/", c.endpoint)
	resp, err := http.PostForm(apiUrl, formValues)
	if err != nil {
		return nil, fmt.Errorf("请求阿里云API失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	return body, nil
}

// sign 计算阿里云 RPC 签名
func (c *AliyunClient) sign(params map[string]string, method string) string {
	// 1. 排序参数
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. 构建规范化查询字符串
	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, percentEncode(k)+"="+percentEncode(params[k]))
	}
	canonicalized := strings.Join(pairs, "&")

	// 3. 构建待签名字符串
	stringToSign := method + "&" + percentEncode("/") + "&" + percentEncode(canonicalized)

	// 4. HMAC-SHA1 签名
	h := hmac.New(sha1.New, []byte(c.AccessKeySecret+"&"))
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return signature
}

// percentEncode 阿里云要求的特殊 URL 编码
func percentEncode(s string) string {
	s = url.QueryEscape(s)
	s = strings.ReplaceAll(s, "+", "%20")
	s = strings.ReplaceAll(s, "*", "%2A")
	s = strings.ReplaceAll(s, "%7E", "~")
	return s
}

// generateNonce 生成随机签名 nonce
func generateNonce() string {
	return fmt.Sprintf("%d%d", time.Now().UnixNano(), rand.Intn(100000))
}
