package service

import (
	"STfreApi/common"
	"STfreApi/model"
	"log"
	"time"
)

// minLogRetentionDays ～留存下限守卫在这里，任何人都别想把日志留存天数调得太短哦（网络安全法要求不少于6个月）～
const minLogRetentionDays = 180

// CleanupExpiredLogs ～定期把过期的日志和审计记录轻轻扫掉，给数据库腾腾地方～
func CleanupExpiredLogs() (int64, error) {
	days := readLogRetentionDays()
	cutoff := time.Now().AddDate(0, 0, -days).Unix()

	var total int64

	logResult := common.DB.Where("created_at < ?", cutoff).Delete(&model.Log{})
	if logResult.Error != nil {
		return total, logResult.Error
	}
	total += logResult.RowsAffected

	auditResult := common.DB.Where("created_at < ?", cutoff).Delete(&model.AuditLog{})
	if auditResult.Error != nil {
		return total, auditResult.Error
	}
	total += auditResult.RowsAffected

	log.Printf("[日志清理] 留存天数: %d, 清理条数: %d", days, total)
	return total, nil
}

// readLogRetentionDays ～偷偷看一眼管理员配的留存天数，配少了也不听话，强制按180天来～
func readLogRetentionDays() int {
	common.OptionLock.RLock()
	raw, ok := common.OptionMap[model.OptionKeyLogRetentionDays]
	common.OptionLock.RUnlock()

	if !ok || raw == "" {
		return minLogRetentionDays
	}

	days, err := parseInt(raw)
	if err != nil || days < minLogRetentionDays {
		return minLogRetentionDays
	}
	return days
}
