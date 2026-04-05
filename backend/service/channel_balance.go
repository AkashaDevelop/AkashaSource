package service

import (
	"STfreApi/adapter"
	"STfreApi/common"
	"STfreApi/model"
	"log"
	"strings"
	"time"
)

func isTokenExpiredError(message string) bool {
	if message == "" {
		return false
	}
	text := strings.ToLower(message)
	return strings.Contains(text, "token") && strings.Contains(text, "expired") ||
		strings.Contains(text, "invalid token") ||
		strings.Contains(text, "access token") ||
		strings.Contains(text, "unauthorized") ||
		strings.Contains(text, "forbidden") ||
		strings.Contains(text, "未登录") ||
		strings.Contains(text, "登录过期")
}

func RefreshChannelBalance(channel *model.Channel) (*adapter.BalanceInfo, error) {
	if !adapter.SupportsAccountFeatures(channel.Type) {
		return nil, nil
	}

	if channel.AccessToken == "" {
		return nil, nil
	}

	adaptor := adapter.GetAccountAdaptor(channel.Type)
	if adaptor == nil {
		return nil, nil
	}

	var balanceInfo *adapter.BalanceInfo
	var err error

	if channel.PlatformUserId > 0 {
		balanceInfo, err = adaptor.GetBalance(channel.BaseURL, channel.AccessToken, channel.PlatformUserId)
	} else {
		balanceInfo, err = adaptor.GetBalance(channel.BaseURL, channel.AccessToken)
	}

	if err != nil {
		errMsg := err.Error()
		if isTokenExpiredError(errMsg) {
			newToken, loginErr := TryAutoRelogin(channel)
			if loginErr == nil && newToken != "" {
				if channel.PlatformUserId > 0 {
					balanceInfo, err = adaptor.GetBalance(channel.BaseURL, newToken, channel.PlatformUserId)
				} else {
					balanceInfo, err = adaptor.GetBalance(channel.BaseURL, newToken)
				}
			}
		}
		if err != nil {
			return nil, err
		}
	}

	if balanceInfo == nil {
		return nil, nil
	}

	now := time.Now().Unix()
	updates := map[string]interface{}{
		"balance":            balanceInfo.Balance,
		"balance_refresh_at": now,
		"status":             model.ChannelStatusActive,
	}

	common.DB.Model(channel).Updates(updates)

	model.AddBalanceLog(channel.Id, balanceInfo.Balance, balanceInfo.Used, balanceInfo.Quota, "余额刷新成功")

	return balanceInfo, nil
}

func RefreshAllChannelBalances() ([]map[string]interface{}, error) {
	var channels []model.Channel
	err := common.DB.Where("status = ? AND access_token != ''", model.ChannelStatusActive).Find(&channels).Error
	if err != nil {
		return nil, err
	}

	results := make([]map[string]interface{}, 0, len(channels))
	for _, channel := range channels {
		if !adapter.SupportsAccountFeatures(channel.Type) {
			continue
		}

		balanceInfo, err := RefreshChannelBalance(&channel)
		logEntry := map[string]interface{}{
			"channel_id":   channel.Id,
			"channel_name": channel.Name,
			"success":      false,
			"balance":      0.0,
			"message":      "",
		}

		if err != nil {
			logEntry["message"] = err.Error()
			log.Printf("[余额监控] 渠道 #%d (%s) 刷新失败: %v", channel.Id, channel.Name, err)
		} else if balanceInfo != nil {
			logEntry["success"] = true
			logEntry["balance"] = balanceInfo.Balance
			logEntry["message"] = "刷新成功"
			log.Printf("[余额监控] 渠道 #%d (%s) 余额: %.2f", channel.Id, channel.Name, balanceInfo.Balance)
		}

		results = append(results, logEntry)
	}

	return results, nil
}

func GetChannelBalanceStatus(channelId int) (map[string]interface{}, error) {
	var channel model.Channel
	err := common.DB.First(&channel, channelId).Error
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"channel_id":          channel.Id,
		"channel_name":        channel.Name,
		"balance":             channel.Balance,
		"last_balance_refresh": channel.BalanceRefreshAt,
		"last_checkin":        channel.LastCheckinAt,
		"checkin_enabled":     channel.CheckinEnabled == 1,
	}

	if channel.AccessToken != "" && adapter.SupportsAccountFeatures(channel.Type) {
		balanceInfo, err := RefreshChannelBalance(&channel)
		if err == nil && balanceInfo != nil {
			result["balance"] = balanceInfo.Balance
			result["used"] = balanceInfo.Used
			result["quota"] = balanceInfo.Quota
		}
	}

	return result, nil
}

func GetLowBalanceChannels(threshold float64) ([]model.Channel, error) {
	var channels []model.Channel
	err := common.DB.Where("balance < ? AND status = ?", threshold, model.ChannelStatusActive).Find(&channels).Error
	return channels, err
}
