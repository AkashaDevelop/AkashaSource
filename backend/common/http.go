package common

import (
	"net/http"
	"net/url"
	"time"
)

// HTTPClient 全局 HTTP 客户端，设置超时
var HTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

// NewHTTPClient 创建一个新的 HTTP 客户端，支持代理
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
