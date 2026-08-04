package qingyuan

// 宸汐清源 · 命中分级小铃铛 (｡•ᴗ•｡)
//
// 以前所有 Finding 都被一视同仁地当成"抓到坏人了"，可实际上里面混了一批
// 纯粹用来做标记的小纸条——比如 last_user_message_focus 只是在说
// "这条是最新的用户发言，等下扫描仔细点哦"，它 score=0，压根不是风险。
//
// 偏偏这些小纸条也会让 token 风险基线一路往上爬、让 AI 复审白白跑一趟，
// 于是正常聊天十几轮之后，风险基线就悄悄顶到上限，一句人畜无害的话都可能被拦下来。
//
// 所以这里给 Finding 挂个小铃铛：能响的（实质命中）才算数，
// 不响的（信息类）只留在事件详情里给管理员看，绝不参与任何"累积/升级/触发"的判断喵～

// informationalTypes 信息类 Finding 白名单
//
// 这些类型只承担"标注上下文"或"记录一次决策"的职责，本身不表示检测到攻击：
//   - last_user_message_focus：标记最后几条用户消息，提示后续扫描重点
//   - risk_annotate：风险分达到标注阈值时留的一条痕迹，风险本身已由其它 Finding 表达
//   - auto_escalation：模式升级的决策记录
//   - ai_re_review_*：AI 旁路复审的结论回执，不重复计入风险累积
var informationalTypes = map[string]bool{
	"last_user_message_focus": true,
	"risk_annotate":           true,
	"auto_escalation":         true,
	"ai_re_review":            true,
}

// IsInformational 判断一条 Finding 是不是"不会响的小铃铛"
//
// 两种情况算信息类：
//  1. 类型在白名单里（哪怕分数非零，也只是标注用途）
//  2. 分数 <= 0（既然一分风险都不值，就没有资格推高任何累积状态）
func IsInformational(f Finding) bool {
	if informationalTypes[f.Type] {
		return true
	}
	return f.Score <= 0
}

// HasSubstantive 这批 Finding 里有没有真正响起来的铃铛
//
// 所有"要不要累积风险 / 要不要叫 AI 复审 / 要不要写一条告警事件"的判断，
// 都该问它，而不是直接看 len(findings) > 0 喵～
func HasSubstantive(findings []Finding) bool {
	for _, f := range findings {
		if !IsInformational(f) {
			return true
		}
	}
	return false
}

// SubstantiveFindings 只挑出实质命中的那些，用于喂给 AI 复审
// （把一堆 score=0 的小纸条送去给 AI 看，既费钱又干扰判断）
func SubstantiveFindings(findings []Finding) []Finding {
	out := make([]Finding, 0, len(findings))
	for _, f := range findings {
		if !IsInformational(f) {
			out = append(out, f)
		}
	}
	return out
}

// decorateActions 按策略阈值给每条实质命中补上"建议动作"
//
// 之前所有规则命中都硬编码成 "monitor"，管理员在事件列表里根本看不出
// 哪条是"真的把请求拦下来了"、哪条只是"记一笔"，排查起来很痛苦。
// 这里按同一套阈值统一标注，让事件详情自己会说话～
func decorateActions(findings []Finding, policy ResolvedPolicy) []Finding {
	blockAt := policy.Config.Risk.BlockThreshold
	annotateAt := policy.Config.Risk.AnnotateThreshold
	for i := range findings {
		if IsInformational(findings[i]) {
			continue
		}
		// 结构性滥用已经在产生时就明确标了 block，不要覆盖掉
		if findings[i].Action == "block" {
			continue
		}
		switch {
		case blockAt > 0 && findings[i].Score >= blockAt:
			findings[i].Action = "block"
		case annotateAt > 0 && findings[i].Score >= annotateAt:
			findings[i].Action = "annotate"
		default:
			findings[i].Action = "monitor"
		}
	}
	return findings
}

// clampScore 把风险分锁在 0~100 之间
//
// 加权链上有好几个放大系数，不封顶的话会算出 300+ 这种没有意义的数字，
// 既让 severity 判定失真，也让管理员没法横向比较不同事件的严重程度喵～
func clampScore(score int) int {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

// dedupeFindings 同一处的同类命中只留分数最高的那条
//
// 一段文本会被拆成好几个"检测视图"（原文 / 去零宽 / URL 解码 / HTML 解码 /
// JSON 反转义 / Base64 解码…），每个视图都要跑一遍全部规则。
// 于是一句普通的话可能在事件详情里留下七八条一模一样的记录，
// 既撑爆了 50 条的上限（把后面真正重要的命中挤掉），也让管理员看得眼花 (´･ω･`)
func dedupeFindings(findings []Finding) []Finding {
	if len(findings) <= 1 {
		return findings
	}
	type key struct{ typ, path string }
	seen := make(map[key]int, len(findings))
	out := make([]Finding, 0, len(findings))
	for _, f := range findings {
		k := key{f.Type, f.Path}
		if idx, ok := seen[k]; ok {
			if f.Score > out[idx].Score {
				out[idx] = f
			}
			continue
		}
		seen[k] = len(out)
		out = append(out, f)
	}
	return out
}
