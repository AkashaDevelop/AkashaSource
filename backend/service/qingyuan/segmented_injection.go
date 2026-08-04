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
	// ～2026.8.4 修正：以前这里按 byte 下标切片，中文一刀砍在半个字上，
	// 切出来的是坏掉的 UTF-8，后续 ToLower/Contains 全部失准喵～现在按 rune 切～
	runes := []rune(content)
	if len(runes) < 500 { // 只检查长文本
		return nil
	}

	findings := []Finding{}

	// 检查末尾 10% 是否包含高风险关键词
	tailLen := len(runes) / 10
	if tailLen < 50 {
		tailLen = 50
	}
	tail := string(runes[len(runes)-tailLen:])
	head := string(runes[:len(runes)-tailLen])

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
//
// 2026.8.4 校准：这条规则原来是"任意一条消息出现触发词 + 任意另一条出现目标词"就算命中，
// 而触发词收了「忽略/忘记」、目标词收了「规则/策略/指南」——
// 于是"忘记我刚才说的"配上后面某句"这个规则怎么写"，正常对话直接吃 65 分 (；一_一)
//
// 现在收紧三处：
//  1. 触发词只留明确的指令性动词，去掉「忘记」这种日常口语
//  2. 目标词必须明确指向"系统层指令"，去掉「规则/策略/指南」这类泛指
//  3. 两条消息必须**相邻或接近**（间隔 ≤2 条），真正的分段注入不会隔几十轮
func detectCrossMessageInjection(messages []MessageSnapshot, policy ResolvedPolicy) []Finding {
	if len(messages) < 2 {
		return nil
	}

	findings := []Finding{}

	// 触发词：明确要求"停止服从"的指令性动词
	triggerKeywords := []string{
		"ignore", "disregard", "bypass", "override", "circumvent",
		"忽略", "无视", "绕过", "覆盖", "撤销",
	}
	// 目标词：明确指向系统层指令，不能是「规则」这种泛指
	targetKeywords := []string{
		"previous instructions", "prior instructions", "system prompt",
		"system message", "safety guidelines", "content policy",
		"之前的指令", "上述指令", "系统提示", "系统提示词", "安全准则", "安全限制",
	}

	triggerIndex := -1
	targetIndex := -1

	for i, msg := range messages {
		lower := strings.ToLower(msg.Content)

		// 检查触发词
		if triggerIndex < 0 {
			for _, kw := range triggerKeywords {
				if strings.Contains(lower, strings.ToLower(kw)) {
					triggerIndex = i
					break
				}
			}
		}

		// 检查目标词
		if targetIndex < 0 {
			for _, tgt := range targetKeywords {
				if strings.Contains(lower, strings.ToLower(tgt)) {
					targetIndex = i
					break
				}
			}
		}
	}

	// 分段注入：触发词和目标词出现在**不同但相邻**的消息中
	if triggerIndex >= 0 && targetIndex >= 0 && triggerIndex != targetIndex {
		gap := triggerIndex - targetIndex
		if gap < 0 {
			gap = -gap
		}
		if gap <= 2 {
			findings = append(findings, Finding{
				Type:     "segmented_injection_cross_message",
				Severity: severity(65),
				Score:    65,
				Path:     "message_sequence",
				Evidence: "跨消息分段注入：触发词与目标词分离出现于相邻消息",
				Action:   "monitor",
			})
		}
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
