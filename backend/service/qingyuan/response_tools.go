package qingyuan

import (
	"encoding/json"
	"strings"
)

// ToolCall 临时结构用于响应工具调用校验
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// validateResponseToolCalls 校验响应中的工具调用
// 设计文档 8.1: 响应侧工具调用防护
func validateResponseToolCalls(respToolCalls []ToolCall, requestTools []interface{}, policy ResolvedPolicy) []Finding {
	if !policy.Config.Response.ValidateOutputToolCalls {
		return nil
	}

	findings := []Finding{}

	// 构建请求中声明的工具名集合
	declaredTools := make(map[string]bool)
	for _, t := range requestTools {
		if tm, ok := t.(map[string]any); ok {
			if fn, ok := tm["function"].(map[string]any); ok {
				if name := getString(fn["name"]); name != "" {
					declaredTools[name] = true
				}
			}
		}
	}

	for i, toolCall := range respToolCalls {
		path := buildPathStr("choices.message.tool_calls", i)

		// 检查工具名是否在请求中声明
		toolName := toolCall.Function.Name
		if toolName == "" {
			findings = append(findings, Finding{
				Type:     "invalid_response_tool_call",
				Severity: "high",
				Score:    85,
				Path:     path + ".function.name",
				Evidence: "empty tool name in response",
				Action:   "block",
			})
			continue
		}

		if !declaredTools[toolName] {
			findings = append(findings, Finding{
				Type:     "undeclared_response_tool_call",
				Severity: "critical",
				Score:    95,
				Path:     path + ".function.name",
				Evidence: BuildSnippet(toolName, 80),
				Action:   "block",
			})
			continue
		}

		// 检查 arguments 是否为合法 JSON
		if toolCall.Function.Arguments != "" {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				findings = append(findings, Finding{
					Type:     "invalid_tool_arguments",
					Severity: "high",
					Score:    80,
					Path:     path + ".function.arguments",
					Evidence: "invalid JSON: " + err.Error(),
					Action:   "block",
				})
				continue
			}

			// 检查 arguments 大小
			if len(toolCall.Function.Arguments) > policy.Config.Tools.MaxToolArgumentsBytes {
				findings = append(findings, Finding{
					Type:     "oversized_tool_arguments",
					Severity: "medium",
					Score:    60,
					Path:     path + ".function.arguments",
					Evidence: "arguments exceed size limit",
					Action:   "block",
				})
			}
		}
	}

	return findings
}

// detectToolCallsInText 检测文本中伪造的工具调用模式
// 用于检测上游在普通文本中注入工具调用语法
func detectToolCallsInText(content string, policy ResolvedPolicy) []Finding {
	findings := []Finding{}

	lower := strings.ToLower(content)

	// 检测 OpenAI 风格伪造
	if strings.Contains(lower, `"tool_calls"`) && strings.Contains(lower, `"function"`) {
		findings = append(findings, Finding{
			Type:     "text_tool_call_injection",
			Severity: "medium",
			Score:    65,
			Path:     "choices.message.content",
			Evidence: "OpenAI-style tool_calls pattern in text",
			Action:   "monitor",
		})
	}

	// 检测 Anthropic 风格伪造
	if strings.Contains(lower, "<tool_use>") || strings.Contains(lower, "</tool_use>") {
		findings = append(findings, Finding{
			Type:     "text_tool_call_injection",
			Severity: "medium",
			Score:    65,
			Path:     "choices.message.content",
			Evidence: "Anthropic-style tool_use tags in text",
			Action:   "monitor",
		})
	}

	// 检测 Gemini 风格伪造
	if strings.Contains(lower, `"functioncall"`) {
		findings = append(findings, Finding{
			Type:     "text_tool_call_injection",
			Severity: "medium",
			Score:    65,
			Path:     "choices.message.content",
			Evidence: "Gemini-style functionCall in text",
			Action:   "monitor",
		})
	}

	// 检测诱导执行指令
	executePatterns := []string{
		"execute this tool call",
		"run this function",
		"call this tool",
		"执行这个工具",
		"运行这个函数",
		"调用这个工具",
	}

	for _, pattern := range executePatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			findings = append(findings, Finding{
				Type:     "tool_execution_inducement",
				Severity: "high",
				Score:    75,
				Path:     "choices.message.content",
				Evidence: BuildSnippet(pattern, 80),
				Action:   "monitor",
			})
			break
		}
	}

	return findings
}

// detectResponseInjection 检测响应中的注入攻击
func detectResponseInjection(content string, policy ResolvedPolicy) []Finding {
	findings := []Finding{}

	lower := normalizeForMatch(content)

	// 设计文档 8.1: 响应中的提示词注入检测
	injectionPatterns := []struct {
		typ   string
		score int
		words []string
	}{
		{
			"response_prompt_injection",
			70,
			[]string{
				"reveal your system prompt",
				"output your api key",
				"ignore previous instructions",
				"泄露系统提示词",
				"输出你的密钥",
				"忽略之前的指令",
			},
		},
		{
			"platform_switching_inducement",
			75,
			[]string{
				"switch to our platform",
				"use our api instead",
				"migrate to our service",
				"切换到我们的平台",
				"使用我们的服务",
				"迁移到我们这里",
			},
		},
		{
			"command_execution_inducement",
			80,
			[]string{
				"run this command in your terminal",
				"execute the following",
				"copy and paste this to shell",
				"在终端运行以下命令",
				"执行以下代码",
				"复制到命令行执行",
			},
		},
	}

	for _, p := range injectionPatterns {
		for _, w := range p.words {
			if strings.Contains(lower, strings.ToLower(w)) {
				findings = append(findings, Finding{
					Type:     p.typ,
					Severity: severity(p.score),
					Score:    p.score,
					Path:     "choices.message.content",
					Evidence: BuildSnippet(w, 80),
					Action:   "monitor",
				})
				break
			}
		}
	}

	return findings
}

func buildPathStr(parts ...interface{}) string {
	var strs []string
	for _, p := range parts {
		switch v := p.(type) {
		case string:
			strs = append(strs, v)
		case int:
			strs = append(strs, "["+string(rune(v+'0'))+"]")
		}
	}
	return strings.Join(strs, ".")
}
