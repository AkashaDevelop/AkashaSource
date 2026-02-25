package model

type Token struct {
	Id             int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId         int    `json:"user_id" gorm:"index"`
	Key            string `json:"key" gorm:"type:char(48);uniqueIndex"`
	Status         int    `json:"status" gorm:"default:1"` // 1: 正常, 2: 禁用, 3: 过期
	Name           string `json:"name" gorm:"index"`
	CreatedTime    int64  `json:"created_time"`
	AccessedTime   int64  `json:"accessed_time"`
	ExpiredTime    int64  `json:"expired_time" gorm:"default:-1"` // -1: 永不过期
	RemainQuota    int64  `json:"remain_quota" gorm:"default:0"`  // 0: 无限 (如果 UnlimitedQuota 为 true)
	UnlimitedQuota bool   `json:"unlimited_quota" gorm:"default:false"`
	UsedQuota      int64  `json:"used_quota" gorm:"default:0"` // 该令牌的用量
}

const (
	TokenStatusActive   = 1
	TokenStatusDisabled = 2
	TokenStatusExpired  = 3
)
