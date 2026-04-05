package model

import (
	"STfreApi/common"
	"time"
)

type ChannelCheckinLog struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	ChannelId int    `json:"channel_id" gorm:"index"`
	Status    string `json:"status" gorm:"type:varchar(20)"` // success, failed, skipped
	Message   string `json:"message" gorm:"type:text"`
	Reward    string `json:"reward" gorm:"type:varchar(50)"`
	CreatedAt int64  `json:"created_at"`
}

func (ChannelCheckinLog) TableName() string {
	return "channel_checkin_logs"
}

type ChannelBalanceLog struct {
	Id        int     `json:"id" gorm:"primaryKey;autoIncrement"`
	ChannelId int     `json:"channel_id" gorm:"index"`
	Balance   float64 `json:"balance"`
	Used      float64 `json:"used"`
	Quota     float64 `json:"quota"`
	Message   string  `json:"message" gorm:"type:text"`
	CreatedAt int64   `json:"created_at"`
}

func (ChannelBalanceLog) TableName() string {
	return "channel_balance_logs"
}

func AddCheckinLog(channelId int, status, message, reward string) error {
	return AddCheckinLogWithTime(channelId, status, message, reward, time.Now().Unix())
}

func AddCheckinLogWithTime(channelId int, status, message, reward string, createdAt int64) error {
	log := &ChannelCheckinLog{
		ChannelId: channelId,
		Status:    status,
		Message:   message,
		Reward:    reward,
		CreatedAt: createdAt,
	}
	return common.DB.Create(log).Error
}

func AddBalanceLog(channelId int, balance, used, quota float64, message string) error {
	log := &ChannelBalanceLog{
		ChannelId: channelId,
		Balance:   balance,
		Used:      used,
		Quota:     quota,
		Message:   message,
		CreatedAt: time.Now().Unix(),
	}
	return common.DB.Create(log).Error
}

func GetCheckinLogs(channelId int, limit int) ([]ChannelCheckinLog, error) {
	var logs []ChannelCheckinLog
	query := common.DB.Model(&ChannelCheckinLog{}).Order("created_at DESC")
	if channelId > 0 {
		query = query.Where("channel_id = ?", channelId)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&logs).Error
	return logs, err
}

func GetBalanceLogs(channelId int, limit int) ([]ChannelBalanceLog, error) {
	var logs []ChannelBalanceLog
	query := common.DB.Model(&ChannelBalanceLog{}).Order("created_at DESC")
	if channelId > 0 {
		query = query.Where("channel_id = ?", channelId)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&logs).Error
	return logs, err
}
