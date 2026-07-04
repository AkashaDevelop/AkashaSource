package model

// ～宸汐玄鉴·行为风控事件表～
// 玄：深邃洞察；鉴：照妖辨善之镜。
// 每一条异常行为都逃不过阿卡夏的镜子，在这里留下痕迹 (｡•ᴗ•｡)

// XuanJianEvent 行为风控事件记录
type XuanJianEvent struct {
	Id           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt    int64  `json:"created_at" gorm:"index"`
	TokenID      int    `json:"token_id" gorm:"index"`
	TokenName    string `json:"token_name" gorm:"type:varchar(100);index"`
	UserID       int    `json:"user_id" gorm:"index"`
	RiskScore    int    `json:"risk_score" gorm:"index"`
	FindingType  string `json:"finding_type" gorm:"type:varchar(60);index"`
	FindingGroup string `json:"finding_group" gorm:"type:varchar(20);index"` // llmjacking/jailbreak/malware_gen/reverse_eng/agent_abuse
	Action       string `json:"action" gorm:"type:varchar(30)"`              // warn/notify/throttle/disable_token/ban_user
	Evidence     string `json:"evidence" gorm:"type:text"`                  // JSON 快照：触发该 Finding 的具体证据
	IP           string `json:"ip" gorm:"type:varchar(64)"`
	Model        string `json:"model" gorm:"type:varchar(200)"`
}

// XuanJianRule 行为风控检测规则（内置 28 条 + 管理员自定义均落库，可视化编辑）
type XuanJianRule struct {
	Id                  int    `json:"id" gorm:"primaryKey;autoIncrement"`
	RuleKey             string `json:"rule_key" gorm:"type:varchar(40);uniqueIndex"` // 内置规则用原 ID（如 "J1"），自定义规则用 "custom_<id>"
	FindingType         string `json:"finding_type" gorm:"type:varchar(60)"`
	Group               string `json:"group" gorm:"type:varchar(20);index"`
	BaseScore           int    `json:"base_score"`
	KeywordsJSON        string `json:"-" gorm:"type:text"`               // JSON 数组，内部存储
	RequireContextJSON  string `json:"-" gorm:"type:text"`               // JSON 数组，可空
	Keywords            []string `json:"keywords" gorm:"-"`              // 供 API 读写用，不落库
	RequireContext      []string `json:"require_context" gorm:"-"`       // 供 API 读写用，不落库
	PromptOnly          bool   `json:"prompt_only"`
	MinCompletionTokens int    `json:"min_completion_tokens"`
	Action              string `json:"action" gorm:"type:varchar(30)"` // warn/notify/throttle/disable_token/ban_user
	Enabled             bool   `json:"enabled" gorm:"default:true;index"`
	IsBuiltin           bool   `json:"is_builtin" gorm:"default:false"`
	CreatedAt           int64  `json:"created_at"`
	UpdatedAt           int64  `json:"updated_at"`
}

