package context_sanitizer

// 初始化函数,在服务启动时调用
func Init() error {
	// 加载策略缓存
	if err := ReloadPolicyCache(); err != nil {
		return err
	}

	// 启动 token 风险追踪清理任务
	InitTokenRiskCleanupJob()

	return nil
}

// GetVersion 返回模块版本
func GetVersion() string {
	return "1.0.0-enhanced"
}

// GetFeatures 返回已启用的功能列表
func GetFeatures() map[string]bool {
	return map[string]bool{
		"request_detection":          true,
		"tool_structure_validation":  true,
		"guard_injection":            true,
		"response_ad_detection":      true,
		"response_tool_validation":   true,
		"multimodal_detection":       true,
		"thinking_attack_detection":  true,
		"context_window_weighting":   true,
		"token_risk_tracking":        true,
		"advanced_obfuscation":       true,
		"stream_processing":          true,
		"circuit_breaker":            true,
		"policy_versioning":          true,
		"adaptive_defense":           true,
	}
}
