package service

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
)

// ShouldDisableChannel 判断是否应该禁用渠道
func ShouldDisableChannel(err *dto.OpenAIError, statusCode int) bool {
	if statusCode == http.StatusUnauthorized {
		return true
	}
	// 402 Payment Required
	if statusCode == 402 {
		return true
	}

	if err != nil {
		codeStr := fmt.Sprintf("%v", err.Code) // Handle interface{}
		switch codeStr {
		case "invalid_api_key", "account_deactivated", "billing_not_active":
			return true
		}
		switch err.Type {
		case "insufficient_quota":
			return true
		}
		
		// Common error messages
		msg := strings.ToLower(err.Message)
		if strings.Contains(msg, "insufficient quota") || 
		   strings.Contains(msg, "quota exceeded") ||
		   strings.Contains(msg, "credit limit reached") {
			return true
		}
	}

	return false
}

// DisableChannel 禁用渠道
func DisableChannel(channelId int, reason string) error {
	var channel model.Channel
	if err := common.DB.First(&channel, channelId).Error; err != nil {
		return err
	}

	if channel.AutoBan == 0 {
		log.Printf("[AutoBan] Channel #%d (%s) encountered error but AutoBan is disabled. Reason: %s", channelId, channel.Name, reason)
		return nil
	}

	channel.Status = model.ChannelStatusAutoDisabled
	log.Printf("[AutoBan] Disabling channel #%d (%s) due to error: %s", channelId, channel.Name, reason)
	
	// 更新数据库
	// 注意：只更新 Status 字段，避免覆盖其他并发更新
	return common.DB.Model(&channel).Update("status", model.ChannelStatusAutoDisabled).Error
}
