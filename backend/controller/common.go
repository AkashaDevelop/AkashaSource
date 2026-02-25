package controller

import (
	"STfreApi/common"
	"STfreApi/model"
)

// RecordLog 记录日志
func RecordLog(userId int, logType int, content string) {
	log := model.Log{
		UserId:    userId,
		Type:      logType,
		Content:   content,
		CreatedAt: common.GetTimestamp(),
	}
	common.DB.Create(&log)
}
