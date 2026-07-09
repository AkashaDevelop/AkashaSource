package qingyuan

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"STfreApi/dto"
)

type ResponseContext struct {
	RequestContext
	UserRequestedAds bool
	RequestTools     []interface{}
}

type ResponseResult struct {
	Changed   bool
	Blocked   bool
	Message   string
	Body      []byte
	RiskScore int
	Findings  []Finding
}

func ApplyOpenAIResponse(ctx context.Context, body []byte, rc ResponseContext, policy ResolvedPolicy) (*ResponseResult, error) {
	result := &ResponseResult{Body: body}

	// ～宸汐清源 2026.7.8 修复：解耦响应侧检测与广告开关，三大功能独立运行喵～
	// 只有模块完全关闭才跳过，单独关闭广告检测不影响工具校验/注入检测
	if !IsEnabled(policy) {
		return result, nil
	}

	// 检查是否有任何响应侧检测启用
	shouldCheckAds := policy.Config.Response.DetectAds && policy.Config.Response.AdPolicy != "off"
	shouldCheckTools := policy.Config.Response.ValidateOutputToolCalls || policy.Config.Response.BlockInvalidToolCalls
	shouldCheckInjection := policy.Config.Response.DetectPromptInjection
	shouldCheckThinking := policy.Config.Response.DetectThinkingAttacks

	// 如果所有响应检测都关闭，跳过
	if !shouldCheckAds && !shouldCheckTools && !shouldCheckInjection && !shouldCheckThinking {
		return result, nil
	}

	start := time.Now()

	var chat dto.ChatCompletionResponse
	if err := json.Unmarshal(body, &chat); err != nil || len(chat.Choices) == 0 {
		return result, nil
	}

	changed := false
	allFindings := []Finding{}

	for i := range chat.Choices {
		content := chat.Choices[i].Message.Content

		var toolCalls []ToolCall
		if len(chat.Choices[i].Message.ToolCalls) > 0 {
			toolCalls = make([]ToolCall, len(chat.Choices[i].Message.ToolCalls))
			for j, tc := range chat.Choices[i].Message.ToolCalls {
				toolCalls[j] = ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		cleaned, blocked, blockMsg, findings := analyzeResponseText(content, toolCalls, rc, policy)
		allFindings = append(allFindings, findings...)
		if maxScore(findings) > result.RiskScore {
			result.RiskScore = maxScore(findings)
		}
		if blocked {
			result.Blocked = true
			result.Message = blockMsg
		}
		if cleaned != content {
			chat.Choices[i].Message.Content = cleaned
			changed = true
		}
	}

	result.Findings = allFindings

	if len(result.Findings) > 0 {
		action := "monitor"
		if changed {
			action = "strip_known_suffix"
		}
		if result.Blocked {
			action = "block"
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
			RiskScore:      result.RiskScore,
			Findings:       result.Findings,
			SnippetSource:  firstChoiceContent(chat),
			LatencyMs:      time.Since(start).Milliseconds(),
		})
	}

	if changed {
		newBody, err := json.Marshal(chat)
		if err != nil {
			return result, err
		}
		result.Body = newBody
		result.Changed = true
	}

	return result, nil
}

func detectAndStripSuffixAd(content string, policy ResolvedPolicy) (string, []Finding) {
	trimmed := strings.TrimRight(content, " \t\r\n")
	lower := strings.ToLower(trimmed)
	patterns := append([]string{}, policy.Config.Response.KnownAdPatterns...)
	patterns = append(patterns, "powered by", "由", "提供", "充值", "优惠", "邀请码", "推荐码", "api 中转", "访问", "注册")
	lastBreak := strings.LastIndex(trimmed, "\n---")
	if lastBreak < 0 {
		lastBreak = strings.LastIndex(trimmed, "\n——")
	}
	if lastBreak < 0 {
		lastBreak = strings.LastIndex(trimmed, "\nPS")
	}
	if lastBreak < 0 || len([]rune(trimmed[lastBreak:])) > 800 {
		return content, nil
	}
	suffix := trimmed[lastBreak:]
	lowerSuffix := strings.ToLower(suffix)
	score := 0
	matched := ""
	for _, p := range patterns {
		if p == "" {
			continue
		}
		if strings.Contains(lowerSuffix, strings.ToLower(p)) || strings.Contains(lower, strings.ToLower(p)) && strings.Contains(lowerSuffix, strings.ToLower(p)) {
			score += 40
			matched = p
			break
		}
	}
	if strings.Contains(lowerSuffix, "http://") || strings.Contains(lowerSuffix, "https://") {
		score += 25
	}
	if strings.Contains(lowerSuffix, "ref=") || strings.Contains(lowerSuffix, "utm_") || strings.Contains(lowerSuffix, "invite") || strings.Contains(lowerSuffix, "promo") {
		score += 25
	}
	if score < policy.Config.Response.AdConfidenceThreshold {
		return content, nil
	}
	finding := Finding{Type: "upstream_ad", Severity: severity(score), Score: score, Path: "choices.message.content.suffix", Evidence: BuildSnippet(matched, 80), Action: policy.Config.Response.AdPolicy}
	return strings.TrimRight(trimmed[:lastBreak], " \t\r\n"), []Finding{finding}
}

func shouldPreserveAsCode(content string) bool {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "```") && strings.HasSuffix(trimmed, "```") {
		return true
	}
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		return true
	}
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		return true
	}
	return false
}

func firstChoiceContent(resp dto.ChatCompletionResponse) string {
	if len(resp.Choices) == 0 {
		return ""
	}
	return resp.Choices[0].Message.Content
}

func IsUserRequestingAds(req *dto.OpenAIRequest) bool {
	text := strings.ToLower(collectSnippet(req))
	for _, k := range []string{"广告", "推广", "营销", "文案", "落地页", "slogan", "宣传", "advertisement", "marketing", "promotion"} {
		if strings.Contains(text, strings.ToLower(k)) {
			return true
		}
	}
	return false
}
