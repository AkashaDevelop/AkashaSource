package model

import (
	"STfreApi/common"
	"time"
)

// Sanction ～宸汐处置登记表：所有自动/手动处置的统一落库记录～
// 检测态（风险画像/限速计数）留在内存防侦察，但"处置态"必须持久化：
// 因为支持"永久"处置，重启后要保留；且管理员需要在前端查看/手动解除。
type Sanction struct {
	Id         int     `json:"id" gorm:"primaryKey;autoIncrement"`
	TargetType string  `json:"target_type" gorm:"type:varchar(16);index:idx_sanction_target"` // token / user / ip
	TargetKey  string  `json:"target_key" gorm:"type:varchar(64);index:idx_sanction_target"`  // tokenID / userID / IP 字符串
	Action     string  `json:"action" gorm:"type:varchar(32)"`                                // 见 Action 枚举
	Factor     float64 `json:"factor"`                                                        // throttle 降速倍率 / billing 计费倍率 / rpm_limit 的 RPM 值
	Reason     string  `json:"reason" gorm:"type:varchar(255)"`                               // 处置原因（命中的规则/finding）
	Source     string  `json:"source" gorm:"type:varchar(32)"`                                // xuanjian_auto / admin_manual
	Enabled    bool    `json:"enabled" gorm:"default:true"`                                   // 管理员可临时禁用而不删除
	ExpiresAt  int64   `json:"expires_at" gorm:"index"`                                       // 0 = 永久
	CreatedAt  int64   `json:"created_at"`
	UpdatedAt  int64   `json:"updated_at"`
}

// 处置目标类型
const (
	SanctionTargetToken = "token"
	SanctionTargetUser  = "user"
	SanctionTargetIP    = "ip"
)

// 处置动作枚举
const (
	SanctionThrottle       = "throttle"        // 高倍率限速（降 RPM），Factor=降速倍率 0~1
	SanctionRPMLimit       = "rpm_limit"       // 固定低 RPM 惩罚，Factor=目标 RPM 绝对值
	SanctionBillingPenalty = "billing_penalty" // 高倍率计费，Factor=计费倍率
	SanctionSuspendToken   = "suspend_token"   // 短暂停用 Token（自动到期恢复）
	SanctionDisableToken   = "disable_token"   // 停用 Token
	SanctionBanIP          = "ban_ip"          // 封禁来源 IP
	SanctionBanUser        = "ban_user"        // 封禁用户账号
)

func (Sanction) TableName() string {
	return "sanctions"
}

// GetActiveSanctions 拉取所有当前有效（启用且未过期）的制裁记录，供内存缓存全量加载
func GetActiveSanctions() ([]Sanction, error) {
	var list []Sanction
	now := time.Now().Unix()
	err := common.DB.Where("enabled = ? AND (expires_at = 0 OR expires_at > ?)", true, now).
		Find(&list).Error
	return list, err
}

// ListSanctions 管理员查询（可按 target_type 过滤，空则全部），含已禁用/已过期，倒序
func ListSanctions(targetType string) ([]Sanction, error) {
	var list []Sanction
	q := common.DB.Model(&Sanction{})
	if targetType != "" {
		q = q.Where("target_type = ?", targetType)
	}
	err := q.Order("id desc").Find(&list).Error
	return list, err
}

// UpsertSanction 按 (target_type, target_key, action) 唯一约束 upsert：
// 同一目标同一动作的重复处置视为"续期/覆盖"，而不是堆叠多条记录。
func UpsertSanction(s *Sanction) error {
	now := time.Now().Unix()
	var existing Sanction
	err := common.DB.Where("target_type = ? AND target_key = ? AND action = ?",
		s.TargetType, s.TargetKey, s.Action).First(&existing).Error
	if err == nil {
		// 已存在 → 更新 factor/reason/source/expires_at/enabled
		existing.Factor = s.Factor
		existing.Reason = s.Reason
		existing.Source = s.Source
		existing.Enabled = true
		existing.ExpiresAt = s.ExpiresAt
		existing.UpdatedAt = now
		return common.DB.Save(&existing).Error
	}
	s.CreatedAt = now
	s.UpdatedAt = now
	if !s.Enabled {
		s.Enabled = true
	}
	return common.DB.Create(s).Error
}

// DeleteSanction 物理删除一条制裁记录（管理员手动解除）
func DeleteSanction(id int) error {
	return common.DB.Delete(&Sanction{}, id).Error
}

// CleanupExpiredSanctions 清理已过期记录，防止表膨胀（后台定时调用）
func CleanupExpiredSanctions() error {
	now := time.Now().Unix()
	return common.DB.Where("expires_at > 0 AND expires_at <= ?", now).
		Delete(&Sanction{}).Error
}
