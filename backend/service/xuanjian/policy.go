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
	MaxRequestsPerWin    int   `json:"max_requests_per_win"`    // 默认 300
	MaxQuotaPerWin       int64 `json:"max_quota_per_win"`       // 默认 10_000_000
	MaxModelsPerWin      int   `json:"max_models_per_win"`      // 默认 8
	MaxIPCIDRsPerWin     int   `json:"max_ip_cidrs_per_win"`    // 默认 15（/24 CIDR 聚合）
	MaxTokensPerUser     int   `json:"max_tokens_per_user"`     // 默认 8（5min 内）
	ShortPromptMaxTokens int   `json:"short_prompt_max_tokens"` // 默认 20

	// 检测组开关（各自可独立禁用，方便误报期间单组关闭）
	EnableAbuseDetection     bool `json:"enable_abuse_detection"`
	EnableJailbreakDetection bool `json:"enable_jailbreak_detection"`
	EnableLLMAbuse           bool `json:"enable_llm_abuse"`
	EnableAgentDetection     bool `json:"enable_agent_detection"`
	EnableDuplicateDetection bool `json:"enable_duplicate_detection"`

	// 自动化节奏检测阈值～API 网关的客户端本来就是程序在调，
	// 间隔稳定是常态而不是罪状，所以门槛要卡得很高才有意义（见 detect_agent.go）
	BotStdDevMs       float64 `json:"bot_stddev_ms"`        // 间隔标准差低于此值才算可疑，默认 30
	BotMinRequests    int     `json:"bot_min_requests"`     // 且窗口内请求数需超过此值，默认 100
	RetryStormMinHits int     `json:"retry_storm_min_hits"` // 重试风暴：相似 prompt 次数阈值，默认 10

	// LLMjacking 专属配置
	LLMJackingNewTokenHours int     `json:"llmjacking_new_token_hours"` // 新 token 判定窗口，默认 24
	LLMJackingQuotaMultiple float64 `json:"llmjacking_quota_multiple"`  // 超出平均值倍数，默认 10

	// 处置配置
	NotifyAdmin      bool `json:"notify_admin"`       // protect 及以上模式是否通知管理员
	AutoDisableScore int  `json:"auto_disable_score"` // strict 模式下自动封 token 阈值，默认 90
	AutoBanScore     int  `json:"auto_ban_score"`     // strict 模式下自动封用户阈值，默认 95

	// 处置力度配置（供规则 Action 落地时取值，全部可热更新）
	//
	// ⚠ 时长类字段的约定（2026.8.4 统一）：
	//     正数 = 具体分钟数
	//     -1   = 永久（制裁层把 <=0 一律当永久处理）
	//      0   = 未配置，normalize() 会补成安全默认值
	//   之所以要区分 0 和 -1：JSON 里缺字段时反序列化出来就是 0，
	//   如果直接把 0 当"永久"，那么任何一个老版本存的配置在升级后
	//   都会静默变成"永久封禁"，这个后果太重了 (；￣Д￣)
	ThrottleFactor                float64 `json:"throttle_factor"`                  // throttle 降速倍率，默认 0.3（降到30%）
	ThrottleDurationMinutes       int     `json:"throttle_duration_minutes"`        // throttle/rpm_limit 持续分钟，默认 15
	PenaltyRPM                    int     `json:"penalty_rpm"`                      // rpm_limit 固定 RPM，默认 5
	SuspendDurationMinutes        int     `json:"suspend_duration_minutes"`         // suspend_token 停用分钟，默认 30
	BillingPenaltyFactor          float64 `json:"billing_penalty_factor"`           // billing_penalty 计费倍率，默认 3.0
	BillingPenaltyDurationMinutes int     `json:"billing_penalty_duration_minutes"` // 计费惩罚持续分钟，默认 60
	BanIPDurationMinutes          int     `json:"ban_ip_duration_minutes"`          // ban_ip 封禁分钟，默认 1440（一天）

	// 白名单豁免（这些 token/用户不参与任何行为检测）
	ExemptTokenIDs []int `json:"exempt_token_ids"`
	ExemptUserIDs  []int `json:"exempt_user_ids"`

	// ── AI 审核配置 ──────────────────────────────────────────────────────
	// ai_review_mode:
	//   "off"  — 关闭 AI 审核
	//   "pre"  — AI 预审：用户消息先经 AI 审核，通过才转发给实际模型
	//   "re"   — 规则引擎初审 + AI 复审：规则命中后再用 AI 复确认
	//   "both" — 先 AI 预审，通过后规则初审命中再 AI 复审
	AIReviewMode         string `json:"ai_review_mode"`           // off/pre/re/both
	AIReviewChannelID    int    `json:"ai_review_channel_id"`     // 审核用渠道 ID（从已有渠道选一个）
	AIReviewModel        string `json:"ai_review_model"`          // 审核用模型名（如 gpt-4o-mini）
	AIReviewTimeoutSec   int    `json:"ai_review_timeout_sec"`    // 审核超时秒数，默认 6（预审是同步阻塞的，超时直接加在用户延迟上）
	AIReviewBlockScore   int    `json:"ai_review_block_score"`    // 风险分达到此值则拦截，默认 70
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
	Action   string // 建议处置：warn/notify/throttle/rpm_limit/suspend_token/billing_penalty/disable_token/ban_ip/ban_user
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
		EnableAbuseDetection:     true,
		EnableJailbreakDetection: true,
		EnableLLMAbuse:           true,
		// ～2026.8.4 修正：这个开关从上线起就一直是 false，理由写的是"agent 结构 FP 率高，
		// 待观察后开启"——可真正 FP 高的是 A1 规则里那批裸词（whoami / ls -la / exec(），
		// agent 结构检测本身（ReAct 格式 + shell 注入双条件）是很准的。
		// 现在规则误报已经治理完毕，这块防护该上岗了喵～
		EnableAgentDetection:     true,
		EnableDuplicateDetection: true,
		BotStdDevMs:              30,
		BotMinRequests:           100,
		RetryStormMinHits:        10,
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
		AIReviewTimeoutSec:   6,
		AIReviewBlockScore:   70,
		AIReviewMaxTextChars: 2000,
		AIReviewPrePrompt:    "",
		AIReviewRePrompt:     "",

		ThrottleFactor:                0.3,
		ThrottleDurationMinutes:       15,
		PenaltyRPM:                    5,
		SuspendDurationMinutes:        30,
		BillingPenaltyFactor:          3.0,
		BillingPenaltyDurationMinutes: 60,
		// ～2026.8.4 修正：这一项之前压根没写进默认值，于是取到 int 零值 0，
		// 而 0 在制裁表里的含义是**永久封禁**——注释明明写着"默认 1440"，
		// 结果只要管理员没在前端存过一次配置，封个 IP 就是无期徒刑 (；￣Д￣)
		BanIPDurationMinutes: 1440,
	}
}

// normalize 给配置补齐兜底值
//
// 管理员存的 JSON 可能只有一部分字段，或者某些数值被填成 0；
// 这些"缺口"如果直接拿去用，行为会很怪（比如封禁时长 0 = 永久）。
// 所以每次加载配置后都过一遍这里，把不合理的值拉回安全区间喵～
func (c *XJConfig) normalize() {
	if c.Mode == "" {
		c.Mode = ModeProtect
	}
	if c.WindowMinutes <= 0 {
		c.WindowMinutes = 5
	}
	if c.MaxRequestsPerWin <= 0 {
		c.MaxRequestsPerWin = 300
	}
	if c.MaxQuotaPerWin <= 0 {
		c.MaxQuotaPerWin = 10_000_000
	}
	if c.MaxModelsPerWin <= 0 {
		c.MaxModelsPerWin = 8
	}
	if c.MaxIPCIDRsPerWin <= 0 {
		c.MaxIPCIDRsPerWin = 15
	}
	if c.MaxTokensPerUser <= 0 {
		c.MaxTokensPerUser = 8
	}
	if c.ShortPromptMaxTokens <= 0 {
		c.ShortPromptMaxTokens = 20
	}
	if c.BotStdDevMs <= 0 {
		c.BotStdDevMs = 30
	}
	if c.BotMinRequests <= 0 {
		c.BotMinRequests = 100
	}
	if c.RetryStormMinHits <= 0 {
		c.RetryStormMinHits = 10
	}
	if c.AutoDisableScore <= 0 {
		c.AutoDisableScore = 90
	}
	if c.AutoBanScore <= 0 {
		c.AutoBanScore = 95
	}
	if c.AIReviewTimeoutSec <= 0 {
		c.AIReviewTimeoutSec = 6
	}
	if c.AIReviewBlockScore <= 0 {
		c.AIReviewBlockScore = 70
	}
	if c.AIReviewMaxTextChars <= 0 {
		c.AIReviewMaxTextChars = 2000
	}
	// 时长类字段：0 表示"没配"，补成安全默认值；想要永久请明确填 -1
	if c.ThrottleDurationMinutes == 0 {
		c.ThrottleDurationMinutes = 15
	}
	if c.SuspendDurationMinutes == 0 {
		c.SuspendDurationMinutes = 30
	}
	if c.BillingPenaltyDurationMinutes == 0 {
		c.BillingPenaltyDurationMinutes = 60
	}
	if c.BanIPDurationMinutes == 0 {
		c.BanIPDurationMinutes = 1440
	}
	if c.PenaltyRPM <= 0 {
		c.PenaltyRPM = 5
	}
	if c.ThrottleFactor <= 0 || c.ThrottleFactor >= 1.0 {
		c.ThrottleFactor = 0.3
	}
	if c.BillingPenaltyFactor <= 1.0 {
		c.BillingPenaltyFactor = 3.0
	}
	if c.LLMJackingNewTokenHours <= 0 {
		c.LLMJackingNewTokenHours = 24
	}
	if c.LLMJackingQuotaMultiple <= 0 {
		c.LLMJackingQuotaMultiple = 10.0
	}
}

var (
	cfgMu     sync.RWMutex
	globalCfg XJConfig
	enabled   bool
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
	cfg.normalize()
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
	cfg.normalize()
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
