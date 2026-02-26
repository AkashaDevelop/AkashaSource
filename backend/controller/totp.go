package controller

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"STfreApi/common"
	"STfreApi/model"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
)

func generateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(99999999))
		if err != nil {
			return nil, err
		}
		codes[i] = fmt.Sprintf("%08d", n.Int64())
	}
	return codes, nil
}

// TOTPSetup generates a TOTP secret and QR URI for the user
func TOTPSetup(c *gin.Context) {
	userId, _ := c.Get("id")
	var user model.User
	if err := common.DB.First(&user, userId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if user.TOTPEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA 已启用"})
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      common.SystemName,
		AccountName: user.Username,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成密钥失败"})
		return
	}

	// Store secret temporarily (not enabled yet)
	common.DB.Model(&user).Update("totp_secret", key.Secret())

	backupCodes, _ := generateBackupCodes(8)

	c.JSON(http.StatusOK, gin.H{
		"secret":       key.Secret(),
		"url":          key.URL(),
		"backup_codes": backupCodes,
	})
}

// TOTPEnable verifies a TOTP code and activates 2FA
func TOTPEnable(c *gin.Context) {
	var req struct {
		Code        string   `json:"code" binding:"required"`
		BackupCodes []string `json:"backup_codes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId, _ := c.Get("id")
	var user model.User
	if err := common.DB.First(&user, userId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if user.TOTPSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先调用 setup 接口"})
		return
	}

	if !totp.Validate(req.Code, user.TOTPSecret) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误"})
		return
	}

	// Hash and store backup codes
	backupJSON, _ := json.Marshal(req.BackupCodes)
	common.DB.Model(&user).Updates(map[string]interface{}{
		"totp_enabled": true,
		"backup_codes": string(backupJSON),
	})

	c.JSON(http.StatusOK, gin.H{"message": "2FA 已启用"})
}

// TOTPDisable disables 2FA (requires password + TOTP code)
func TOTPDisable(c *gin.Context) {
	var req struct {
		Password string `json:"password" binding:"required"`
		Code     string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId, _ := c.Get("id")
	var user model.User
	if err := common.DB.First(&user, userId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if !user.TOTPEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA 未启用"})
		return
	}
	if !common.ValidatePassword(req.Password, user.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码错误"})
		return
	}
	if !totp.Validate(req.Code, user.TOTPSecret) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误"})
		return
	}

	common.DB.Model(&user).Updates(map[string]interface{}{
		"totp_enabled":  false,
		"totp_secret":   "",
		"backup_codes":  "",
		"totp_fail_count": 0,
		"totp_locked_at":  0,
	})
	c.JSON(http.StatusOK, gin.H{"message": "2FA 已禁用"})
}

// TOTPLogin verifies 2FA code during login (second step)
func TOTPLogin(c *gin.Context) {
	var req struct {
		UserId int    `json:"user_id" binding:"required"`
		Code   string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user model.User
	if err := common.DB.First(&user, req.UserId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// Check lock
	if user.TOTPLockedAt > 0 {
		lockUntil := time.Unix(user.TOTPLockedAt, 0).Add(15 * time.Minute)
		if time.Now().Before(lockUntil) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "2FA 已锁定，请15分钟后重试"})
			return
		}
		common.DB.Model(&user).Updates(map[string]interface{}{
			"totp_fail_count": 0, "totp_locked_at": 0,
		})
	}

	valid := totp.Validate(req.Code, user.TOTPSecret)

	// Try backup code if TOTP fails
	if !valid {
		var codes []string
		json.Unmarshal([]byte(user.BackupCodes), &codes)
		for i, bc := range codes {
			if bc == req.Code {
				valid = true
				codes = append(codes[:i], codes[i+1:]...)
				newJSON, _ := json.Marshal(codes)
				common.DB.Model(&user).Update("backup_codes", string(newJSON))
				break
			}
		}
	}

	if !valid {
		failCount := user.TOTPFailCount + 1
		updates := map[string]interface{}{"totp_fail_count": failCount}
		if failCount >= 5 {
			updates["totp_locked_at"] = time.Now().Unix()
		}
		common.DB.Model(&user).Updates(updates)
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误"})
		return
	}

	// Reset fail count
	common.DB.Model(&user).Updates(map[string]interface{}{
		"totp_fail_count": 0, "totp_locked_at": 0,
	})

	// Generate JWT
	token, err := common.GenerateToken(user.Id, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "登录成功",
		"token":   token,
		"user": gin.H{
			"id": user.Id, "username": user.Username,
			"role": user.Role, "quota": user.Quota,
		},
	})
}
