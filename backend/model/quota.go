package model

import (
	"STfreApi/common"
	"errors"

	"gorm.io/gorm"
)

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

// TransferAffQuotaToQuota 将用户的邀请返利余额（aff_quota）转入可用额度（quota）。
// 用条件式原子 UPDATE（WHERE aff_quota >= ?）避免并发转账超支，兼容 SQLite/MySQL/PostgreSQL，
// 不依赖 SELECT ... FOR UPDATE（SQLite 不支持该语法）。
func TransferAffQuotaToQuota(userId int, quota int64) error {
	if quota <= 0 {
		return errors.New("转移额度必须大于 0")
	}
	result := common.DB.Model(&User{}).
		Where("id = ? AND aff_quota >= ?", userId, quota).
		Updates(map[string]interface{}{
			"aff_quota": gorm.Expr("aff_quota - ?", quota),
			"quota":     gorm.Expr("quota + ?", quota),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("邀请返利余额不足")
	}
	return nil
}
