package service

import (
	"STfreApi/common"
	"STfreApi/model"
	"log"
	"time"
)

var logQueue chan model.Log

func InitLogQueue() {
	if logQueue != nil {
		return
	}
	logQueue = make(chan model.Log, 1000)
	go flushWorker()
}

func EnqueueLog(logItem model.Log) {
	if logQueue == nil {
		common.DB.Create(&logItem)
		return
	}
	select {
	case logQueue <- logItem:
	default:
		common.DB.Create(&logItem)
	}
}

func flushWorker() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	buffer := make([]model.Log, 0, 100)
	flush := func() {
		if len(buffer) == 0 {
			return
		}
		if err := common.DB.Create(&buffer).Error; err != nil {
			log.Printf("批量写入日志失败: %v", err)
		}
		buffer = buffer[:0]
	}

	for {
		select {
		case item := <-logQueue:
			buffer = append(buffer, item)
			if len(buffer) >= 100 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}
