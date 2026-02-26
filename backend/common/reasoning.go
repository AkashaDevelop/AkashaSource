package common

import "strings"

// ReasoningParams holds parsed reasoning suffix parameters
type ReasoningParams struct {
	OriginalModel    string
	CleanModel       string
	ReasoningEffort  string // "high", "medium", "low", or ""
	ThinkingMode     bool
}

// ParseReasoningSuffix parses model name suffixes like -high, -medium, -low, -thinking
// and returns the clean model name plus reasoning parameters.
// Example: "claude-3-5-sonnet-latest-thinking" -> model="claude-3-5-sonnet-latest", ThinkingMode=true
func ParseReasoningSuffix(model string) ReasoningParams {
	params := ReasoningParams{OriginalModel: model, CleanModel: model}

	if strings.HasSuffix(model, "-thinking") {
		params.CleanModel = strings.TrimSuffix(model, "-thinking")
		params.ThinkingMode = true
		return params
	}

	for _, effort := range []string{"-high", "-medium", "-low"} {
		if strings.HasSuffix(model, effort) {
			params.CleanModel = strings.TrimSuffix(model, effort)
			params.ReasoningEffort = strings.TrimPrefix(effort, "-")
			return params
		}
	}

	return params
}
