package model

import (
	"time"
	"STfreApi/common"
)

type NotificationLimit struct {
	Id             int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId         int    `json:"user_id" gorm:"index:idx_user_notify"`
	NotifyType     string `json:"notify_type" gorm:"index:idx_user_notify"`
	LastNotifyTime int64  `json:"last_notify_time"`
	Count          int    `json:"count" gorm:"default:1"`
	CreatedAt      int64  `json:"created_at"`
}

func CheckNotificationLimit(userId int, notifyType string) (bool, error) {
	now := time.Now().Unix()
	var limit NotificationLimit
	err := common.DB.Where("user_id = ? AND notify_type = ?", userId, notifyType).First(&limit).Error

	if err != nil {
		limit = NotificationLimit{
			UserId:         userId,
			NotifyType:     notifyType,
			LastNotifyTime: now,
			Count:          1,
			CreatedAt:      now,
		}
		return true, common.DB.Create(&limit).Error
	}

	if now-limit.LastNotifyTime < 60 {
		if limit.Count >= 5 {
			return false, nil
		}
		limit.Count++
	} else {
		limit.Count = 1
		limit.LastNotifyTime = now
	}

	return true, common.DB.Save(&limit).Error
}
