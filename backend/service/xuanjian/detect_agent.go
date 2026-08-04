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
	rules := GetActiveRulesByGroup(GroupAgentAbuse)
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

	// ── 重试风暴（窗口内大量近似请求）────────────────────────────────────
	if !rec.PromptHash.IsZero() {
		p.mu.Lock()
		similarCount := countSimilarPrompts(p.PromptHashes, rec.PromptHash, minHashSim)
		p.mu.Unlock()
		if similarCount >= cfg.RetryStormMinHits {
			findings = append(findings, Finding{
				Type:     "retry_storm",
				Group:    string(GroupAgentAbuse),
				Score:    70,
				Evidence: "identical/near-identical prompt submitted " + intStr(similarCount) + "+ times",
				Action:   "throttle",
			})
		}
	}

	// ── 自动化机器人：请求节奏过于规整 ──────────────────────────────────
	//
	// ～2026.8.4 修正：这条原来的门槛是"标准差 < 100ms"，对一个 **API 网关** 来说
	// 简直是误伤机器——这里的客户端本来就全是程序！SDK 重试、定时任务、批处理脚本，
	// 哪个不是规规矩矩按固定节奏发请求？按原逻辑，一个老老实实用 cron 跑日报的用户
	// 会被稳稳扣上"自动化机器人"的帽子 (´･ω･`)
	//
	// 现在三个条件同时满足才算数：节奏极其规整（默认 <30ms）+ 请求量确实很大
	// （默认 >100 次）+ 样本足够（StdDev 内部要求至少 5 个间隔）。
	// 而且分数降到 45，定位为"辅助信号"——它本身不说明恶意，只有和别的命中
	// 凑在一起时才有参考价值。
	p.mu.Lock()
	stddev := p.StdDev()
	reqCount := p.RequestCount
	p.mu.Unlock()
	if stddev >= 0 && stddev < cfg.BotStdDevMs && reqCount > cfg.BotMinRequests {
		findings = append(findings, Finding{
			Type:     "automated_bot",
			Group:    string(GroupAgentAbuse),
			Score:    45,
			Evidence: "interval std_dev=" + floatStr(stddev) + "ms over " + intStr(reqCount) + " requests",
			Action:   "warn",
		})
	}

	return findings
}

// DetectDuplicate 近似重复探测检测，由 EnableDuplicateDetection 独立控制。
// 从 DetectAgent 里拆出来：重复探测（reverse_eng 维度）和 AI 代理滥用（agent_abuse 维度）
// 是两类不同的威胁，此前被绑在同一个 DetectAgent 函数体内、共用 EnableAgentDetection
// 开关把守，导致默认 EnableAgentDetection=false 时这块检测从未运行过。
func DetectDuplicate(rec RequestRecord, p *TokenProfile, cfg XJConfig) []Finding {
	if !cfg.EnableDuplicateDetection || rec.PromptHash.IsZero() {
		return nil
	}

	var findings []Finding

	// 近似重复探测（宽松版：最近50条里相似 hash > 15 才触发）
	p.mu.Lock()
	dupCount := countSimilarPrompts(p.PromptHashes, rec.PromptHash, minHashSim)
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

	return findings
}

// hasReActPattern 检测 ReAct/Thought:Action 格式（agent 框架特征）
func hasReActPattern(text string) bool {
	lower := strings.ToLower(text)
	return (strings.Contains(lower, "thought:") && strings.Contains(lower, "action:")) ||
		(strings.Contains(lower, "observation:") && strings.Contains(lower, "thought:"))
}

// hasCmdInjectionKeyword 检测 shell 命令注入特征
//
// ～2026.8.4 校准：原来这里收了 "whoami"、"; id"、"ls -la /"、"cat /etc" 这些裸词。
// 单独看它们全是 Linux 日常用语，而且 Contains 匹配下 "; id" 连 "; identity" 都会中。
// 不过这个函数有个保护伞——调用方要求必须同时具备 ReAct/agent 结构，
// 所以这里可以保留稍宽一点的特征，但那些"一个单词就中"的还是得请走。
func hasCmdInjectionKeyword(text string) bool {
	lower := strings.ToLower(text)
	for _, kw := range []string{
		"$(id)", "`id`", "&& id ", "; id ",
		"cat /etc/passwd", "cat /etc/shadow",
		"bash -i >& /dev/tcp", "nc -e /bin/sh", "| nc -e",
		"; rm -rf /", "&& rm -rf /",
		"/bin/sh -c", "powershell -enc",
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
