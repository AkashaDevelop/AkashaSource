package model

const (
	ContextSanitizationScopeGlobal       = "global"
	ContextSanitizationScopeModel        = "model"
	ContextSanitizationScopeChannel      = "channel"
	ContextSanitizationScopeChannelModel = "channel_model"

	ContextSanitizationModeOff      = "off"
	ContextSanitizationModeMonitor  = "monitor"
	ContextSanitizationModeProtect  = "protect"
	ContextSanitizationModeBalanced = "balanced"
	ContextSanitizationModeStrict   = "strict"

	// 方向
	ContextSanitizationDirectionRequest  = "request"
	ContextSanitizationDirectionResponse = "response"

	// 阶段
	ContextSanitizationStageRequestDetect    = "request_detect"
	ContextSanitizationStageToolValidate     = "tool_validate"
	ContextSanitizationStageGuardInject      = "guard_inject"
	ContextSanitizationStageResponseAdDetect = "response_ad_detect"
	ContextSanitizationStageDegraded         = "degraded"
	ContextSanitizationStageCircuitOpen      = "circuit_open"

	// 动作
	ContextSanitizationActionMonitor          = "monitor"
	ContextSanitizationActionAllow            = "allow"
	ContextSanitizationActionInjectGuard      = "inject_guard"
	ContextSanitizationActionBlock            = "block"
	ContextSanitizationActionStripKnownSuffix = "strip_known_suffix"
	ContextSanitizationActionSkip             = "skip"
	ContextSanitizationActionDegrade          = "degrade"
)

type ContextSanitizationPolicy struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string `json:"name" gorm:"type:varchar(100);index"`
	Enabled   bool   `json:"enabled" gorm:"default:true;index"`
	Scope     string `json:"scope" gorm:"type:varchar(30);index"`
	ChannelId *int   `json:"channel_id" gorm:"index"`
	ModelName string `json:"model_name" gorm:"type:varchar(200);index"`
	Mode      string `json:"mode" gorm:"type:varchar(30);default:'monitor'"`
	Config    string `json:"config" gorm:"type:text"`
	Version   int    `json:"version" gorm:"default:1"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type ContextSanitizationPolicyRevision struct {
	Id         int    `json:"id" gorm:"primaryKey;autoIncrement"`
	PolicyId   int    `json:"policy_id" gorm:"index"`
	Version    int    `json:"version" gorm:"index"`
	Snapshot   string `json:"snapshot" gorm:"type:text"`
	CreatedAt  int64  `json:"created_at"`
	CreatedBy  int    `json:"created_by" gorm:"index"`
	ChangeNote string `json:"change_note" gorm:"type:varchar(255)"`
}

type ContextSanitizationEvent struct {
	Id             int    `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt      int64  `json:"created_at" gorm:"index"`
	RequestId      string `json:"request_id" gorm:"type:varchar(128);index"`
	UserId         int    `json:"user_id" gorm:"index"`
	TokenId        int    `json:"token_id" gorm:"index"`
	TokenName      string `json:"token_name" gorm:"type:varchar(100);index"`
	ChannelId      int    `json:"channel_id" gorm:"index"`
	ChannelName    string `json:"channel_name" gorm:"type:varchar(100);index"`
	RequestedModel string `json:"requested_model" gorm:"type:varchar(200);index"`
	MappedModel    string `json:"mapped_model" gorm:"type:varchar(200);index"`
	PolicyId       int    `json:"policy_id" gorm:"index"`
	PolicyVersion  int    `json:"policy_version"`
	Direction      string `json:"direction" gorm:"type:varchar(30);index"`
	Stage          string `json:"stage" gorm:"type:varchar(50);index"`
	Mode           string `json:"mode" gorm:"type:varchar(30);index"`
	Action         string `json:"action" gorm:"type:varchar(50);index"`
	RiskScore      int    `json:"risk_score" gorm:"index"`
	Categories     string `json:"categories" gorm:"type:text"`
	Findings       string `json:"findings" gorm:"type:text"`
	Snippet        string `json:"snippet" gorm:"type:varchar(500)"`
	ContentHash    string `json:"content_hash" gorm:"type:varchar(128);index"`
	LatencyMs      int64  `json:"latency_ms"`
	Degraded       bool   `json:"degraded" gorm:"default:false;index"`
	CircuitState   string `json:"circuit_state" gorm:"type:varchar(30)"`
	Error          string `json:"error" gorm:"type:varchar(500)"`
}
