package common

import (
	"net/http"
	"time"
)

// HTTPClient 全局 HTTP 客户端，设置超时
var HTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}
