package controller

import (
	"encoding/json"
	"strings"

	"STfreApi/common"
	"STfreApi/model"

	"github.com/gin-gonic/gin"
)

// Get2FAStatus / Setup2FA / Enable2FA / Disable2FA / RegenerateBackupCodes 兼容层。
func Get2FAStatus(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}
	var user model.User
	if err := common.DB.Select("id", "totp_enabled", "backup_codes").First(&user, userID).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}
	var codes []string
	_ = json.Unmarshal([]byte(user.BackupCodes), &codes)
	common.OK(c, gin.H{"enabled": user.TOTPEnabled, "backup_codes_count": len(codes)})
}

func Setup2FA(c *gin.Context) {
	TOTPSetup(c)
}

func Enable2FA(c *gin.Context) {
	TOTPEnable(c)
}

func Disable2FA(c *gin.Context) {
	TOTPDisable(c)
}

func RegenerateBackupCodes(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}
	var user model.User
	if err := common.DB.First(&user, userID).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}
	if !user.TOTPEnabled {
		common.Fail(c, common.CodeParamError, "2FA 未启用")
		return
	}
	codes, err := generateBackupCodes(8)
	if err != nil {
		common.Fail(c, common.CodeServerError, "生成备份码失败")
		return
	}
	data, _ := json.Marshal(codes)
	if err = common.DB.Model(&model.User{}).Where("id = ?", userID).Update("backup_codes", string(data)).Error; err != nil {
		common.Fail(c, common.CodeServerError, "保存备份码失败")
		return
	}
	common.OK(c, gin.H{"backup_codes": codes})
}

// Admin2FAStats 对齐 new-api 的管理员 2FA 统计接口。
func Admin2FAStats(c *gin.Context) {
	var total int64
	var enabled int64

	if err := common.DB.Model(&model.User{}).Count(&total).Error; err != nil {
		common.Fail(c, common.CodeServerError, "统计失败")
		return
	}
	if err := common.DB.Model(&model.User{}).Where("totp_enabled = ?", true).Count(&enabled).Error; err != nil {
		common.Fail(c, common.CodeServerError, "统计失败")
		return
	}

	disabled := total - enabled
	if disabled < 0 {
		disabled = 0
	}

	common.OK(c, gin.H{
		"total_users":    total,
		"enabled_users":  enabled,
		"disabled_users": disabled,
	})
}

// AdminDisable2FA 对齐 new-api 的管理员禁用用户 2FA。
func AdminDisable2FA(c *gin.Context) {
	userID := strings.TrimSpace(c.Param("id"))
	if userID == "" {
		common.Fail(c, common.CodeParamError, "用户 ID 不能为空")
		return
	}

	updates := map[string]interface{}{
		"totp_enabled":    false,
		"totp_secret":     "",
		"backup_codes":    "",
		"totp_fail_count": 0,
		"totp_locked_at":  0,
	}
	result := common.DB.Model(&model.User{}).Where("id = ?", userID).Updates(updates)
	if result.Error != nil {
		common.Fail(c, common.CodeServerError, "禁用 2FA 失败")
		return
	}
	if result.RowsAffected == 0 {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}

	common.OKMsg(c, "已禁用该用户 2FA", nil)
}
