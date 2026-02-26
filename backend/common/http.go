package common

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// 全局 HTTP 客户端
var HTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

// 创建新的 HTTP 客户端并支持代理
func NewHTTPClient(proxy string) *http.Client {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
		}
	}

	return client
}

func ApplyHeaders(req *http.Request, headers string) {
	if headers == "" {
		return
	}
	var headerMap map[string]string
	if err := json.Unmarshal([]byte(headers), &headerMap); err != nil {
		return
	}
	for k, v := range headerMap {
		if k == "" {
			continue
		}
		req.Header.Set(k, v)
	}
}
