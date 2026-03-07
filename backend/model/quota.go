package model

import "STfreApi/common"

func UpdateUserUsedQuotaAndRequestCount(userId int, quota int) error {
	return common.DB.Model(&User{}).
		Where("id = ?", userId).
		UpdateColumn("used_quota", common.DB.Raw("used_quota + ?", quota)).
		UpdateColumn("request_count", common.DB.Raw("request_count + 1")).
		Error
}

func UpdateChannelUsedQuota(channelId int, quota int) error {
	return common.DB.Model(&Channel{}).
		Where("id = ?", channelId).
		UpdateColumn("used_quota", common.DB.Raw("used_quota + ?", quota)).
		Error
}
