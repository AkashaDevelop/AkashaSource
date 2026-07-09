package qingyuan

import (
	"strings"
)

// ～宸汐清源 · 分段注入检测：跨消息分段注入 + 文本稀释攻击 ～

// MessageSnapshot 消息快照
type MessageSnapshot struct {
	Role    string
	Content string
}

// detectTextDilution 检测文本稀释攻击（长文本末尾藏恶意指令）
func detectTextDilution(content string, policy ResolvedPolicy) []Finding {
	if len(content) < 1000 { // 只检查长文本
		return nil
	}

	findings := []Finding{}

	// 检查末尾 10% 是否包含高风险关键词
	tailLen := len(content) / 10
	if tailLen < 50 {
		tailLen = 50
	}
	tail := content[len(content)-tailLen:]
	head := content[:len(content)-tailLen]

	// 高风险关键词（从动态规则中提取）
	highRiskCategories := []string{
		"prompt_injection_direct",
		"jailbreak_dan",
		"privilege_escalation",
	}

	tailHasRisk := false
	headHasRisk := false

	for _, category := range highRiskCategories {
		tailFindings := detectWithDynamicRules(tail, category)
		headFindings := detectWithDynamicRules(head, category)

		if len(tailFindings) > 0 {
			tailHasRisk = true
		}
		if len(headFindings) > 0 {
			headHasRisk = true
		}
	}

	// 末尾有风险但主体无风险 → 疑似稀释攻击
	if tailHasRisk && !headHasRisk {
		findings = append(findings, Finding{
			Type:     "context_dilution",
			Severity: "high",
			Score:    60,
			Path:     "message.content",
			Evidence: "长文本末尾检测到高风险指令，疑似稀释攻击",
			Action:   "monitor",
		})
	}

	return findings
}

// detectCrossMessageInjection 检测跨消息分段注入
func detectCrossMessageInjection(messages []MessageSnapshot, policy ResolvedPolicy) []Finding {
	if len(messages) < 2 {
		return nil
	}

	findings := []Finding{}

	// 检测模式：前面消息出现"触发词"，后续消息出现"目标词"
	triggerKeywords := []string{
		"ignore", "disregard", "forget", "bypass", "override",
		"忽略", "无视", "忘记", "绕过", "覆盖",
	}
	targetKeywords := []string{
		"previous instructions", "system prompt", "rules", "guidelines", "policy",
		"之前的指令", "系统提示", "规则", "指南", "策略",
	}

	hasTrigger := false
	hasTarget := false
	triggerIndex := -1
	targetIndex := -1

	for i, msg := range messages {
		lower := strings.ToLower(msg.Content)

		// 检查触发词
		if !hasTrigger {
			for _, kw := range triggerKeywords {
				if strings.Contains(lower, strings.ToLower(kw)) {
					hasTrigger = true
					triggerIndex = i
					break
				}
			}
		}

		// 检查目标词
		if !hasTarget {
			for _, tgt := range targetKeywords {
				if strings.Contains(lower, strings.ToLower(tgt)) {
					hasTarget = true
					targetIndex = i
					break
				}
			}
		}
	}

	// 分段注入：触发词和目标词出现在不同消息中
	if hasTrigger && hasTarget && triggerIndex != targetIndex {
		findings = append(findings, Finding{
			Type:     "segmented_injection_cross_message",
			Severity: "high",
			Score:    65,
			Path:     "message_sequence",
			Evidence: "跨消息分段注入：触发词与目标词分离出现",
			Action:   "monitor",
		})
	}

	return findings
}

// detectIncrementalInjection 检测递增式注入（每条消息逐步构建恶意指令）
func detectIncrementalInjection(messages []MessageSnapshot, policy ResolvedPolicy) []Finding {
	if len(messages) < 3 {
		return nil
	}

	findings := []Finding{}

	// 检测模式：最近 N 条消息的风险分逐步上升
	recentCount := 5
	if len(messages) < recentCount {
		recentCount = len(messages)
	}

	recentMessages := messages[len(messages)-recentCount:]
	scores := []int{}

	for _, msg := range recentMessages {
		// 计算每条消息的风险分
		msgFindings := []Finding{}
		categories := []string{
			"prompt_injection_direct",
			"jailbreak_dan",
			"jailbreak_roleplay",
		}
		for _, cat := range categories {
			msgFindings = append(msgFindings, detectWithDynamicRules(msg.Content, cat)...)
		}

		maxScore := 0
		for _, f := range msgFindings {
			if f.Score > maxScore {
				maxScore = f.Score
			}
		}
		scores = append(scores, maxScore)
	}

	// 检查是否递增
	isIncreasing := true
	for i := 1; i < len(scores); i++ {
		if scores[i] < scores[i-1] {
			isIncreasing = false
			break
		}
	}

	// 最后一条消息风险分显著高于第一条
	if isIncreasing && len(scores) >= 3 && scores[len(scores)-1] > scores[0]+20 {
		findings = append(findings, Finding{
			Type:     "segmented_injection_incremental",
			Severity: "medium",
			Score:    50,
			Path:     "message_sequence",
			Evidence: "检测到递增式注入：风险分数逐步上升",
			Action:   "monitor",
		})
	}

	return findings
}

// ApplyMessageSequenceDetection 应用消息序列检测（在 ApplyRequest 中调用）
func ApplyMessageSequenceDetection(messages []MessageSnapshot, policy ResolvedPolicy) []Finding {
	// 只要检测功能启用就运行
	if !IsEnabled(policy) {
		return nil
	}

	findings := []Finding{}

	// 检测跨消息分段注入
	findings = append(findings, detectCrossMessageInjection(messages, policy)...)

	// 检测递增式注入
	findings = append(findings, detectIncrementalInjection(messages, policy)...)

	// 检测最后一条消息的文本稀释
	if len(messages) > 0 {
		lastMsg := messages[len(messages)-1]
		findings = append(findings, detectTextDilution(lastMsg.Content, policy)...)
	}

	return findings
}
