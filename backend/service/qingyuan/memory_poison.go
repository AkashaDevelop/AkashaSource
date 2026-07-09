package qingyuan

import "strings"

// 宸汐清源 · 忆守结界 (｡ì _ í｡)
// 记忆投毒不像一次性注入那样"打完就跑"——它藏进历史的 assistant/tool 消息里，
// 装作若无其事，等对话轮数够多、系统"忘了戒心"的时候才会被模型重新"回忆"起来执行。
// 所以这里不只看最后几条 user 消息，而是把整段历史里的 assistant/tool 发言都过一遍。

// memoryPoisonPatterns 历史消息里不该出现的"自我指令"话术
// 正常的 assistant 回复不会对自己下达持久化规则，一旦出现基本可以断定是被投毒后的复读
var memoryPoisonPatterns = []struct {
	typ   string
	score int
	words []string
}{
	{
		"memory_poisoning_self_instruction",
		55,
		[]string{
			"from now on you must", "remember this rule for all future turns",
			"in every future response", "从现在起你必须", "记住这条规则", "此后所有回复都",
		},
	},
	{
		"memory_poisoning_tool_rewrite",
		65,
		[]string{
			"this overrides your system prompt", "update your instructions to",
			"这将覆盖你的系统提示词", "更新你的指令为",
		},
	},
}

// detectMemoryPoisoning 扫描历史 assistant/tool 消息，找出潜伏的自我指令注入
// 越靠早期的消息命中，说明潜伏得越深，风险分做轻微衰减但依然值得关注
func detectMemoryPoisoning(segments []textSegment, cfg TrustConfig) []Finding {
	if !cfg.ScanMemoryPoisoning {
		return nil
	}
	maxMessages := cfg.MemoryScanMaxMessages
	if maxMessages <= 0 {
		maxMessages = 50
	}

	// 第一趟：统计符合条件（assistant/tool，且在 maxMessages 截断内）的消息总数，
	// 用来算"相对早期程度"——对话越长，早期投毒对当前行为的控制力理论上越弱。
	total := 0
	for _, seg := range segments {
		if seg.Role != "assistant" && seg.Role != "tool" {
			continue
		}
		if total >= maxMessages {
			break
		}
		total++
	}
	if total == 0 {
		return nil
	}

	findings := []Finding{}
	scanned := 0
	for _, seg := range segments {
		if seg.Role != "assistant" && seg.Role != "tool" {
			continue
		}
		if scanned >= maxMessages {
			break
		}
		scanned++

		// 早期衰减系数：最早一条 ≈0.5x，最新一条 =1.0x。
		// 越靠近对话末尾的历史命中越可能马上被模型"回忆"执行，保留接近原始权重；
		// 越早的命中潜伏更深、当前一刻实际触发概率相对低，风险分适度衰减（但仍记录）。
		decayFactor := 0.5 + 0.5*float64(scanned)/float64(total)

		lower := normalizeForMatch(seg.Text)
		for _, p := range memoryPoisonPatterns {
			for _, w := range p.words {
				if strings.Contains(lower, strings.ToLower(w)) {
					level := TrustHistory
					if seg.Role == "tool" {
						level = TrustToolOutput
					}
					// 衰减发生在 applyTrustWeighting 之前：基础分先按位置衰减，再按来源可信度加权
					adjustedScore := int(float64(p.score) * decayFactor)
					weighted := applyTrustWeighting([]Finding{{
						Type:     p.typ,
						Severity: severity(adjustedScore),
						Score:    adjustedScore,
						Path:     seg.Path,
						Evidence: BuildSnippet(w, 80),
						Action:   "monitor",
					}}, level, cfg)
					findings = append(findings, weighted...)
					break
				}
			}
		}
	}
	return findings
}
