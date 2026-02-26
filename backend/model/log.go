package model

type Log struct {
	Id               int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId           int    `json:"user_id" gorm:"index"`
	CreatedAt        int64  `json:"created_at" gorm:"index"`
	Type             int    `json:"type" gorm:"default:0"` // 1: 消费, 2: 充值, 3: 系统
	Content          string `json:"content" gorm:"type:text"`
	Username         string `json:"username" gorm:"index"`
	TokenName        string `json:"token_name" gorm:"index"`
	ModelName        string `json:"model_name" gorm:"index"`
	Quota            int64  `json:"quota" gorm:"default:0"`
	PromptTokens     int    `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens int    `json:"completion_tokens" gorm:"default:0"`
	CachedTokens     int    `json:"cached_tokens" gorm:"default:0"`
	ChannelId        int    `json:"channel_id" gorm:"index"`
}

const (
	LogTypeConsume = 1
	LogTypeTopup   = 2
	LogTypeSystem  = 3
	LogTypeFail    = 4
)
