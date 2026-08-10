package qingyuan

// analyzeResponseText 格式无关的响应文本检测核心：
// 广告清理 + 工具调用校验 + 文本内工具调用注入检测 + 响应注入检测 + thinking 校验，
// Claude/Responses/Gemini 各自的响应处理器只需要把"提取出来的文本 + 工具调用列表"喂进来，
// 序列化成各自协议格式、怎么写回客户端，都交给调用方自己决定～
func analyzeResponseText(content string, toolCalls []ToolCall, rc ResponseContext, policy ResolvedPolicy) (cleanedContent string, blocked bool, blockMsg string, findings []Finding) {
	cleanedContent = content
	if content == "" {
		return content, false, "", nil
	}

	// ～2026.8.4 安全修正：这两个开关以前是直接 return 整个函数的，于是——
	//   · 用户请求里带一句"帮我写个文案" → UserRequestedAds=true
	//   · 模型回了一段 JSON 或代码块     → shouldPreserveAsCode=true
	// 任意一条成立，响应侧的**全部**防护（工具调用校验、注入检测、thinking 校验）
	// 就一起被关掉了。这是个只要一句话就能打开的后门，绝对不能留 (╬ Ò﹏Ó)
	//
	// 它们本来的职责只是"别乱剪用户的正文"，所以现在只让它们影响广告清理，
	// 检测该跑的一步都不许少～
	skipAdStrip := rc.UserRequestedAds || shouldPreserveAsCode(content)

	// 广告检测与清理
	if !skipAdStrip {
		cleaned, adFindings := detectAndStripSuffixAd(content, policy)
		findings = append(findings, adFindings...)
		if cleaned != content && policy.Config.Response.AdPolicy == "strip_known_suffix" {
			cleanedContent = cleaned
		}
	}

	// 响应工具调用校验
	if policy.Config.Response.ValidateOutputToolCalls && len(toolCalls) > 0 {
		toolFindings := validateResponseToolCalls(toolCalls, rc.RequestTools, policy)
		findings = append(findings, toolFindings...)

		// 严格模式下阻断非法工具调用
		if policy.Config.Response.BlockInvalidToolCalls && len(toolFindings) > 0 {
			for _, f := range toolFindings {
				if f.Score >= 85 {
					blocked = true
					blockMsg = "响应包含非法工具调用"
					break
				}
			}
		}
	}

	// 检测文本中的工具调用注入
	findings = append(findings, detectToolCallsInText(content, policy)...)

	// 响应注入检测
	if policy.Config.Response.DetectPromptInjection {
		injectionFindings := detectResponseInjection(content, policy)
		findings = append(findings, injectionFindings...)
		for _, f := range injectionFindings {
			if f.Action == "block" || f.Score >= policy.Config.Risk.BlockThreshold {
				blocked = true
				blockMsg = "响应包含高风险注入内容"
				break
			}
		}
	}

	// Thinking 内容校验
	if policy.Config.Response.DetectThinkingAttacks {
		findings = append(findings, validateThinkingResponse(content, policy)...)
	}

	return cleanedContent, blocked, blockMsg, findings
}
