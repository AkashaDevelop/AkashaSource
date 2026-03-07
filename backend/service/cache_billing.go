package service

import "STfreApi/dto"

// GetCachedTokens 从 Usage 中提取缓存 token 数量
func GetCachedTokens(usage *dto.Usage) int {
	if usage == nil {
		return 0
	}

	// OpenAI format: prompt_tokens_details.cached_tokens
	if usage.PromptTokensDetails != nil && usage.PromptTokensDetails.CachedTokens > 0 {
		return usage.PromptTokensDetails.CachedTokens
	}

	// Claude format: input_tokens_details.cached_tokens
	if usage.InputTokensDetails != nil && usage.InputTokensDetails.CachedTokens > 0 {
		return usage.InputTokensDetails.CachedTokens
	}

	// DeepSeek/Qwen format: prompt_cache_hit_tokens
	if usage.PromptCacheHitTokens > 0 {
		return usage.PromptCacheHitTokens
	}

	// Legacy format: cached_tokens
	if usage.CachedTokens > 0 {
		return usage.CachedTokens
	}

	return 0
}

// GetPromptTokens 获取 prompt tokens（兼容 OpenAI 和 Claude）
func GetPromptTokens(usage *dto.Usage) int {
	if usage == nil {
		return 0
	}
	if usage.PromptTokens > 0 {
		return usage.PromptTokens
	}
	return usage.InputTokens
}

// GetCompletionTokens 获取 completion tokens（兼容 OpenAI 和 Claude）
func GetCompletionTokens(usage *dto.Usage) int {
	if usage == nil {
		return 0
	}
	if usage.CompletionTokens > 0 {
		return usage.CompletionTokens
	}
	return usage.OutputTokens
}
