package model

import (
	"STfreApi/common"
	"fmt"
	"time"

	"gorm.io/gorm"
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
	// ～国际支付渠道下单订阅要用到的产品标识，留空代表该渠道不支持这个套餐喵～
	StripePriceId  string `json:"stripe_price_id" gorm:"type:varchar(128);default:''"`
	CreemProductId string `json:"creem_product_id" gorm:"type:varchar(128);default:''"`
	CreatedAt      int64  `json:"created_at"`
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

// ActivateSubscription ～把一份 pending 的订阅正式激活，CAS 保护防止并发重复执行～
// SQLite 不支持 SELECT ... FOR UPDATE（见 quota.go 里 TransferAffQuotaToQuota 的说明），
// 这里改用条件 UPDATE + RowsAffected 判断的写法，三种数据库都能用同一套逻辑喵～
func ActivateSubscription(tx *gorm.DB, subId int) error {
	result := tx.Model(&UserSubscription{}).
		Where("id = ? AND status = ?", subId, SubStatusPending).
		Update("status", SubStatusActive)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// 已经被并发的另一次调用抢先激活过了，或者状态本来就不是 pending，幂等跳过
		return nil
	}

	var sub UserSubscription
	if err := tx.Preload("Plan").First(&sub, subId).Error; err != nil {
		return err
	}
	plan := sub.Plan
	if plan == nil {
		return fmt.Errorf("套餐不存在")
	}

	now := time.Now().Unix()
	var expiredAt int64
	if plan.DurationDays > 0 {
		expiredAt = now + int64(plan.DurationDays)*86400
	}

	updates := map[string]interface{}{
		"started_at": now,
		"expired_at": expiredAt,
	}

	if plan.Type == PlanTypeGroup || plan.Type == PlanTypeCombo {
		if plan.GroupName != "" {
			var user User
			tx.Select("group").First(&user, sub.UserId)
			updates["original_group"] = user.Group
			if err := tx.Model(&User{}).Where("id = ?", sub.UserId).
				Update("group", plan.GroupName).Error; err != nil {
				return err
			}
		}
	}

	if plan.Type == PlanTypeQuota || plan.Type == PlanTypeCombo {
		if plan.Quota > 0 {
			// 订阅额度作为独立资金池记录在订阅上，不直接充值到用户钱包
			updates["quota"] = plan.Quota
		}
	}

	return tx.Model(&UserSubscription{}).Where("id = ?", subId).Updates(updates).Error
}
