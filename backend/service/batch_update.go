package service

import "STfreApi/common"

// BatchUpdateChannelStatus 批量更新渠道状态
func BatchUpdateChannelStatus(channelIds []int, status int) error {
	return common.DB.Table("channels").
		Where("id IN ?", channelIds).
		Update("status", status).Error
}

// BatchUpdateChannelPriority 批量更新渠道优先级
func BatchUpdateChannelPriority(channelIds []int, priority int) error {
	return common.DB.Table("channels").
		Where("id IN ?", channelIds).
		Update("priority", priority).Error
}

// BatchDeleteChannels 批量删除渠道
func BatchDeleteChannels(channelIds []int) error {
	return common.DB.Table("channels").
		Where("id IN ?", channelIds).
		Delete(nil).Error
}
