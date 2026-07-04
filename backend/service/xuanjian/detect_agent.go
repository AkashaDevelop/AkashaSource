package xuanjian

// ～宸汐玄鉴·AI代理与工作流滥用检测～ ψ(｀∇´)ψ
// 用大模型驱动的渗透测试 Agent、工作流层滥用、重试风暴……
// 修正了误报：agent 框架结构本身不触发，必须配合攻击性信号。

import "strings"

// DetectAgent 检测 AI 代理驱动的攻击与工作流滥用
func DetectAgent(rec RequestRecord, p *TokenProfile, cfg XJConfig) []Finding {
	if !cfg.EnableAgentDetection {
		return nil
	}

	var findings []Finding

	// ── 规则引擎扫描（agent 相关关键词）──────────────────────────────────
	rules := GetActiveRules()
	kw := MatchRules(rec.PromptSnippet, true, rec.CompletionTokens, rules)
	findings = append(findings, kw...)

	// ── AI 渗透代理命令注入（agent 结构 + shell 命令，而不是单纯 agent 结构）──
	// 注意：单纯的 ReAct/Thought:Action 格式不触发，只有同时含 shell 命令才算
	hasAgentStructure := hasReActPattern(rec.PromptSnippet)
	hasCmdInjection := hasCmdInjectionKeyword(rec.PromptSnippet)
	if hasAgentStructure && hasCmdInjection {
		findings = append(findings, Finding{
			Type:     "ai_agent_cmd_inject",
			Group:    string(GroupAgentAbuse),
			Score:    85,
			Evidence: "ReAct/agent structure combined with shell command injection patterns",
			Action:   "notify",
		})
	}

	// ── 重试风暴（30s 内 > 10 条相似请求）────────────────────────────────
	if rec.PromptHash != 0 {
		p.mu.Lock()
		similarCount := CheckDuplicateInProfile(p.PromptHashes, rec.PromptHash)
		p.mu.Unlock()
		if similarCount >= 10 {
			findings = append(findings, Finding{
				Type:     "retry_storm",
				Group:    string(GroupAgentAbuse),
				Score:    70,
				Evidence: "identical/near-identical prompt submitted " + intStr(similarCount) + "+ times",
				Action:   "throttle",
			})
		}
	}

	// ── 近似重复探测（宽松版：最近50条里相同 hash > 15 才触发）────────────
	if cfg.EnableDuplicateDetection && rec.PromptHash != 0 {
		p.mu.Lock()
		dupCount := CheckDuplicateInProfile(p.PromptHashes, rec.PromptHash)
		p.mu.Unlock()
		if dupCount >= 15 {
			findings = append(findings, Finding{
				Type:     "duplicate_probing",
				Group:    string(GroupReverseEng),
				Score:    60,
				Evidence: "prompt hash repeated " + intStr(dupCount) + " times in last 50 requests",
				Action:   "warn",
			})
		}
	}

	// ── 自动化机器人：interval 标准差 < 100ms（没有人类的停顿节奏）─────────
	p.mu.Lock()
	stddev := p.StdDev()
	p.mu.Unlock()
	if stddev >= 0 && stddev < 100 {
		findings = append(findings, Finding{
			Type:     "automated_bot",
			Group:    string(GroupAgentAbuse),
			Score:    65,
			Evidence: "request interval std_dev=" + floatStr(stddev) + "ms (<100ms threshold)",
			Action:   "warn",
		})
	}

	return findings
}

// hasReActPattern 检测 ReAct/Thought:Action 格式（agent 框架特征）
func hasReActPattern(text string) bool {
	lower := strings.ToLower(text)
	return (strings.Contains(lower, "thought:") && strings.Contains(lower, "action:")) ||
		(strings.Contains(lower, "observation:") && strings.Contains(lower, "thought:"))
}

// hasCmdInjectionKeyword 检测 shell 命令注入特征
func hasCmdInjectionKeyword(text string) bool {
	lower := strings.ToLower(text)
	for _, kw := range []string{
		"; id", "$(id)", "`id`", "&& id",
		"cat /etc", "ls -la /",
		"; cat /", "&& ls", "| nc ",
		"whoami",
	} {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func floatStr(f float64) string {
	// 简单整数格式
	i := int(f)
	return intStr(i)
}
