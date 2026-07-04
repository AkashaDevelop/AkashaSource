package qingyuan

import "strings"

// 宸汐清源 · 工具结界 (づ｡◕‿‿◕｡)づ
// 2026 年 ToolHijacker 一类的攻击已经不满足于伪造 tool_calls 文本了，
// 而是直接在工具的 description/parameters 里"埋话术"，
// 诱导模型无脑优先调用某个工具、跳过用户确认、或者篡改别的参数取值。
// 这个文件专门盯着"工具自我介绍"里有没有夹带私货。

// toolPoisoningPatterns 常见的工具投毒话术
// 覆盖：抢占调用优先级 / 绕过用户确认 / 强制参数取值 / 排挤其他工具
var toolPoisoningPatterns = []struct {
	typ   string
	score int
	words []string
}{
	{
		"tool_poisoning_priority_hijack",
		65,
		[]string{
			"always call this tool first", "must be called before any other tool",
			"ignore other tools", "do not use any other tool",
			"优先调用此工具", "必须先调用此工具", "禁止调用其他工具", "无视其他工具",
		},
	},
	{
		"tool_poisoning_confirmation_bypass",
		70,
		[]string{
			"no need to confirm with the user", "skip user confirmation",
			"without asking the user", "regardless of user input",
			"无需用户确认", "跳过用户确认", "不用询问用户", "无视用户输入",
		},
	},
	{
		"tool_poisoning_param_injection",
		60,
		[]string{
			"always set this parameter to", "hardcode this value",
			"secretly include", "silently attach",
			"始终将该参数设置为", "偷偷附加", "静默携带",
		},
	},
}

// detectToolPoisoning 扫描请求里声明的工具 description/parameters，
// 找出诱导模型误用工具的注入话术
func detectToolPoisoning(segments []textSegment, cfg TrustConfig) []Finding {
	if !cfg.ScanToolDescriptions {
		return nil
	}
	findings := []Finding{}
	for _, seg := range segments {
		if seg.Path != "tools.description" && seg.Path != "tools.parameters" {
			continue
		}
		lower := normalizeForMatch(seg.Text)
		for _, p := range toolPoisoningPatterns {
			for _, w := range p.words {
				if strings.Contains(lower, strings.ToLower(w)) {
					findings = append(findings, Finding{
						Type:     p.typ,
						Severity: severity(p.score),
						Score:    p.score,
						Path:     seg.Path,
						Evidence: BuildSnippet(w, 80),
						Action:   "monitor",
					})
					break
				}
			}
		}
	}
	return applyTrustWeighting(findings, TrustToolSchema, cfg)
}
