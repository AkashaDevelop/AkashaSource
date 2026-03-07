package service

import (
	"time"
	"STfreApi/common"
	"STfreApi/model"
)

const (
	SubscriptionStatusActive  = 1
	SubscriptionStatusExpired = 3
	BatchSize                 = 300
)

func StartSubscriptionResetTask() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			resetExpiredSubscriptions()
			cleanupExpiredRecords()
		}
	}()
}

func resetExpiredSubscriptions() {
	now := time.Now().Unix()
	var subscriptions []model.UserSubscription

	err := common.DB.Where("status = ? AND expired_at <= ? AND expired_at > 0",
		SubscriptionStatusActive, now).
		Limit(BatchSize).
		Find(&subscriptions).Error

	if err != nil {
		return
	}

	for _, sub := range subscriptions {
		sub.Status = SubscriptionStatusExpired
		common.DB.Save(&sub)
	}
}

func cleanupExpiredRecords() {
	thirtyMinutesAgo := time.Now().Add(-30 * time.Minute).Unix()
	common.DB.Where("status = ? AND expired_at < ?",
		SubscriptionStatusExpired, thirtyMinutesAgo).
		Delete(&model.UserSubscription{})
}
