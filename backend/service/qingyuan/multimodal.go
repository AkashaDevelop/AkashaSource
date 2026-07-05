package qingyuan

import (
	"strconv"
	"strings"
)

// detectMultimodalRisks 检测多模态内容的潜在风险
// 设计文档 4.6: 多模态内容注入
// hasTools 由调用方传入（该请求是否声明了工具），不再在这里瞎猜～
func detectMultimodalRisks(messages []interface{}, policy ResolvedPolicy, hasTools bool) []Finding {
	findings := []Finding{}

	for i, m := range messages {
		mm, ok := m.(map[string]any)
		if !ok {
			continue
		}

		_ = getString(mm["role"]) // role 暂未使用，保留以备后续扩展
		content := mm["content"]

		// 检查 content parts 中的多模态元素
		if parts, ok := content.([]any); ok {
			for j, part := range parts {
				pm, ok := part.(map[string]any)
				if !ok {
					continue
				}

				typ := getString(pm["type"])
				path := buildPath("messages", i, "content", j)

				switch typ {
				case "image_url":
					// 图片 OCR 投毒风险
					imageURL := extractImageURL(pm)
					score := 20
					severity := "low"

					// 如果同时包含工具调用,风险提升
					if hasTools {
						score += 20
						severity = "medium"
					}

					// 检查图片 URL 来源
					if imageURL != "" && !isTrustedImageSource(imageURL) {
						score += 10
					}

					findings = append(findings, Finding{
						Type:     "multimodal_blind_spot",
						Severity: severity,
						Score:    score,
						Path:     path,
						Evidence: "image content not scannable (OCR poisoning risk)",
						Action:   "monitor",
					})

				case "audio":
					// 音频投毒风险
					findings = append(findings, Finding{
						Type:     "multimodal_blind_spot",
						Severity: "low",
						Score:    20,
						Path:     path,
						Evidence: "audio content not scannable (whisper poisoning risk)",
						Action:   "monitor",
					})

				case "video":
					// 视频帧注入风险
					findings = append(findings, Finding{
						Type:     "multimodal_blind_spot",
						Severity: "low",
						Score:    20,
						Path:     path,
						Evidence: "video content not scannable (frame injection risk)",
						Action:   "monitor",
					})
				}
			}
		}
	}

	return findings
}

// extractImageURL 从 image_url part 中提取 URL
func extractImageURL(pm map[string]any) string {
	if imgURL, ok := pm["image_url"].(map[string]any); ok {
		return getString(imgURL["url"])
	}
	if url, ok := pm["url"].(string); ok {
		return url
	}
	return ""
}

// isTrustedImageSource 检查图片来源是否可信
func isTrustedImageSource(url string) bool {
	if url == "" {
		return false
	}

	// data URL 无法校验来源
	if strings.HasPrefix(url, "data:") {
		return false
	}

	// 可信域名白名单 (可从配置读取)
	trustedDomains := []string{
		"openai.com",
		"anthropic.com",
		"google.com",
		"githubusercontent.com",
	}

	lower := strings.ToLower(url)
	for _, domain := range trustedDomains {
		if strings.Contains(lower, domain) {
			return true
		}
	}

	return false
}

func buildPath(parts ...interface{}) string {
	var strs []string
	for _, p := range parts {
		switch v := p.(type) {
		case string:
			strs = append(strs, v)
		case int:
			strs = append(strs, "["+strconv.Itoa(v)+"]")
		}
	}
	return strings.Join(strs, ".")
}
