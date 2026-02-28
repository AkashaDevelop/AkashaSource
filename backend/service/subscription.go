package service

import (
	"STfreApi/model"
	"time"
)

func InitSubscriptionExpiry() {
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			model.ExpireSubscriptions()
		}
	}()
}
