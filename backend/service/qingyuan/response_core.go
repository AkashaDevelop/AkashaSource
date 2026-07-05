package qingyuan

// analyzeResponseText 格式无关的响应文本检测核心：
// 广告清理 + 工具调用校验 + 文本内工具调用注入检测 + 响应注入检测 + thinking 校验，
// Claude/Responses/Gemini 各自的响应处理器只需要把"提取出来的文本 + 工具调用列表"喂进来，
// 序列化成各自协议格式、怎么写回客户端，都交给调用方自己决定～
func analyzeResponseText(content string, toolCalls []ToolCall, rc ResponseContext, policy ResolvedPolicy) (cleanedContent string, blocked bool, blockMsg string, findings []Finding) {
	cleanedContent = content
	if content == "" || shouldPreserveAsCode(content) || rc.UserRequestedAds {
		return content, false, "", nil
	}

	// 广告检测与清理
	cleaned, adFindings := detectAndStripSuffixAd(content, policy)
	findings = append(findings, adFindings...)

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
		findings = append(findings, detectResponseInjection(content, policy)...)
	}

	// Thinking 内容校验
	if policy.Config.Response.DetectThinkingAttacks {
		findings = append(findings, validateThinkingResponse(content, policy)...)
	}

	if cleaned != content && policy.Config.Response.AdPolicy == "strip_known_suffix" {
		cleanedContent = cleaned
	}

	return cleanedContent, blocked, blockMsg, findings
}
