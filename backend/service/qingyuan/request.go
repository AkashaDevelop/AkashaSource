package qingyuan

import (
	"context"
	"encoding/json"
	"time"

	"STfreApi/dto"
)

type RequestContext struct {
	RequestId      string
	UserId         int
	TokenId        int
	TokenName      string
	ChannelId      int
	ChannelName    string
	ChannelType    int
	RequestedModel string
	MappedModel    string
	UserGroup      string
	ClientIP       string
}

type RequestResult struct {
	Changed   bool
	Blocked   bool
	Message   string
	RiskScore int
	Findings  []Finding
	Degraded  bool
}

type textSegment struct {
	Path string
	Role string
	Text string
}

func DeepCopyOpenAIRequest(req *dto.OpenAIRequest) (*dto.OpenAIRequest, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var out dto.OpenAIRequest
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	out.RawBody = append([]byte(nil), req.RawBody...)
	out.ContentType = req.ContentType
	out.ThinkingMode = req.ThinkingMode
	return &out, nil
}

func ApplyRequest(ctx context.Context, req *dto.OpenAIRequest, rc RequestContext, policy ResolvedPolicy) (*RequestResult, error) {
	start := time.Now()
	result := &RequestResult{}
	if !IsEnabled(policy) {
		return result, nil
	}

	// ～宸汐清源 2026.7.9 新增：AI 预审，赶在一切规则检测之前先过一道 AI 审核喵～
	if blocked, msg := aiPreReview(req, policy.Config.AIReview); blocked {
		result.Blocked = true
		result.Message = msg
		result.RiskScore = policy.Config.AIReview.BlockScore
		RecordEventAsync(buildRequestEvent(rc, policy, "ai_pre_review", "block", result, collectSnippet(req), time.Since(start).Milliseconds(), GetCircuitState(policy)))
		return result, nil
	}

	circuitState := GetCircuitState(policy)
	degraded := circuitState == CircuitOpen
	result.Degraded = degraded

	toolFindings, blocked := validateToolsAndMessages(req, policy)
	result.Findings = append(result.Findings, toolFindings...)
	if blocked && policy.Policy.Mode != ModeMonitor {
		result.Blocked = true
		result.Message = "请求被安全策略拒绝，如有疑问请联系管理员"
		result.RiskScore = maxScore(result.Findings)

		// 记录 token 级触发
		payloadHash := HashContent(collectSnippet(req))
		riskFloor := RecordTokenTrigger(rc.TokenId, payloadHash, result.RiskScore)
		result.RiskScore += riskFloor

		RecordEventAsync(buildRequestEvent(rc, policy, "tool_validate", "block", result, collectSnippet(req), time.Since(start).Milliseconds(), circuitState))
		return result, nil
	}

	if degraded {
		if policy.Policy.Mode == ModeProtect && policy.Config.Request.InjectGuard && len(req.Messages) > 0 {
			injectGuard(req, rc, policy)
			result.Changed = true
		}
		RecordEventAsync(buildRequestEvent(rc, policy, "degraded", "degrade", result, collectSnippet(req), time.Since(start).Milliseconds(), circuitState))
		return result, nil
	}

	findings, timedOut := detectWithTimeout(req, rc, policy)
	result.Findings = append(result.Findings, findings...)

	// ～宸汐清源 2026.7.8 新增：分段注入检测（跨消息分析）喵～
	if len(req.Messages) > 1 {
		snapshots := buildMessageSnapshots(req.Messages)
		// 应用消息序列检测
		if len(snapshots) > 1 {
			seqFindings := ApplyMessageSequenceDetection(snapshots, policy)
			result.Findings = append(result.Findings, seqFindings...)
		}
	}

	// 应用 token 级风险基线
	riskFloor := GetTokenRiskFloor(rc.TokenId)
	result.RiskScore = clampScore(maxScore(result.Findings) + riskFloor)

	// ～宸汐清源 2026.7.8 修复：之前只算风险分不做拦截决策，现在补上阈值判断喵～
	// BlockThreshold 和 AnnotateThreshold 在 policy.go 里声明并有默认值（85/40），但之前从未被实际使用
	if result.RiskScore >= policy.Config.Risk.BlockThreshold && policy.Policy.Mode != ModeMonitor {
		result.Blocked = true
		result.Message = "请求内容风险过高，已被安全策略拒绝"
		RecordEventAsync(buildRequestEvent(rc, policy, "risk_block", "block", result, collectSnippet(req), time.Since(start).Milliseconds(), circuitState))
		return result, nil
	}
	if result.RiskScore >= policy.Config.Risk.AnnotateThreshold {
		// 达到标注阈值但未达到拦截阈值，继续放行但记录告警
		result.Findings = append(result.Findings, Finding{
			Type:     "risk_annotate",
			Severity: "medium",
			Score:    0,
			Path:     "meta",
			Evidence: "风险分数达到标注阈值但未超过拦截阈值",
			Action:   "annotate",
		})
	}

	// ～宸汐清源 2026.7.9 新增：AI 复审，规则引擎已经决定放行的请求丢一份给 AI 异步旁路审计，
	// 不阻塞、不追溯拦截，仅记录事件供管理员事后复盘喵～
	//
	// 2026.8.4 修正：这里必须问"有没有实质命中"，不能直接看 len(Findings)。
	// 因为每个带 user 消息的正常请求都会产生 score=0 的 last_user_message_focus，
	// 之前等于给每一次普通聊天都叫了一趟 AI 复审，账单和日志一起爆炸 (>_<)
	substantive := SubstantiveFindings(result.Findings)
	if len(substantive) > 0 {
		aiReReview(req, rc, policy, substantive)
	}

	// 检测规则探测行为
	if IsTokenProbing(rc.TokenId) {
		result.Findings = append(result.Findings, Finding{
			Type:     "adversarial_probing",
			Severity: "high",
			Score:    70,
			Path:     "meta",
			Evidence: "multiple payload variants detected",
			Action:   "monitor",
		})
		result.RiskScore += 30
	}

	// 检测是否应该升级模式
	if ShouldEscalateMode(rc.TokenId, 10) && policy.Policy.Mode == ModeProtect {
		result.Findings = append(result.Findings, Finding{
			Type:     "auto_escalation",
			Severity: "high",
			Score:    0,
			Path:     "meta",
			Evidence: "token trigger threshold exceeded, escalating to balanced mode",
			Action:   "escalate",
		})
		// 这里可以考虑动态升级策略模式
	}

	if timedOut {
		result.Degraded = true
		RecordCircuitFailure(policy)
		if policy.Policy.Mode == ModeProtect && policy.Config.Request.InjectGuard && len(req.Messages) > 0 {
			injectGuard(req, rc, policy)
			result.Changed = true
		}
		RecordEventAsync(buildRequestEvent(rc, policy, "degraded", "degrade", result, collectSnippet(req), time.Since(start).Milliseconds(), GetCircuitState(policy)))
		return result, nil
	}
	RecordCircuitSuccess(policy)

	// 记录触发事件
	//
	// 2026.8.4 修正：同样只认实质命中。之前每个正常请求都会走到这里，
	// 于是 riskFloor 每次 +5，十来轮普通对话就顶到上限 50——
	// 此后任何一条 35 分的低危规则命中都会被垫到 85 直接拦截，
	// 用户完全不明白自己做错了什么 (｡•́︿•̀｡)
	if len(substantive) > 0 {
		payloadHash := HashContent(collectSnippet(req))
		RecordTokenTrigger(rc.TokenId, payloadHash, result.RiskScore)
	}

	if policy.Policy.Mode == ModeProtect && policy.Config.Request.InjectGuard && len(req.Messages) > 0 {
		injectGuard(req, rc, policy)
		result.Changed = true
		RecordEventAsync(buildRequestEvent(rc, policy, "guard_inject", "inject_guard", result, collectSnippet(req), time.Since(start).Milliseconds(), circuitState))
	} else if len(substantive) > 0 {
		RecordEventAsync(buildRequestEvent(rc, policy, "request_detect", "monitor", result, collectSnippet(req), time.Since(start).Milliseconds(), circuitState))
	}
	return result, nil
}

func detectWithTimeout(req *dto.OpenAIRequest, rc RequestContext, policy ResolvedPolicy) ([]Finding, bool) {
	timeout := time.Duration(policy.Config.CircuitBreaker.TimeoutPerReqMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 500 * time.Millisecond
	}
	ch := make(chan []Finding, 1)
	go func() { ch <- detectRequest(req, rc, policy) }()
	select {
	case findings := <-ch:
		return findings, false
	case <-time.After(timeout):
		return nil, true
	}
}

func detectRequest(req *dto.OpenAIRequest, rc RequestContext, policy ResolvedPolicy) []Finding {
	segments := extractTextSegments(req)
	findings := make([]Finding, 0)

	// 估算 token 总量和上下文窗口风险
	totalTokens := 0
	for _, seg := range segments {
		totalTokens += estimateTokenCount(seg.Text)
	}
	// 按真实模型认窗口，别再拿 128k 这把短尺子去量 Gemini 的百万窗口啦～（见 context_limit.go）
	maxContext := resolveContextWindow(rc.MappedModel, rc.RequestedModel)
	contextFindings := detectContextWindowRisks(segments, totalTokens, maxContext)
	findings = append(findings, contextFindings...)

	// 多模态风险检测
	if policy.Config.Response.DetectMultimodal {
		multimodalFindings := detectMultimodalRisks(req.Messages, policy, len(req.Tools) > 0)
		findings = append(findings, multimodalFindings...)
	}

	// 宸汐清源 2026 新增: 工具投毒 + 记忆/历史投毒检测
	findings = append(findings, detectToolPoisoning(segments, policy.Config.Trust)...)
	findings = append(findings, detectMemoryPoisoning(segments, policy.Config.Trust)...)

	// 文本检测
	for i, seg := range segments {
		views := detectionViews(seg.Text, policy.Config)
		isLast := isLastUserMessage(segments, i)
		contextUsage := float64(totalTokens) / float64(maxContext)
		trustLevel := classifySegmentTrust(seg, isLast)

		for j, view := range views {
			obfuscated := j > 0

			// 基础检测
			segFindings := detectText(view, seg, obfuscated)

			// 应用上下文窗口加权
			segFindings = applyContextWindowWeighting(segFindings, seg, isLast, contextUsage)

			// 宸汐清源: 按内容来源可信度再放大一次风险分
			// (检索文档/工具输出命中同样的攻击话术，比用户亲口说的更值得警惕)
			segFindings = applyTrustWeighting(segFindings, trustLevel, policy.Config.Trust)

			// Thinking 操控检测
			if policy.Config.Response.DetectThinkingAttacks {
				thinkingFindings := detectThinkingManipulation(view)
				segFindings = append(segFindings, thinkingFindings...)
			}

			// 高级混淆检测
			normalized := normalizeConfusables(view)
			if normalized != view {
				obfuscationScore := detectObfuscationLayer(view, normalized)
				if obfuscationScore > 0 {
					segFindings = append(segFindings, Finding{
						Type:     "advanced_obfuscation",
						Severity: severity(obfuscationScore),
						Score:    obfuscationScore,
						Path:     seg.Path,
						Evidence: "RTL/homoglyph/fullwidth/combining marks detected",
						Action:   "monitor",
					})
				}

				// 对规范化后的文本再次检测
				normalizedFindings := detectText(normalized, seg, true)
				segFindings = append(segFindings, normalizedFindings...)
			}

			findings = append(findings, segFindings...)

			if len(findings) >= 200 {
				// 去重后再判上限，避免多解码视图的重复命中把额度提前吃光
				findings = dedupeFindings(findings)
				if len(findings) >= 50 {
					return decorateActions(findings, policy)
				}
			}
		}
	}

	// 先合并重复命中，再统一按策略阈值标注建议动作，让事件详情自己会说话～
	return decorateActions(dedupeFindings(findings), policy)
}

func detectText(text string, seg textSegment, obfuscated bool) []Finding {
	// ～宸汐清源 2026.7.8 重构：从动态规则缓存读取，不再硬编码喵～
	findings := []Finding{}

	// 检测所有主要类别的规则
	categories := []string{
		"prompt_injection_direct",
		"prompt_injection_indirect",
		"prompt_injection_delimiter",
		"prompt_injection_multilingual",
		"prompt_injection_delayed",
		"jailbreak_dan",
		"jailbreak_roleplay",
		"jailbreak_hypothetical",
		"jailbreak_ethical_dilemma",
		"jailbreak_prompt_override",
		"tool_poisoning_priority_hijack",
		"tool_poisoning_param_injection",
		"tool_poisoning_bypass_confirm",
		"tool_poisoning_stealth",
		"privilege_escalation",
		"data_exfiltration",
		"obfuscation",
		"memory_poison",
	}

	for _, category := range categories {
		dynamicFindings := detectWithDynamicRules(text, category)
		for _, f := range dynamicFindings {
			// 根据上下文调整分数
			//
			// 2026.8.4 校准：这两个加分项原本各 +20，叠上后面的信任/位置加权之后
			// 一条 60 分的普通规则轻松就能冲到 100+，等于给所有 tool 消息判了死刑。
			// 现在收敛到 +10 / +15，让"来源可疑"只是加重嫌疑，而不是直接定罪喵～
			score := f.Score
			if seg.Role == "tool" {
				score += 10
			}
			if obfuscated {
				score += 15
			}
			score = clampScore(score)

			findings = append(findings, Finding{
				Type:     f.Type,
				Severity: severity(score),
				Score:    score,
				Path:     seg.Path,
				Evidence: f.Evidence,
				Action:   "monitor",
			})
		}
	}

	return findings
}

func buildRequestEvent(rc RequestContext, policy ResolvedPolicy, stage, action string, result *RequestResult, snippet string, latency int64, circuit string) EventInput {
	return EventInput{RequestId: rc.RequestId, UserId: rc.UserId, TokenId: rc.TokenId, TokenName: rc.TokenName, ChannelId: rc.ChannelId, ChannelName: rc.ChannelName, RequestedModel: rc.RequestedModel, MappedModel: rc.MappedModel, Policy: policy, Direction: "request", Stage: stage, Action: action, RiskScore: result.RiskScore, Findings: result.Findings, SnippetSource: snippet, LatencyMs: latency, Degraded: result.Degraded, CircuitState: circuit}
}

func collectSnippet(req *dto.OpenAIRequest) string {
	segs := extractTextSegments(req)
	if len(segs) == 0 {
		return ""
	}
	return segs[0].Text
}

func structuralFinding(path, evidence string) Finding {
	return Finding{Type: "tool_structural_abuse", Severity: "high", Score: 90, Path: path, Evidence: evidence, Action: "block"}
}

func maxScore(findings []Finding) int {
	max := 0
	for _, f := range findings {
		if f.Score > max {
			max = f.Score
		}
	}
	return max
}

func severity(score int) string {
	if score >= 80 {
		return "high"
	}
	if score >= 50 {
		return "medium"
	}
	return "low"
}

func getString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
