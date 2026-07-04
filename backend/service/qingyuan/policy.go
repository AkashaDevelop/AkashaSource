package qingyuan

import (
	"encoding/json"
	"errors"
	"regexp"

	"STfreApi/model"
)

const (
	ModeOff      = model.ContextSanitizationModeOff
	ModeMonitor  = model.ContextSanitizationModeMonitor
	ModeProtect  = model.ContextSanitizationModeProtect
	ModeBalanced = model.ContextSanitizationModeBalanced
	ModeStrict   = model.ContextSanitizationModeStrict
)

type PolicyConfig struct {
	Request        RequestConfig        `json:"request"`
	Detection      DetectionConfig      `json:"detection"`
	Risk           RiskConfig           `json:"risk"`
	Tools          ToolsConfig          `json:"tools"`
	Response       ResponseConfig       `json:"response"`
	Trust          TrustConfig          `json:"trust"`
	Logging        LoggingConfig        `json:"logging"`
	CircuitBreaker CircuitBreakerConfig `json:"circuit_breaker"`
	Degradation    DegradationConfig    `json:"degradation"`
}

// TrustConfig 宸汐清源 · 信任边界配置～
// 对应 2026 年"默认不信任非运营者内容"的架构原则：
// 检索文档/工具输出/工具自我介绍命中攻击特征时，风险分要比普通文本放得更大声
type TrustConfig struct {
	EnableProvenance           bool    `json:"enable_provenance"`
	RetrievedDocRiskMultiplier float64 `json:"retrieved_doc_risk_multiplier"`
	ToolOutputRiskMultiplier   float64 `json:"tool_output_risk_multiplier"`
	ScanToolDescriptions       bool    `json:"scan_tool_descriptions"`
	ScanMemoryPoisoning        bool    `json:"scan_memory_poisoning"`
	MemoryScanMaxMessages      int     `json:"memory_scan_max_messages"`
}

type RequestConfig struct {
	InjectGuard                bool   `json:"inject_guard"`
	GuardPosition              string `json:"guard_position"`
	GuardLanguage              string `json:"guard_language"`
	PreserveUserSystemMessages bool   `json:"preserve_user_system_messages"`
}

type DetectionConfig struct {
	RemoveZeroWidthForDetection    bool `json:"remove_zero_width_for_detection"`
	DecodeJSONEscapesForDetection  bool `json:"decode_json_escapes_for_detection"`
	DecodeURLEncodingForDetection  bool `json:"decode_url_encoding_for_detection"`
	DecodeHTMLEntitiesForDetection bool `json:"decode_html_entities_for_detection"`
	DetectBase64                   bool `json:"detect_base64"`
	MaxDecodeLayers                int  `json:"max_decode_layers"`
	MaxScanChars                   int  `json:"max_scan_chars"`
}

type RiskConfig struct {
	AnnotateThreshold        int  `json:"annotate_threshold"`
	BlockThreshold           int  `json:"block_threshold"`
	BlockStructuralToolAbuse bool `json:"block_structural_tool_abuse"`
}

type ToolsConfig struct {
	Enabled                    bool     `json:"enabled"`
	ValidateToolSchema         bool     `json:"validate_tool_schema"`
	ValidateToolChoice         bool     `json:"validate_tool_choice"`
	ValidateAssistantToolCalls bool     `json:"validate_assistant_tool_calls"`
	AllowedToolNames           []string `json:"allowed_tool_names"`
	BlockedToolNames           []string `json:"blocked_tool_names"`
	ToolNameRegex              string   `json:"tool_name_regex"`
	MaxTools                   int      `json:"max_tools"`
	MaxToolSchemaBytes         int      `json:"max_tool_schema_bytes"`
	MaxToolArgumentsBytes      int      `json:"max_tool_arguments_bytes"`
}

type ResponseConfig struct {
	DetectAds                bool     `json:"detect_ads"`
	AdPolicy                 string   `json:"ad_policy"`
	AdConfidenceThreshold    int      `json:"ad_confidence_threshold"`
	KnownAdPatterns          []string `json:"known_ad_patterns"`
	PreserveCodeBlocks       bool     `json:"preserve_code_blocks"`
	PreserveJSONOutput       bool     `json:"preserve_json_output"`
	ValidateOutputToolCalls  bool     `json:"validate_output_tool_calls"`
	BlockInvalidToolCalls    bool     `json:"block_invalid_output_tool_calls"`
	StreamTailBufferSize     int      `json:"stream_tail_buffer_size"`
	DetectPromptInjection    bool     `json:"detect_prompt_injection"`
	DetectMultimodal         bool     `json:"detect_multimodal"`
	DetectThinkingAttacks    bool     `json:"detect_thinking_attacks"`
}

type LoggingConfig struct {
	LogEvents       bool `json:"log_events"`
	LogRawContent   bool `json:"log_raw_content"`
	LogSnippetChars int  `json:"log_snippet_chars"`
	HashContent     bool `json:"hash_content"`
}

type CircuitBreakerConfig struct {
	Enabled          bool `json:"enabled"`
	FailureThreshold int  `json:"failure_threshold"`
	TimeoutPerReqMs  int  `json:"timeout_per_req_ms"`
	CooldownSeconds  int  `json:"cooldown_seconds"`
}

type DegradationConfig struct {
	MonitorTimeoutAction string `json:"monitor_timeout_action"`
	ProtectTimeoutAction string `json:"protect_timeout_action"`
}

type ResolvedPolicy struct {
	Policy *model.ContextSanitizationPolicy
	Config PolicyConfig
	Source string
}

func DefaultConfig() PolicyConfig {
	return PolicyConfig{
		Request: RequestConfig{
			InjectGuard:                true,
			GuardPosition:              "first_system",
			GuardLanguage:              "auto",
			PreserveUserSystemMessages: true,
		},
		Detection: DetectionConfig{
			RemoveZeroWidthForDetection:    true,
			DecodeJSONEscapesForDetection:  true,
			DecodeURLEncodingForDetection:  true,
			DecodeHTMLEntitiesForDetection: true,
			DetectBase64:                   true,
			MaxDecodeLayers:                2,
			MaxScanChars:                   65536,
		},
		Risk: RiskConfig{AnnotateThreshold: 40, BlockThreshold: 85, BlockStructuralToolAbuse: true},
		Tools: ToolsConfig{
			Enabled:                    true,
			ValidateToolSchema:         true,
			ValidateToolChoice:         true,
			ValidateAssistantToolCalls: true,
			ToolNameRegex:              "^[a-zA-Z0-9_-]{1,64}$",
			MaxTools:                   128,
			MaxToolSchemaBytes:         65536,
			MaxToolArgumentsBytes:      262144,
		},
		Response: ResponseConfig{
			DetectAds:               true,
			AdPolicy:                "monitor",
			AdConfidenceThreshold:   75,
			PreserveCodeBlocks:      true,
			PreserveJSONOutput:      true,
			ValidateOutputToolCalls: true,
			BlockInvalidToolCalls:   false,
			StreamTailBufferSize:    2048,
			DetectPromptInjection:   true,
			DetectMultimodal:        true,
			DetectThinkingAttacks:   true,
		},
		Trust: TrustConfig{
			EnableProvenance:           true,
			RetrievedDocRiskMultiplier: 1.6,
			ToolOutputRiskMultiplier:   1.4,
			ScanToolDescriptions:       true,
			ScanMemoryPoisoning:        true,
			MemoryScanMaxMessages:      50,
		},
		Logging:        LoggingConfig{LogEvents: true, LogSnippetChars: 160, HashContent: true},
		CircuitBreaker: CircuitBreakerConfig{Enabled: true, FailureThreshold: 5, TimeoutPerReqMs: 500, CooldownSeconds: 30},
		Degradation:    DegradationConfig{MonitorTimeoutAction: "skip", ProtectTimeoutAction: "fallback_to_guard"},
	}
}

func MergeConfig(raw string) (PolicyConfig, error) {
	cfg := DefaultConfig()
	if raw == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return cfg, err
	}
	if cfg.Request.GuardPosition == "" {
		cfg.Request.GuardPosition = "first_system"
	}
	if cfg.Request.GuardLanguage == "" {
		cfg.Request.GuardLanguage = "auto"
	}
	if cfg.Detection.MaxDecodeLayers <= 0 {
		cfg.Detection.MaxDecodeLayers = 2
	}
	if cfg.Detection.MaxScanChars <= 0 {
		cfg.Detection.MaxScanChars = 65536
	}
	if cfg.Tools.ToolNameRegex == "" {
		cfg.Tools.ToolNameRegex = "^[a-zA-Z0-9_-]{1,64}$"
	}
	if cfg.Tools.MaxTools <= 0 {
		cfg.Tools.MaxTools = 128
	}
	if cfg.Tools.MaxToolSchemaBytes <= 0 {
		cfg.Tools.MaxToolSchemaBytes = 65536
	}
	if cfg.Response.AdPolicy == "" {
		cfg.Response.AdPolicy = "monitor"
	}
	if cfg.Logging.LogSnippetChars <= 0 {
		cfg.Logging.LogSnippetChars = 160
	}
	if cfg.CircuitBreaker.TimeoutPerReqMs <= 0 {
		cfg.CircuitBreaker.TimeoutPerReqMs = 500
	}
	if cfg.CircuitBreaker.FailureThreshold <= 0 {
		cfg.CircuitBreaker.FailureThreshold = 5
	}
	if cfg.CircuitBreaker.CooldownSeconds <= 0 {
		cfg.CircuitBreaker.CooldownSeconds = 30
	}
	if cfg.Trust.RetrievedDocRiskMultiplier <= 0 {
		cfg.Trust.RetrievedDocRiskMultiplier = 1.6
	}
	if cfg.Trust.ToolOutputRiskMultiplier <= 0 {
		cfg.Trust.ToolOutputRiskMultiplier = 1.4
	}
	if cfg.Trust.MemoryScanMaxMessages <= 0 {
		cfg.Trust.MemoryScanMaxMessages = 50
	}
	return cfg, nil
}

func ValidatePolicy(p *model.ContextSanitizationPolicy) error {
	if p.Name == "" {
		return errors.New("策略名称不能为空")
	}
	if !validScope(p.Scope) {
		return errors.New("策略范围无效")
	}
	if !validMode(p.Mode) {
		return errors.New("策略模式无效")
	}
	if p.Scope == model.ContextSanitizationScopeModel && p.ModelName == "" {
		return errors.New("模型策略必须指定模型名称")
	}
	if p.Scope == model.ContextSanitizationScopeChannel && p.ChannelId == nil {
		return errors.New("渠道策略必须指定渠道")
	}
	if p.Scope == model.ContextSanitizationScopeChannelModel && (p.ChannelId == nil || p.ModelName == "") {
		return errors.New("渠道模型策略必须指定渠道和模型")
	}
	if p.Config != "" {
		if _, err := MergeConfig(p.Config); err != nil {
			return err
		}
	}
	if _, err := regexp.Compile(DefaultConfig().Tools.ToolNameRegex); err != nil {
		return err
	}
	return nil
}

func validScope(scope string) bool {
	switch scope {
	case model.ContextSanitizationScopeGlobal, model.ContextSanitizationScopeModel, model.ContextSanitizationScopeChannel, model.ContextSanitizationScopeChannelModel:
		return true
	default:
		return false
	}
}

func validMode(mode string) bool {
	switch mode {
	case ModeOff, ModeMonitor, ModeProtect, ModeBalanced, ModeStrict:
		return true
	default:
		return false
	}
}

func IsEnabled(policy ResolvedPolicy) bool {
	return policy.Policy != nil && policy.Policy.Enabled && policy.Policy.Mode != ModeOff
}
