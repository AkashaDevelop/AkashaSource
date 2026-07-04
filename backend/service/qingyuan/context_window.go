package qingyuan

import (
)

// detectContextWindowRisks 检测上下文窗口末期注入风险
// 设计文档 4.9 & 6.6: 上下文窗口末期注入
func detectContextWindowRisks(segments []textSegment, totalTokenEstimate int, maxContext int) []Finding {
	findings := []Finding{}

	if maxContext <= 0 {
		maxContext = 128000 // 默认上下文长度
	}

	// 计算上下文使用率
	usageRatio := float64(totalTokenEstimate) / float64(maxContext)

	// 超过 80% 时提升整体风险
	if usageRatio > 0.8 {
		score := 20
		if usageRatio > 0.95 {
			score = 35
		}

		findings = append(findings, Finding{
			Type:     "context_window_overflow",
			Severity: severity(score),
			Score:    score,
			Path:     "messages",
			Evidence: "context usage exceeds threshold",
			Action:   "monitor",
		})
	}

	// 重点检测最后几条用户消息
	userMessageCount := 0
	for i := len(segments) - 1; i >= 0; i-- {
		seg := segments[i]
		if seg.Role == "user" {
			userMessageCount++
			if userMessageCount <= 3 {
				// 标记为需要重点检测
				findings = append(findings, Finding{
					Type:     "last_user_message_focus",
					Severity: "info",
					Score:    0,
					Path:     seg.Path,
					Evidence: "last user message requires enhanced scanning",
					Action:   "monitor",
				})
			}
			if userMessageCount >= 3 {
				break
			}
		}
	}

	return findings
}

// estimateTokenCount 快速估算 token 数量
func estimateTokenCount(text string) int {
	// 英文: 字符数 / 4
	// 中文: 字符数 / 1.5
	// 混合: 使用加权平均

	if text == "" {
		return 0
	}

	runes := []rune(text)
	totalRunes := len(runes)

	if totalRunes == 0 {
		return 0
	}

	// 统计 CJK 字符
	cjkCount := 0
	for _, r := range runes {
		if isCJK(r) {
			cjkCount++
		}
	}

	// CJK 字符按 1:1.5 计算, ASCII 按 1:4 计算
	asciiCount := totalRunes - cjkCount
	estimatedTokens := int(float64(cjkCount)/1.5 + float64(asciiCount)/4.0)

	return estimatedTokens
}

// isCJK 判断是否为中日韩字符
func isCJK(r rune) bool {
	// CJK Unified Ideographs: U+4E00 - U+9FFF
	// CJK Extension A: U+3400 - U+4DBF
	// CJK Extension B+: U+20000 - U+2A6DF
	// Hiragana: U+3040 - U+309F
	// Katakana: U+30A0 - U+30FF
	// Hangul: U+AC00 - U+D7AF

	return (r >= 0x4e00 && r <= 0x9fff) ||
		(r >= 0x3400 && r <= 0x4dbf) ||
		(r >= 0x20000 && r <= 0x2a6df) ||
		(r >= 0x3040 && r <= 0x309f) ||
		(r >= 0x30a0 && r <= 0x30ff) ||
		(r >= 0xac00 && r <= 0xd7af)
}

// applyContextWindowWeighting 根据上下文位置调整风险分
func applyContextWindowWeighting(findings []Finding, seg textSegment, isLastUserMessage bool, contextUsageRatio float64) []Finding {
	if len(findings) == 0 {
		return findings
	}

	weighted := make([]Finding, len(findings))
	copy(weighted, findings)

	for i := range weighted {
		originalScore := weighted[i].Score

		// 最后用户消息加权 1.5x
		if isLastUserMessage && seg.Role == "user" {
			weighted[i].Score = int(float64(originalScore) * 1.5)
		}

		// 上下文接近满时加权
		if contextUsageRatio > 0.8 {
			multiplier := 1.2
			if contextUsageRatio > 0.95 {
				multiplier = 1.5
			}
			weighted[i].Score = int(float64(weighted[i].Score) * multiplier)
		}

		// 更新 severity
		weighted[i].Severity = severity(weighted[i].Score)
	}

	return weighted
}

// isLastUserMessage 判断当前 segment 是否为最后的用户消息
func isLastUserMessage(segments []textSegment, currentIndex int) bool {
	if currentIndex < 0 || currentIndex >= len(segments) {
		return false
	}

	current := segments[currentIndex]
	if current.Role != "user" {
		return false
	}

	// 检查后面是否还有用户消息
	for i := currentIndex + 1; i < len(segments); i++ {
		if segments[i].Role == "user" {
			return false
		}
	}

	return true
}
