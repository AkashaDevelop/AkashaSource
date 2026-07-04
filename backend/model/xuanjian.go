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
