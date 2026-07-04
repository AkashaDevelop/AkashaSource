package xuanjian

// ～宸汐玄鉴·策略配置中心～ (｡•ᴗ•｡)
// 所有阈值、开关、豁免列表都在这里管理，
// 超管在前端改一下，这里就能热更新，不用重启哦！

import (
	"encoding/json"
	"sync"

	"STfreApi/common"
	"STfreApi/model"
)

const (
	ModeOff     = "off"
	ModeMonitor = "monitor"
	ModeProtect = "protect"
	ModeStrict  = "strict"
)

// AI 审核模式
const (
	AIReviewOff  = "off"  // 关闭 AI 审核
	AIReviewPre  = "pre"  // AI 预审：用户消息先经 AI 审核
	AIReviewRe   = "re"   // 规则引擎初审 + AI 复审
	AIReviewBoth = "both" // 先 AI 预审，再规则初审 + AI 复审
)

// XJConfig 宸汐玄鉴完整策略配置
type XJConfig struct {
	Mode string `json:"mode"` // off/monitor/protect/strict

	// 滑动窗口
	WindowMinutes int `json:"window_minutes"` // 默认 5

	// 行为阈值（误报修正后的默认值）
	MaxRequestsPerWin    int     `json:"max_requests_per_win"`     // 默认 300
	MaxQuotaPerWin       int64   `json:"max_quota_per_win"`        // 默认 10_000_000
	MaxModelsPerWin      int     `json:"max_models_per_win"`       // 默认 8
	MaxIPCIDRsPerWin     int     `json:"max_ip_cidrs_per_win"`     // 默认 15（/24 CIDR 聚合）
	MaxTokensPerUser     int     `json:"max_tokens_per_user"`      // 默认 8（5min 内）
	ShortPromptMaxTokens int     `json:"short_prompt_max_tokens"`  // 默认 20
	ShortPromptRatio     float64 `json:"short_prompt_ratio"`       // 默认 0.7

	// 检测组开关（各自可独立禁用，方便误报期间单组关闭）
	EnableAbuseDetection     bool `json:"enable_abuse_detection"`
	EnableJailbreakDetection bool `json:"enable_jailbreak_detection"`
	EnableLLMAbuse           bool `json:"enable_llm_abuse"`
	EnableAgentDetection     bool `json:"enable_agent_detection"`
	EnableDuplicateDetection bool `json:"enable_duplicate_detection"`

	// LLMjacking 专属配置
	LLMJackingNewTokenHours int     `json:"llmjacking_new_token_hours"` // 新 token 判定窗口，默认 24
	LLMJackingQuotaMultiple float64 `json:"llmjacking_quota_multiple"`  // 超出平均值倍数，默认 10

	// 处置配置
	NotifyAdmin      bool `json:"notify_admin"`       // protect 及以上模式是否通知管理员
	AutoDisableScore int  `json:"auto_disable_score"` // strict 模式下自动封 token 阈值，默认 90
	AutoBanScore     int  `json:"auto_ban_score"`     // strict 模式下自动封用户阈值，默认 95

	// 白名单豁免（这些 token/用户不参与任何行为检测）
	ExemptTokenIDs []int `json:"exempt_token_ids"`
	ExemptUserIDs  []int `json:"exempt_user_ids"`

	// ── AI 审核配置 ──────────────────────────────────────────────────────
	// ai_review_mode:
	//   "off"  — 关闭 AI 审核
	//   "pre"  — AI 预审：用户消息先经 AI 审核，通过才转发给实际模型
	//   "re"   — 规则引擎初审 + AI 复审：规则命中后再用 AI 复确认
	//   "both" — 先 AI 预审，通过后规则初审命中再 AI 复审
	AIReviewMode         string `json:"ai_review_mode"`          // off/pre/re/both
	AIReviewChannelID    int    `json:"ai_review_channel_id"`    // 审核用渠道 ID（从已有渠道选一个）
	AIReviewModel        string `json:"ai_review_model"`         // 审核用模型名（如 gpt-4o-mini）
	AIReviewTimeoutSec   int    `json:"ai_review_timeout_sec"`   // 审核超时秒数，默认 10
	AIReviewBlockScore   int    `json:"ai_review_block_score"`   // 风险分达到此值则拦截，默认 70
	AIReviewMaxTextChars int    `json:"ai_review_max_text_chars"` // 送审文本最大字符数，默认 2000

	// 自定义审核提示词（留空则使用内置默认提示词）
	AIReviewPrePrompt string `json:"ai_review_pre_prompt"` // AI 预审系统提示词
	AIReviewRePrompt  string `json:"ai_review_re_prompt"`  // AI 复审系统提示词
}

// Finding 玄鉴检测到的单个风险发现
type Finding struct {
	Type     string // Finding 类型，如 llmjacking_scanner
	Group    string // 分组：llmjacking/jailbreak/malware_gen/reverse_eng/agent_abuse
	Score    int    // 风险分 0-100
	Evidence string // 触发的具体证据片段（纯文本，不含敏感原始内容）
	Action   string // 建议处置：warn/notify/throttle/disable_token/ban_user
}

// DefaultConfig 返回开箱即用的默认配置
func DefaultConfig() XJConfig {
	return XJConfig{
		Mode:                     ModeProtect,
		WindowMinutes:            5,
		MaxRequestsPerWin:        300,
		MaxQuotaPerWin:           10_000_000,
		MaxModelsPerWin:          8,
		MaxIPCIDRsPerWin:         15,
		MaxTokensPerUser:         8,
		ShortPromptMaxTokens:     20,
		ShortPromptRatio:         0.7,
		EnableAbuseDetection:     true,
		EnableJailbreakDetection: true,
		EnableLLMAbuse:           true,
		EnableAgentDetection:     false, // 初期关闭，agent 结构FP率高，待观察后开启
		EnableDuplicateDetection: true,
		LLMJackingNewTokenHours:  24,
		LLMJackingQuotaMultiple:  10.0,
		NotifyAdmin:              true,
		AutoDisableScore:         90,
		AutoBanScore:             95,
		ExemptTokenIDs:           []int{},
		ExemptUserIDs:            []int{},

		AIReviewMode:         "off",
		AIReviewChannelID:    0,
		AIReviewModel:        "",
		AIReviewTimeoutSec:   10,
		AIReviewBlockScore:   70,
		AIReviewMaxTextChars: 2000,
		AIReviewPrePrompt:    "",
		AIReviewRePrompt:     "",
	}
}

var (
	cfgMu      sync.RWMutex
	globalCfg  XJConfig
	enabled    bool
)

// LoadConfig 从 options 表读取配置到内存
func LoadConfig() {
	common.OptionLock.RLock()
	enabledVal := common.OptionMap[model.OptionKeyXuanJianEnabled]
	policyVal := common.OptionMap[model.OptionKeyXuanJianPolicy]
	common.OptionLock.RUnlock()

	cfgMu.Lock()
	defer cfgMu.Unlock()

	enabled = enabledVal == "true"

	cfg := DefaultConfig()
	if policyVal != "" {
		_ = json.Unmarshal([]byte(policyVal), &cfg)
	}
	globalCfg = cfg
}

// GetConfig 返回当前生效的策略配置副本
func GetConfig() (XJConfig, bool) {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	return globalCfg, enabled
}

// IsEnabled 快速判断模块是否启用（O(1)，对主链路零开销）
func IsEnabled() bool {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	return enabled && globalCfg.Mode != ModeOff
}

// UpdateConfig 热更新配置（管理员保存后调用）
func UpdateConfig(cfg XJConfig, isEnabled bool) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	enabledStr := "false"
	if isEnabled {
		enabledStr = "true"
	}
	if err := common.DB.Save(&model.Option{Key: model.OptionKeyXuanJianEnabled, Value: enabledStr}).Error; err != nil {
		return err
	}
	if err := common.DB.Save(&model.Option{Key: model.OptionKeyXuanJianPolicy, Value: string(data)}).Error; err != nil {
		return err
	}
	common.UpdateOptionMap(model.OptionKeyXuanJianEnabled, enabledStr)
	common.UpdateOptionMap(model.OptionKeyXuanJianPolicy, string(data))

	cfgMu.Lock()
	globalCfg = cfg
	enabled = isEnabled
	cfgMu.Unlock()
	return nil
}

// IsExempt 判断 token/user 是否在白名单中
func IsExempt(tokenID, userID int) bool {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	for _, id := range globalCfg.ExemptTokenIDs {
		if id == tokenID {
			return true
		}
	}
	for _, id := range globalCfg.ExemptUserIDs {
		if id == userID {
			return true
		}
	}
	return false
}
