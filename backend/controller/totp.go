package controller

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"STfreApi/common"
	"STfreApi/model"
	"log"

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
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}
	if user.TOTPEnabled {
		common.Fail(c, common.CodeParamError, "2FA 已启用")
		return
	}

	issuer := common.SystemName
	if issuer == "" {
		issuer = "Akasha"
	}
	accountName := user.Username
	if accountName == "" {
		accountName = user.Email
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
	})
	if err != nil {
		log.Printf("[TOTP] Generate failed: issuer=%q account=%q err=%v", issuer, accountName, err)
		common.Fail(c, common.CodeServerError, "生成密钥失败")
		return
	}

	// Generate and store backup codes alongside secret (not enabled yet)
	// 备份码入库前要哈希一下喵～不能明文存，不然数据库一泄露备份码就全裸奔了
	backupCodes, _ := generateBackupCodes(8)
	hashedCodes := make([]string, len(backupCodes))
	for i, code := range backupCodes {
		h, err := common.Password2Hash(code)
		if err != nil {
			common.Fail(c, common.CodeServerError, "生成备份码失败")
			return
		}
		hashedCodes[i] = h
	}
	backupJSON, _ := json.Marshal(hashedCodes)

	common.DB.Model(&user).Updates(map[string]interface{}{
		"totp_secret":  key.Secret(),
		"backup_codes": string(backupJSON),
	})

	common.OK(c, gin.H{
		"uri":          key.URL(),
		"secret":       key.Secret(),
		"backup_codes": backupCodes, // 明文只在这次响应里出现一次，用户务必保存好
	})
}

// TOTPEnable verifies a TOTP code and activates 2FA
func TOTPEnable(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}

	userId, _ := c.Get("id")
	var user model.User
	if err := common.DB.First(&user, userId).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}
	if user.TOTPSecret == "" {
		common.Fail(c, common.CodeParamError, "请先调用 setup 接口")
		return
	}

	if !totp.Validate(req.Code, user.TOTPSecret) {
		common.Fail(c, common.CodeParamError, "验证码错误")
		return
	}

	common.DB.Model(&user).Update("totp_enabled", true)

	common.OK(c, gin.H{"message": "2FA 已启用"})
}

// TOTPDisable disables 2FA (requires password + TOTP code)
func TOTPDisable(c *gin.Context) {
	var req struct {
		Password string `json:"password" binding:"required"`
		Code     string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}

	userId, _ := c.Get("id")
	var user model.User
	if err := common.DB.First(&user, userId).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}
	if !user.TOTPEnabled {
		common.Fail(c, common.CodeParamError, "2FA 未启用")
		return
	}
	if !common.ValidatePassword(req.Password, user.Password) {
		common.Fail(c, common.CodeParamError, "密码错误")
		return
	}
	if !totp.Validate(req.Code, user.TOTPSecret) {
		common.Fail(c, common.CodeParamError, "验证码错误")
		return
	}

	common.DB.Model(&user).Updates(map[string]interface{}{
		"totp_enabled":    false,
		"totp_secret":     "",
		"backup_codes":    "",
		"totp_fail_count": 0,
		"totp_locked_at":  0,
	})
	common.OK(c, gin.H{"message": "2FA 已禁用"})
}

// TOTPLogin verifies 2FA code during login (second step)
func TOTPLogin(c *gin.Context) {
	var req struct {
		UserId        int    `json:"user_id" binding:"required"`
		Code          string `json:"code" binding:"required"`
		PreAuthTicket string `json:"pre_auth_ticket" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}

	// 票据必须对应"刚刚密码校验通过"的那次登录，否则光靠 user_id + 验证码
	// 就能跳过密码硬撞 2FA，票据一次性消费（取出即删）防止被反复拿去撞库
	if !common.ConsumePreAuthTicket(req.PreAuthTicket, req.UserId) {
		common.Fail(c, common.CodeUnauthorized, "登录会话已过期，请重新登录")
		return
	}

	var user model.User
	if err := common.DB.First(&user, req.UserId).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}

	// Check lock
	if user.TOTPLockedAt > 0 {
		lockUntil := time.Unix(user.TOTPLockedAt, 0).Add(15 * time.Minute)
		if time.Now().Before(lockUntil) {
			common.Fail(c, common.CodeForbidden, "2FA 已锁定，请15分钟后重试")
			return
		}
		common.DB.Model(&user).Updates(map[string]interface{}{
			"totp_fail_count": 0, "totp_locked_at": 0,
		})
	}

	valid := totp.Validate(req.Code, user.TOTPSecret)

	// Try backup code if TOTP fails
	if !valid {
		var hashedCodes []string
		json.Unmarshal([]byte(user.BackupCodes), &hashedCodes)
		for i, h := range hashedCodes {
			if common.ValidatePassword(req.Code, h) {
				valid = true
				hashedCodes = append(hashedCodes[:i], hashedCodes[i+1:]...)
				newJSON, _ := json.Marshal(hashedCodes)
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
		common.Fail(c, common.CodeParamError, "验证码错误")
		return
	}

	// Reset fail count
	common.DB.Model(&user).Updates(map[string]interface{}{
		"totp_fail_count": 0, "totp_locked_at": 0,
	})

	// Generate JWT
	token, err := common.GenerateToken(user.Id, user.Username, user.Role)
	if err != nil {
		common.Fail(c, common.CodeServerError, "生成令牌失败")
		return
	}

	common.OK(c, gin.H{
		"token": token,
		"user": gin.H{
			"id": user.Id, "username": user.Username,
			"role": user.Role, "quota": user.Quota,
		},
	})
}
