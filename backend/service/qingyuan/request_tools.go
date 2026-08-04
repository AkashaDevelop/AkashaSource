package qingyuan

import (
	"encoding/json"
	"regexp"

	"STfreApi/dto"
)

// 宸汐清源 · 械律结界 (｡•ω•｡)
//
// 这里管的是"请求里的工具声明和消息结构合不合规矩"——
// 工具名有没有乱起、tool_choice 指没指向一个根本不存在的工具、
// 谁在冒充 assistant 往历史里塞 tool_calls……
//
// 和内容检测不同，这一层抓的是**结构性伪造**：不需要读懂意思，
// 光看形状不对就能确定有人在动手脚，所以命中即高分、直接拦。

func validateToolsAndMessages(req *dto.OpenAIRequest, policy ResolvedPolicy) ([]Finding, bool) {
	if !policy.Config.Tools.Enabled {
		return nil, false
	}
	findings := []Finding{}
	blocked := false
	toolNames := map[string]bool{}

	// ～2026.8.4 修正：以前这里是 `nameRe, _ := regexp.Compile(...)`，
	// 管理员在策略里填了一个写错的正则，编译失败 → nameRe 为 nil →
	// 下面的短路条件让工具名校验被整个跳过，防护静默消失且毫无提示 (；￣Д￣)
	// 现在编译失败就退回内置正则，并留一条 Finding 让管理员知道配置有问题～
	nameRe, reErr := regexp.Compile(policy.Config.Tools.ToolNameRegex)
	if reErr != nil {
		nameRe = regexp.MustCompile(defaultToolNameRegex)
		findings = append(findings, Finding{
			Type:     "policy_config_invalid",
			Severity: "medium",
			Score:    0, // 这是配置问题不是攻击，不参与风险累积
			Path:     "config.tools.tool_name_regex",
			Evidence: "工具名正则编译失败，已回退内置默认值: " + reErr.Error(),
			Action:   "annotate",
		})
	}

	// ～工具白/黑名单：空切片视为不限制，完全向后兼容～
	allowedSet := toolNameSet(policy.Config.Tools.AllowedToolNames)
	blockedSet := toolNameSet(policy.Config.Tools.BlockedToolNames)
	if len(req.Tools) > policy.Config.Tools.MaxTools {
		findings = append(findings, structuralFinding("tools", "工具数量超过限制"))
		blocked = true
	}
	for _, t := range req.Tools {
		tm, ok := t.(map[string]any)
		if !ok {
			continue
		}
		fn, ok := tm["function"].(map[string]any)
		if !ok {
			continue
		}
		name := getString(fn["name"])
		if name == "" || !nameRe.MatchString(name) {
			findings = append(findings, structuralFinding("tools.name", "工具名称无效"))
			blocked = true
		}
		if len(blockedSet) > 0 && blockedSet[name] {
			findings = append(findings, structuralFinding("tools.name", "工具名称命中黑名单"))
			blocked = true
		}
		if len(allowedSet) > 0 && !allowedSet[name] {
			findings = append(findings, structuralFinding("tools.name", "工具名称不在白名单内"))
			blocked = true
		}
		if toolNames[name] {
			findings = append(findings, structuralFinding("tools.name", "工具名称重复"))
			blocked = true
		}
		toolNames[name] = true
		if b, err := json.Marshal(fn["parameters"]); err == nil && len(b) > policy.Config.Tools.MaxToolSchemaBytes {
			findings = append(findings, structuralFinding("tools.parameters", "工具 schema 过大"))
			blocked = true
		}
	}
	if policy.Config.Tools.ValidateToolChoice {
		findings, blocked = appendToolChoiceFindings(req.ToolChoice, toolNames, findings, blocked)
	}
	if policy.Config.Tools.ValidateAssistantToolCalls {
		f2, b2 := validateMessageToolHistory(req.Messages)
		findings = append(findings, f2...)
		blocked = blocked || b2
	}
	return findings, blocked && policy.Config.Risk.BlockStructuralToolAbuse
}

func appendToolChoiceFindings(choice any, toolNames map[string]bool, findings []Finding, blocked bool) ([]Finding, bool) {
	if choice == nil {
		return findings, blocked
	}
	if s, ok := choice.(string); ok {
		if s == "auto" || s == "none" || s == "required" {
			return findings, blocked
		}
		return append(findings, structuralFinding("tool_choice", "tool_choice 字符串无效")), true
	}
	cm, ok := choice.(map[string]any)
	if !ok {
		return findings, blocked
	}
	fn, _ := cm["function"].(map[string]any)
	name := getString(fn["name"])
	if name != "" && !toolNames[name] {
		return append(findings, structuralFinding("tool_choice", "tool_choice 指向未声明工具")), true
	}
	return findings, blocked
}

func validateMessageToolHistory(messages []interface{}) ([]Finding, bool) {
	findings := []Finding{}
	blocked := false
	validCalls := map[string]bool{}
	for _, m := range messages {
		mm, ok := m.(map[string]any)
		if !ok {
			continue
		}
		role := getString(mm["role"])
		if _, exists := mm["tool_calls"]; exists && role != "assistant" {
			findings = append(findings, structuralFinding("messages.tool_calls", "非 assistant 消息包含真实 tool_calls 字段"))
			blocked = true
		}
		if role == "assistant" {
			if calls, ok := mm["tool_calls"].([]any); ok {
				for _, c := range calls {
					if cm, ok := c.(map[string]any); ok {
						if id := getString(cm["id"]); id != "" {
							validCalls[id] = true
						}
					}
				}
			}
		}
		if role == "tool" {
			id := getString(mm["tool_call_id"])
			if id == "" || !validCalls[id] {
				findings = append(findings, structuralFinding("messages.tool_call_id", "tool 消息缺少合法 tool_call_id"))
				blocked = true
			}
		}
	}
	return findings, blocked
}
func toolNameSet(names []string) map[string]bool {
	if len(names) == 0 {
		return nil
	}
	set := make(map[string]bool, len(names))
	for _, n := range names {
		if n != "" {
			set[n] = true
		}
	}
	return set
}
