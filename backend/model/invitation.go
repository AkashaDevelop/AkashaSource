package model

type Invitation struct {
	Id          int     `json:"id" gorm:"primaryKey;autoIncrement"`
	Code        string  `json:"code" gorm:"type:varchar(32);uniqueIndex"`
	InviterId   int     `json:"inviter_id" gorm:"index"`
	InviteeId   int     `json:"invitee_id" gorm:"index"` // 谁使用了
	Status      int     `json:"status" gorm:"default:1"` // 1: 未使用, 2: 已使用
	Cost        float64 `json:"cost" gorm:"default:0"`   // 生成时的成本 (Quota)
	CreatedAt   int64   `json:"created_at" gorm:"autoCreateTime"`
	UsedAt      int64   `json:"used_at"`
}

const (
	InvitationStatusUnused = 1
	InvitationStatusUsed   = 2
)
