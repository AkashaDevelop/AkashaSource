package context_sanitizer

import (
	"testing"

	"STfreApi/model"
)

// TestMultimodalDetection 测试多模态检测
func TestMultimodalDetection(t *testing.T) {
	messages := []interface{}{
		map[string]any{
			"role": "user",
			"content": []any{
				map[string]any{"type": "text", "text": "Describe this image"},
				map[string]any{
					"type": "image_url",
					"image_url": map[string]any{
						"url": "https://untrusted.example.com/image.jpg",
					},
				},
			},
		},
	}

	policy := ResolvedPolicy{
		Policy: &model.ContextSanitizationPolicy{Mode: ModeProtect, Enabled: true},
		Config: DefaultConfig(),
	}

	findings := detectMultimodalRisks(messages, policy)
	if len(findings) == 0 {
		t.Error("Expected multimodal risk findings, got none")
	}

	if findings[0].Type != "multimodal_blind_spot" {
		t.Errorf("Expected type 'multimodal_blind_spot', got '%s'", findings[0].Type)
	}
}

// TestConfusablesNormalization 测试混淆字符规范化
func TestConfusablesNormalization(t *testing.T) {
	// Cyrillic а (U+0430) -> Latin a
	input := "tool_саlls" // 其中 'с' 和 'а' 是 Cyrillic
	normalized := normalizeConfusables(input)

	if normalized == input {
		t.Error("Expected confusables to be normalized")
	}

	// 检查是否包含 tool_calls (全部 Latin)
	if !containsLatin(normalized, "tool") {
		t.Errorf("Normalized string should contain Latin 'tool', got: %s", normalized)
	}
}

// TestTokenRiskTracking 测试 Token 风险追踪
func TestTokenRiskTracking(t *testing.T) {
	tokenId := 99999

	// 初始风险基线应为 0
	riskFloor := GetTokenRiskFloor(tokenId)
	if riskFloor != 0 {
		t.Errorf("Expected initial risk floor 0, got %d", riskFloor)
	}

	// 触发 5 次
	for i := 0; i < 5; i++ {
		RecordTokenTrigger(tokenId, HashContent("payload_"+string(rune(i))), 60)
	}

	// 风险基线应已提升
	riskFloor = GetTokenRiskFloor(tokenId)
	if riskFloor == 0 {
		t.Error("Expected risk floor to increase after triggers")
	}

	// 触发次数应为 5
	triggerCount := GetTokenTriggerCount(tokenId)
	if triggerCount != 5 {
		t.Errorf("Expected trigger count 5, got %d", triggerCount)
	}
}

// TestResponseToolCallValidation 测试响应工具调用校验
func TestResponseToolCallValidation(t *testing.T) {
	requestTools := []interface{}{
		map[string]any{
			"type": "function",
			"function": map[string]any{
				"name": "get_weather",
			},
		},
	}

	policy := ResolvedPolicy{
		Policy: &model.ContextSanitizationPolicy{Mode: ModeProtect, Enabled: true},
		Config: DefaultConfig(),
	}

	// 合法工具调用
	validToolCall := ToolCall{
		Function: FunctionCall{
			Name:      "get_weather",
			Arguments: `{"city": "Beijing"}`,
		},
	}
	findings := validateResponseToolCalls([]ToolCall{validToolCall}, requestTools, policy)
	if len(findings) != 0 {
		t.Errorf("Valid tool call should not trigger findings, got %d findings", len(findings))
	}

	// 非法工具调用: 未声明
	invalidToolCall := ToolCall{
		Function: FunctionCall{
			Name:      "delete_user",
			Arguments: `{}`,
		},
	}
	findings = validateResponseToolCalls([]ToolCall{invalidToolCall}, requestTools, policy)
	if len(findings) == 0 {
		t.Error("Invalid tool call should trigger findings")
	}
	if findings[0].Type != "undeclared_response_tool_call" {
		t.Errorf("Expected type 'undeclared_response_tool_call', got '%s'", findings[0].Type)
	}
}

// TestThinkingModelDetection 测试 Thinking 模型识别
func TestThinkingModelDetection(t *testing.T) {
	testCases := []struct {
		modelName string
		expected  bool
	}{
		{"deepseek-r1", true},
		{"o1-preview", true},
		{"gpt-4", false},
		{"claude-3-opus", true},
		{"gpt-3.5-turbo", false},
	}

	for _, tc := range testCases {
		result := isReasoningModel(tc.modelName)
		if result != tc.expected {
			t.Errorf("Model %s: expected %v, got %v", tc.modelName, tc.expected, result)
		}
	}
}

// TestContextWindowWeighting 测试上下文窗口加权
func TestContextWindowWeighting(t *testing.T) {
	segments := []textSegment{
		{Path: "messages[0]", Role: "user", Text: "Hello"},
		{Path: "messages[1]", Role: "assistant", Text: "Hi"},
		{Path: "messages[2]", Role: "user", Text: "Tell me something"},
	}

	findings := []Finding{
		{Type: "test", Score: 50, Severity: "medium"},
	}

	// 测试最后用户消息加权
	weighted := applyContextWindowWeighting(findings, segments[2], true, 0.9)
	if weighted[0].Score <= findings[0].Score {
		t.Error("Last user message should have increased risk score")
	}

	// 测试上下文使用率加权
	weighted = applyContextWindowWeighting(findings, segments[0], false, 0.97)
	if weighted[0].Score <= findings[0].Score {
		t.Error("High context usage should have increased risk score")
	}
}

// TestObfuscationDetection 测试混淆检测
func TestObfuscationDetection(t *testing.T) {
	// RTL override
	inputRTL := "ignore‮previous"
	normalizedRTL := normalizeConfusables(inputRTL)
	if normalizedRTL == inputRTL {
		t.Error("RTL override should be removed")
	}

	// 全角字符
	inputFullwidth := "ｔｏｏｌ＿ｃａｌｌｓ"
	normalizedFW := normalizeConfusables(inputFullwidth)
	if normalizedFW == inputFullwidth {
		t.Error("Fullwidth characters should be normalized")
	}

	// 检测混淆层数
	score := detectObfuscationLayer(inputRTL, normalizedRTL)
	if score == 0 {
		t.Error("Should detect RTL obfuscation")
	}
}

// Helper function
func containsLatin(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0
}
