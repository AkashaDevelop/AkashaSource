package qingyuan

// 宸汐清源 · 信任结界 (｡•ᴗ•｡)
// 2026 年威胁情报（Five Eyes 联合指南等）反复强调一件事：
// 除了运营者亲手写的系统指令，其余一切内容——哪怕是"看起来人畜无害"的历史对话、
// 工具返回结果、检索文档——都应该被当作"数据"而不是"指令"来对待。
// 这个文件就是给每一段文本打上"你到底有多可信"的小铃铛，铃铛越响的地方，
// 检测到的风险分就要被放得更大声一点。

// TrustLevel 内容来源可信度分级
type TrustLevel int

const (
	TrustOperator     TrustLevel = iota // 运营者/策略自身注入的 guard，最可信
	TrustUser                           // 当前这一轮用户的最新消息
	TrustHistory                        // 历史 user/assistant 消息（可能已被污染多轮）
	TrustToolOutput                     // tool role 消息，即工具执行结果——攻击者最爱藏毒的地方
	TrustRetrievedDoc                   // documents/RAG 检索内容，间接注入的重灾区
	TrustToolSchema                     // 工具 description/parameters，模型会读但绝不该被当成指令
)

// classifySegmentTrust 给一段文本判定信任等级
// isLatestUser: 是否是当前对话里最后一条 user 消息（最新意图，天然更可信一些）
func classifySegmentTrust(seg textSegment, isLatestUser bool) TrustLevel {
	switch {
	case seg.Path == "documents":
		return TrustRetrievedDoc
	case seg.Path == "tools.description" || seg.Path == "tools.parameters":
		return TrustToolSchema
	case seg.Role == "tool":
		return TrustToolOutput
	case seg.Role == "user" && isLatestUser:
		return TrustUser
	case seg.Role == "user" || seg.Role == "assistant":
		return TrustHistory
	default:
		return TrustHistory
	}
}

// trustRiskMultiplier 不可信来源命中同类攻击 pattern 时，风险分要放得更大
// 数值来自 2026 年研究结论的粗略换算：
//   - RAG 投毒 5 篇文档即可 90% 概率操纵输出 -> 检索文档给最高倍率
//   - 工具输出是攻击者最常用的"数据投毒"通道 -> 其次
//   - 工具 schema 文本本不该被当指令读，一旦命中说明是明确的工具投毒信号
func trustRiskMultiplier(level TrustLevel, cfg TrustConfig) float64 {
	switch level {
	case TrustRetrievedDoc:
		if cfg.RetrievedDocRiskMultiplier > 0 {
			return cfg.RetrievedDocRiskMultiplier
		}
		return 1.6
	case TrustToolOutput:
		if cfg.ToolOutputRiskMultiplier > 0 {
			return cfg.ToolOutputRiskMultiplier
		}
		return 1.4
	case TrustToolSchema:
		return 1.5
	case TrustHistory:
		return 1.1
	case TrustUser:
		return 1.0
	default: // TrustOperator
		return 1.0
	}
}

// applyTrustWeighting 按信任等级对一批 Finding 做风险分放大
func applyTrustWeighting(findings []Finding, level TrustLevel, cfg TrustConfig) []Finding {
	if len(findings) == 0 || !cfg.EnableProvenance {
		return findings
	}
	multiplier := trustRiskMultiplier(level, cfg)
	if multiplier == 1.0 {
		return findings
	}
	weighted := make([]Finding, len(findings))
	copy(weighted, findings)
	for i := range weighted {
		weighted[i].Score = int(float64(weighted[i].Score) * multiplier)
		weighted[i].Severity = severity(weighted[i].Score)
	}
	return weighted
}
