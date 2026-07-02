package moderation

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	tmsServiceName = "tms"
	tmsAction      = "TextModeration"
	tmsVersion     = "2020-12-29"
)

// TMSConfig ～连上腾讯云天御前，先把钥匙和小纸条准备齐全～
type TMSConfig struct {
	SecretId  string
	SecretKey string
	Region    string
	BizType   string
	Timeout   int
}

type tmsRequestBody struct {
	Content string `json:"Content"`
	BizType string `json:"BizType,omitempty"`
}

type tmsResponseError struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

type tmsResponseBody struct {
	RequestId  string            `json:"RequestId"`
	Suggestion string            `json:"Suggestion"`
	Label      string            `json:"Label"`
	SubLabel   string            `json:"SubLabel"`
	Score      int               `json:"Score"`
	Keywords   []string          `json:"Keywords"`
	Error      *tmsResponseError `json:"Error"`
}

type tmsResponseWrapper struct {
	Response tmsResponseBody `json:"Response"`
}

// TMSResult ～这就是天御大人审完之后交回来的判定小卡片～
type TMSResult struct {
	Suggestion string   // Block / Review / Pass
	Label      string   // 一级风险标签
	SubLabel   string   // 二级风险标签
	Score      int      // 置信度 0-100
	Keywords   []string // 命中的敏感词
}

// ModerateText ～把文本悄悄递给腾讯云内容安全（TMS）把把关，返回真实的审核结论～
func ModerateText(cfg TMSConfig, text string) (*TMSResult, error) {
	if cfg.SecretId == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("腾讯云内容安全未配置 SecretId/SecretKey")
	}
	region := cfg.Region
	if region == "" {
		region = "ap-guangzhou"
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5
	}

	reqBody := tmsRequestBody{
		Content: base64.StdEncoding.EncodeToString([]byte(text)),
		BizType: cfg.BizType,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Unix()
	host := tmsServiceName + ".tencentcloudapi.com"
	authorization := signTC3(cfg.SecretId, cfg.SecretKey, tmsServiceName, tmsAction, string(payload), timestamp)

	httpReq, err := http.NewRequest(http.MethodPost, "https://"+host, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Host", host)
	httpReq.Header.Set("X-TC-Action", tmsAction)
	httpReq.Header.Set("X-TC-Timestamp", strconv.FormatInt(timestamp, 10))
	httpReq.Header.Set("X-TC-Version", tmsVersion)
	httpReq.Header.Set("X-TC-Region", region)
	httpReq.Header.Set("Authorization", authorization)

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var wrapper tmsResponseWrapper
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("解析腾讯云内容安全响应失败: %w", err)
	}
	if wrapper.Response.Error != nil {
		return nil, fmt.Errorf("腾讯云内容安全接口错误: [%s] %s", wrapper.Response.Error.Code, wrapper.Response.Error.Message)
	}

	return &TMSResult{
		Suggestion: wrapper.Response.Suggestion,
		Label:      wrapper.Response.Label,
		SubLabel:   wrapper.Response.SubLabel,
		Score:      wrapper.Response.Score,
		Keywords:   wrapper.Response.Keywords,
	}, nil
}
