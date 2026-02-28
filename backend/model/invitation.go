package model

type Invitation struct {
	Id        int     `json:"id" gorm:"primaryKey;autoIncrement"`
	Code      string  `json:"code" gorm:"type:varchar(64);uniqueIndex"`
	InviterId int     `json:"inviter_id" gorm:"index"`
	InviteeId int     `json:"invitee_id" gorm:"index"`
	Status    int     `json:"status" gorm:"default:1"` // 1: 活跃, 2: 已用完
	MaxUses   int     `json:"max_uses" gorm:"default:1"` // 0 = unlimited
	UsedCount int     `json:"used_count" gorm:"default:0"`
	Cost      float64 `json:"cost" gorm:"default:0"`
	CreatedAt int64   `json:"created_at" gorm:"autoCreateTime"`
	UsedAt    int64   `json:"used_at"`
}

const (
	InvitationStatusUnused = 1
	InvitationStatusUsed   = 2
)
