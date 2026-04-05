package service

import (
	"STfreApi/adapter"
	"STfreApi/common"
	"STfreApi/model"
	"log"
	"strings"
	"time"
)

func isAlreadyCheckedInMessage(message string) bool {
	if message == "" {
		return false
	}
	text := strings.ToLower(message)
	return strings.Contains(text, "already checked in") ||
		strings.Contains(text, "already signed") ||
		strings.Contains(text, "already sign in") ||
		strings.Contains(message, "今日已签到") ||
		strings.Contains(message, "今天已签到") ||
		strings.Contains(message, "已经签到") ||
		strings.Contains(message, "重复签到") ||
		strings.Contains(message, "签到过")
}

func isUnsupportedCheckinMessage(message string) bool {
	if message == "" {
		return false
	}
	text := strings.ToLower(message)
	return strings.Contains(text, "invalid url (post /api/user/checkin)") ||
		(strings.Contains(text, "http 404") && strings.Contains(text, "/api/user/checkin")) ||
		strings.Contains(text, "checkin endpoint not found") ||
		strings.Contains(text, "check-in is not supported") ||
		strings.Contains(text, "checkin is not supported") ||
		strings.Contains(text, "does not support checkin") ||
		strings.Contains(text, "not support checkin")
}

func CheckinChannel(channel *model.Channel) (*adapter.CheckinResult, error) {
	if !adapter.SupportsAccountFeatures(channel.Type) {
		return &adapter.CheckinResult{
			Success: false,
			Message: "该渠道类型不支持签到功能",
		}, nil
	}

	if channel.AccessToken == "" {
		return &adapter.CheckinResult{
			Success: false,
			Message: "未配置访问令牌",
		}, nil
	}

	adaptor := adapter.GetAccountAdaptor(channel.Type)
	if adaptor == nil {
		return &adapter.CheckinResult{
			Success: false,
			Message: "不支持的渠道类型",
		}, nil
	}

	var result *adapter.CheckinResult
	var err error

	if channel.PlatformUserId > 0 {
		result, err = adaptor.Checkin(channel.BaseURL, channel.AccessToken, channel.PlatformUserId)
	} else {
		result, err = adaptor.Checkin(channel.BaseURL, channel.AccessToken)
	}

	if err != nil {
		return &adapter.CheckinResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	if isAlreadyCheckedInMessage(result.Message) {
		result.Success = true
		result.Message = "今日已签到"
	}

	if isUnsupportedCheckinMessage(result.Message) {
		return &adapter.CheckinResult{
			Success: false,
			Message: "站点不支持签到接口",
		}, nil
	}

	return result, nil
}

func CheckinChannelWithLog(channel *model.Channel) (*adapter.CheckinResult, error) {
	result, err := CheckinChannel(channel)
	if err != nil {
		model.AddCheckinLog(channel.Id, "failed", err.Error(), "")
		return nil, err
	}

	status := "failed"
	if result.Success {
		status = "success"
	}

	model.AddCheckinLog(channel.Id, status, result.Message, result.Reward)

	if result.Success {
		now := time.Now().Unix()
		common.DB.Model(channel).Updates(map[string]interface{}{
			"last_checkin_at": now,
			"status":          model.ChannelStatusActive,
		})
	}

	return result, nil
}

func CheckinAllChannels() ([]map[string]interface{}, error) {
	var channels []model.Channel
	err := common.DB.Where("checkin_enabled = ? AND status = ?", 1, model.ChannelStatusActive).Find(&channels).Error
	if err != nil {
		return nil, err
	}

	results := make([]map[string]interface{}, 0, len(channels))
	for _, channel := range channels {
		result, err := CheckinChannelWithLog(&channel)
		logEntry := map[string]interface{}{
			"channel_id":   channel.Id,
			"channel_name": channel.Name,
			"success":      false,
			"message":      "",
			"reward":       "",
		}
		if err != nil {
			logEntry["message"] = err.Error()
			log.Printf("[渠道签到] 渠道 #%d (%s) 签到失败: %v", channel.Id, channel.Name, err)
		} else if result != nil {
			logEntry["success"] = result.Success
			logEntry["message"] = result.Message
			logEntry["reward"] = result.Reward
			if result.Success {
				log.Printf("[渠道签到] 渠道 #%d (%s) 签到成功: %s", channel.Id, channel.Name, result.Message)
			} else {
				log.Printf("[渠道签到] 渠道 #%d (%s) 签到失败: %s", channel.Id, channel.Name, result.Message)
			}
		}
		results = append(results, logEntry)
	}

	return results, nil
}

func TryAutoRelogin(channel *model.Channel) (string, error) {
	if channel.AccountUsername == "" || channel.AccountPassword == "" {
		return "", nil
	}

	adaptor := adapter.GetAccountAdaptor(channel.Type)
	if adaptor == nil {
		return "", nil
	}

	token, err := adaptor.Login(channel.BaseURL, channel.AccountUsername, channel.AccountPassword)
	if err != nil {
		return "", err
	}

	common.DB.Model(channel).Updates(map[string]interface{}{
		"access_token": token,
		"status":       model.ChannelStatusActive,
	})

	return token, nil
}
