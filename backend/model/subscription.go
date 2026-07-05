package model

import (
	"STfreApi/common"
	"time"
)

type SubscriptionPlan struct {
	Id           int     `json:"id" gorm:"primaryKey;autoIncrement"`
	Name         string  `json:"name" gorm:"type:varchar(100)"`
	Description  string  `json:"description" gorm:"type:text"`
	Price        float64 `json:"price"`         // CNY
	DurationDays int     `json:"duration_days"` // 0 = permanent
	Type         string  `json:"type" gorm:"type:varchar(20)"` // group, quota, rpm, combo
	GroupName    string  `json:"group_name" gorm:"type:varchar(100)"`
	Quota        int64   `json:"quota"` // quota units to add
	RPM          int     `json:"rpm"`   // requests per minute limit
	Enabled      bool    `json:"enabled" gorm:"default:true"`
	SortOrder    int     `json:"sort_order" gorm:"default:0"`
	CreatedAt    int64   `json:"created_at"`
}

type UserSubscription struct {
	Id            int               `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId        int               `json:"user_id" gorm:"index"`
	PlanId        int               `json:"plan_id"`
	Plan          *SubscriptionPlan `json:"plan,omitempty" gorm:"foreignKey:PlanId"`
	Status        int               `json:"status" gorm:"default:0"`
	OriginalGroup string            `json:"original_group" gorm:"type:varchar(100)"`
	Quota         int64             `json:"quota" gorm:"default:0"`     // 订阅额度总量（拷贝自 Plan.Quota）
	UsedQuota     int64             `json:"used_quota" gorm:"default:0"` // 订阅已用额度
	StartedAt     int64             `json:"started_at"`
	ExpiredAt     int64             `json:"expired_at"` // 0 = never
	CreatedAt     int64             `json:"created_at"`
}

const (
	SubStatusPending   = 0
	SubStatusActive    = 1
	SubStatusExpired   = 2
	SubStatusCancelled = 3
)

const (
	PlanTypeGroup = "group"
	PlanTypeQuota = "quota"
	PlanTypeRPM   = "rpm"
	PlanTypeCombo = "combo"
)

// GetUserActiveRPM returns the highest RPM from active subscriptions for a user (0 = no subscription)
func GetUserActiveRPM(userId int) int {
	now := time.Now().Unix()
	var subs []UserSubscription
	common.DB.Preload("Plan").Where(
		"user_id = ? AND status = ? AND (expired_at = 0 OR expired_at > ?)",
		userId, SubStatusActive, now,
	).Find(&subs)

	maxRPM := 0
	for _, s := range subs {
		if s.Plan == nil {
			continue
		}
		if (s.Plan.Type == PlanTypeRPM || s.Plan.Type == PlanTypeCombo) && s.Plan.RPM > maxRPM {
			maxRPM = s.Plan.RPM
		}
	}
	return maxRPM
}

// ExpireSubscriptions checks and expires overdue subscriptions, reverting group changes
func ExpireSubscriptions() {
	now := time.Now().Unix()
	var subs []UserSubscription
	common.DB.Preload("Plan").Where(
		"status = ? AND expired_at > 0 AND expired_at <= ?", SubStatusActive, now,
	).Find(&subs)

	for _, sub := range subs {
		common.DB.Model(&UserSubscription{}).Where("id = ?", sub.Id).Update("status", SubStatusExpired)
		if sub.Plan != nil && (sub.Plan.Type == PlanTypeGroup || sub.Plan.Type == PlanTypeCombo) && sub.OriginalGroup != "" {
			common.DB.Model(&User{}).Where("id = ?", sub.UserId).Update("group", sub.OriginalGroup)
		}
	}
}
