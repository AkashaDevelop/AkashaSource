package qingyuan

// AnalyzeText 是 analyzeResponseText 的导出包装，供 controller 层处理
// Claude/Responses API 这些"纯文本 + 工具调用列表"就能表达的响应格式直接调用，
// 不需要关心具体协议的 JSON 形状～
func AnalyzeText(content string, toolCalls []ToolCall, rc ResponseContext, policy ResolvedPolicy) (cleaned string, blocked bool, blockMsg string, findings []Finding) {
	return analyzeResponseText(content, toolCalls, rc, policy)
}

// RecordResponseFindings 统一记录响应侧检测事件，Claude/Responses/Gemini 这些非 OpenAI JSON 形状的
// 响应处理器复用同一份事件落库逻辑，不用各自再拼一遍 EventInput～
func RecordResponseFindings(rc ResponseContext, policy ResolvedPolicy, findings []Finding, changed bool) {
	if len(findings) == 0 {
		return
	}
	action := "monitor"
	if changed {
		action = "strip_known_suffix"
	}
	for _, f := range findings {
		if f.Action == "block" {
			action = "block"
			break
		}
	}
	RecordEventAsync(EventInput{
		RequestId:      rc.RequestId,
		UserId:         rc.UserId,
		TokenId:        rc.TokenId,
		TokenName:      rc.TokenName,
		ChannelId:      rc.ChannelId,
		ChannelName:    rc.ChannelName,
		RequestedModel: rc.RequestedModel,
		MappedModel:    rc.MappedModel,
		Policy:         policy,
		Direction:      "response",
		Stage:          "response_validation",
		Action:         action,
		RiskScore:      maxScore(findings),
		Findings:       findings,
	})
}
