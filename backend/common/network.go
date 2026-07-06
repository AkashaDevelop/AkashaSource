package common

import (
	"os"
	"strings"
)

const trustedProxiesEnvKey = "AKASHA_TRUSTED_PROXIES"

// TrustedProxies 返回可信的反向代理 CIDR/IP 列表，喂给 gin 的 SetTrustedProxies 用～
// 默认只信任本机回环地址（同机反代最常见场景），避免 gin 默认"信任所有代理"导致
// X-Forwarded-For 可被伪造、绕过基于 IP 的登录限流。
// 通过环境变量 AKASHA_TRUSTED_PROXIES（逗号分隔 IP/CIDR）可覆盖为内网网段等；
// 显式设为 "none" 则完全不信任任何代理头。
func TrustedProxies() []string {
	val := strings.TrimSpace(os.Getenv(trustedProxiesEnvKey))
	if val == "" {
		return []string{"127.0.0.1", "::1"}
	}
	if val == "none" {
		return nil
	}
	parts := strings.Split(val, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			result = append(result, p)
		}
	}
	return result
}
