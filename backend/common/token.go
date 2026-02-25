package common

import "github.com/pkoukk/tiktoken-go"

// CountToken 计算 Token 数量
// 这是一个简化版本，实际需要根据模型不同选择不同的编码器
func CountToken(text string) int {
	// 默认使用 cl100k_base (GPT-4, GPT-3.5-Turbo)
	tkm, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return len(text) / 4 // 降级处理：按字符估算
	}
	token := tkm.Encode(text, nil, nil)
	return len(token)
}
