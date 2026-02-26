package model

type Group struct {
	Id             int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name           string `json:"name" gorm:"uniqueIndex;type:varchar(100)"`
	Description    string `json:"description" gorm:"type:varchar(500)"`
	ModelRatios    string `json:"model_ratios" gorm:"type:text"`    // JSON: {"gpt-4": 1.5}
	AllowedChannels string `json:"allowed_channels" gorm:"type:text"` // 逗号分隔渠道ID
	QPM            int    `json:"qpm" gorm:"default:0"`              // 每分钟请求数，0=不限
	CreatedAt      int64  `json:"created_at"`
}
