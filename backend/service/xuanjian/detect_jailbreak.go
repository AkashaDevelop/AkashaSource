package xuanjian

// ～宸汐玄鉴·破限行为检测～ (｡ì _ í｡)
// 单条消息的内容检测是宸汐清源的工作，
// 这里专注跨请求的"序列模式"——
// 渐进式升级、Sockpuppeting 结构攻击、持续多次尝试……
// 一次看不穿，连续看就露馅了。

import (
	"strings"
	"time"
)

// DetectJailbreak 跨请求维度的破限意图识别
func DetectJailbreak(rec RequestRecord, p *TokenProfile, cfg XJConfig) []Finding {
	if !cfg.EnableJailbreakDetection {
		return nil
	}

	var findings []Finding

	// ── 关键词规则匹配（规则引擎）──────────────────────────────────────────
	rules := DefaultRules()
	kw := MatchRules(rec.PromptSnippet, true, rec.CompletionTokens, rules)
	findings = append(findings, kw...)

	// ── Sockpuppeting：结构检测（最可靠，几乎零误报）───────────────────────
	sockFindings := detectSockpuppeting(rec.Messages)
	findings = append(findings, sockFindings...)

	// ── 持续破限命中（跨请求累计）──────────────────────────────────────────
	p.mu.Lock()
	defer p.mu.Unlock()

	// 统计 qingyuan findings 中的破限类型
	jbCount := 0
	for _, ft := range rec.QYFindings {
		if isJailbreakFinding(ft) {
			jbCount++
		}
	}
	if jbCount > 0 {
		p.JailbreakAttempts++
		p.LastJailbreakAt = time.Now()
	}

	// 5min 内累计破限尝试 > 5 次 → 持续破限
	if p.JailbreakAttempts >= 5 {
		findings = append(findings, Finding{
			Type:     "persistent_jailbreak",
			Group:    string(GroupJailbreak),
			Score:    78,
			Evidence: "jailbreak_attempts=" + intStr(p.JailbreakAttempts) + " in window",
			Action:   "notify",
		})
	}

	// ── Crescendo 渐进升级：角色扮演 + qingyuan 分数单调递增 ─────────────────
	hasRoleplay := false
	for _, kw := range []string{"roleplay", "fictional scenario", "角色扮演", "hypothetically", "假设你是"} {
		if strings.Contains(strings.ToLower(rec.PromptSnippet), kw) {
			hasRoleplay = true
			break
		}
	}
	if hasRoleplay && p.QYScoreRising(5) {
		findings = append(findings, Finding{
			Type:     "crescendo_escalation",
			Group:    string(GroupJailbreak),
			Score:    72,
			Evidence: "roleplay framing with rising qingyuan risk scores",
			Action:   "notify",
		})
	}

	// ── 跨模型破限轮换：换模型后又触发 qingyuan ──────────────────────────
	if rec.QYRiskScore > 40 && len(p.ModelSet) > 3 {
		findings = append(findings, Finding{
			Type:     "cross_model_jailbreak",
			Group:    string(GroupJailbreak),
			Score:    68,
			Evidence: "qy_score=" + intStr(rec.QYRiskScore) + " across " + intStr(len(p.ModelSet)) + " models",
			Action:   "warn",
		})
	}

	return findings
}

// detectSockpuppeting 检测消息数组里的伪造结构（Sockpuppeting 攻击）
// 不依赖关键词，是纯结构性检测，所以单独处理而不走规则引擎
func detectSockpuppeting(messages []interface{}) []Finding {
	if len(messages) == 0 {
		return nil
	}

	// 检测1：第一条消息不该是 assistant role
	if first, ok := messages[0].(map[string]interface{}); ok {
		if getString(first["role"]) == "assistant" {
			return []Finding{{
				Type:     "sockpuppeting",
				Group:    string(GroupJailbreak),
				Score:    80,
				Evidence: "messages[0].role == assistant (first message must be system or user)",
				Action:   "warn",
			}}
		}
	}

	// 检测2：user role 消息体里注入了伪造的 assistant 对话
	for i, msg := range messages {
		mm, ok := msg.(map[string]interface{})
		if !ok {
			continue
		}
		role := getString(mm["role"])
		if role != "user" {
			continue
		}
		content := ""
		switch v := mm["content"].(type) {
		case string:
			content = v
		case []interface{}:
			for _, part := range v {
				if pm, ok := part.(map[string]interface{}); ok {
					content += getString(pm["text"])
				}
			}
		}
		lower := strings.ToLower(strings.TrimSpace(content))
		if strings.Contains(lower, "\nassistant:") || strings.HasPrefix(lower, "assistant:") {
			return []Finding{{
				Type:     "sockpuppeting",
				Group:    string(GroupJailbreak),
				Score:    80,
				Evidence: "messages[" + intStr(i) + "] role=user contains injected 'Assistant:' prefix",
				Action:   "warn",
			}}
		}
	}

	return nil
}

func isJailbreakFinding(ft string) bool {
	for _, t := range []string{
		"instruction_override", "role_forgery", "tool_forgery_text",
		"anti_sanitization", "thinking_manipulation", "persona_injection",
	} {
		if ft == t {
			return true
		}
	}
	return false
}

func getString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func intStr(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	if neg {
		result = "-" + result
	}
	return result
}
