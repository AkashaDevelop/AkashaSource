package model

type Redemption struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string `json:"name" gorm:"index"`
	Code      string `json:"code" gorm:"unique;index"`
	Quota     int64  `json:"quota"`
	CreatedBy int    `json:"created_by"`
	UsedBy    int    `json:"used_by" gorm:"index"`    // 0 if unused
	Status    int    `json:"status" gorm:"default:1"` // 1: unused, 2: used, 3: disabled
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime"`
	UsedAt    int64  `json:"used_at"`
}

const (
	RedemptionStatusUnused   = 1
	RedemptionStatusUsed     = 2
	RedemptionStatusDisabled = 3
)
