package common

import "github.com/pkoukk/tiktoken-go"

// CountToken 计算 Token 数量（使用 cl100k_base 编码器，向后兼容）
func CountToken(text string) int {
	return CountTokenForModel(text, "")
}

// CountTokenForModel 根据模型选择合适的编码器计算 Token 数量。
// 对于 tiktoken 不支持的模型（如 Claude/Gemini），降级为 cl100k_base，
// 再失败则按字符数估算。
func CountTokenForModel(text, model string) int {
	var tkm *tiktoken.Tiktoken
	var err error
	if model != "" {
		tkm, err = tiktoken.EncodingForModel(model)
	}
	if tkm == nil || err != nil {
		tkm, err = tiktoken.GetEncoding("cl100k_base")
	}
	if err != nil || tkm == nil {
		return len(text) / 4
	}
	return len(tkm.Encode(text, nil, nil))
}
