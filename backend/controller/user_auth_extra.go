package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"STfreApi/common"
	oauthctl "STfreApi/controller/oauth"
	"STfreApi/model"
	"STfreApi/service"

	"github.com/gin-gonic/gin"
)

func EmailBind(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}

	emailAddr := strings.TrimSpace(c.Query("email"))
	code := strings.TrimSpace(c.Query("code"))
	if emailAddr == "" || code == "" {
		common.Fail(c, common.CodeParamError, "邮箱或验证码不能为空")
		return
	}
	if !CheckEmailVerifyCode(emailAddr, code) {
		common.Fail(c, common.CodeParamError, "验证码无效或已过期")
		return
	}

	// 与 new-api 保持“验证码通过即绑定”语义
	if err := common.DB.Model(&model.User{}).Where("id = ?", userID).Update("email", emailAddr).Error; err != nil {
		common.Fail(c, common.CodeServerError, "邮箱绑定失败")
		return
	}
	common.OKMsg(c, "邮箱绑定成功", nil)
}

// TelegramLogin 复用现有 Telegram OAuth 回调逻辑。
func TelegramLogin(c *gin.Context) {
	oauthctl.TelegramCallback(c)
}

// TelegramBind 对齐 new-api: 已登录用户绑定 Telegram。
func TelegramBind(c *gin.Context) {
	if common.TelegramBotToken == "" {
		common.Fail(c, common.CodeForbidden, "管理员未开启 Telegram 登录")
		return
	}

	userID := c.GetInt("id")
	if userID <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}

	if !checkTelegramAuthorization(c, common.TelegramBotToken) {
		common.Fail(c, common.CodeParamError, "无效的请求")
		return
	}

	telegramID := strings.TrimSpace(c.Query("id"))
	if telegramID == "" {
		common.Fail(c, common.CodeParamError, "缺少 Telegram ID")
		return
	}

	var count int64
	common.DB.Model(&model.User{}).
		Where("telegram_id = ? AND id <> ?", telegramID, userID).
		Count(&count)
	if count > 0 {
		common.Fail(c, common.CodeConflict, "该 Telegram 账户已被绑定")
		return
	}

	if err := common.DB.Model(&model.User{}).Where("id = ?", userID).Update("telegram_id", telegramID).Error; err != nil {
		common.Fail(c, common.CodeServerError, "绑定 Telegram 失败")
		return
	}

	common.OKMsg(c, "绑定成功", nil)
}

func checkTelegramAuthorization(c *gin.Context, token string) bool {
	hash := c.Query("hash")
	if hash == "" {
		return false
	}

	params := []string{}
	for key, values := range c.Request.URL.Query() {
		if key == "hash" || len(values) == 0 {
			continue
		}
		params = append(params, fmt.Sprintf("%s=%s", key, values[0]))
	}
	sort.Strings(params)
	dataCheckString := strings.Join(params, "\n")

	secretKey := sha256.Sum256([]byte(token))
	mac := hmac.New(sha256.New, secretKey[:])
	_, _ = mac.Write([]byte(dataCheckString))
	return hex.EncodeToString(mac.Sum(nil)) == hash
}

// GetUserGroups 对齐 new-api 的 /api/user/groups 与 /api/user/self/groups。
func GetUserGroups(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}

	var user model.User
	if err := common.DB.Select("id", "group", "extra_groups").First(&user, userID).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}

	userUsableGroups := service.GetUserUsableGroupsFull(user.Group, user.ExtraGroups)
	usable := make(map[string]map[string]interface{}, len(userUsableGroups))
	for name, desc := range userUsableGroups {
		if name == "auto" {
			usable[name] = map[string]interface{}{
				"ratio": "自动",
				"desc":  desc,
			}
			continue
		}
		usable[name] = map[string]interface{}{
			"ratio": service.GetUserGroupRatio(user.Group, name),
			"desc":  desc,
		}
	}

	common.OK(c, usable)
}

// GetChannelAffinityUsageCacheStats 读取真实渠道亲和缓存明细统计。
func GetChannelAffinityUsageCacheStats(c *gin.Context) {
	ruleName := strings.TrimSpace(c.Query("rule_name"))
	keyFP := strings.TrimSpace(c.Query("key_fp"))
	usingGroup := strings.TrimSpace(c.Query("using_group"))

	stats, err := getChannelAffinityUsageCacheStats(ruleName, usingGroup, keyFP)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	common.OK(c, stats)
}

// Logout 对齐 new-api 的 /api/user/logout：将当前 JWT 拉黑，使其在到期前立即失效。
func Logout(c *gin.Context) {
	tokenString, _ := c.Get("jwt_token")
	expiresAt, _ := c.Get("jwt_expires_at")
	if ts, ok := tokenString.(string); ok && ts != "" {
		if exp, ok := expiresAt.(time.Time); ok {
			common.BlacklistToken(ts, exp)
		}
	}
	common.OKMsg(c, "登出成功", nil)
}

// DeleteSelf 对齐 new-api 的 /api/user/self DELETE。
func DeleteSelf(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}
	if err := common.DB.Delete(&model.User{}, userID).Error; err != nil {
		common.Fail(c, common.CodeServerError, "注销失败")
		return
	}
	common.OKMsg(c, "账号已注销", nil)
}

// GenerateAccessToken 对齐 new-api 的 /api/user/token。
func GenerateAccessToken(c *gin.Context) {
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
	token, err := common.GenerateToken(user.Id, user.Username, user.Role)
	if err != nil {
		common.Fail(c, common.CodeServerError, "生成令牌失败")
		return
	}
	_ = common.DB.Model(&model.User{}).Where("id = ?", userID).Update("access_token", token).Error
	common.OK(c, gin.H{"access_token": token})
}

// GetUserModels 对齐 new-api 的 /api/user/models，复用现有模型聚合逻辑。
func GetUserModels(c *gin.Context) {
	ListModels(c)
}

// GetAffCode 对齐 new-api 的 /api/user/aff。
func GetAffCode(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}
	var user model.User
	if err := common.DB.Select("id", "aff_code").First(&user, userID).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}
	if strings.TrimSpace(user.AffCode) == "" {
		user.AffCode = fmt.Sprintf("aff_%d_%d", userID, time.Now().Unix()%100000)
		_ = common.DB.Model(&model.User{}).Where("id = ?", userID).Update("aff_code", user.AffCode).Error
	}
	common.OK(c, gin.H{"aff_code": user.AffCode})
}

// TransferAffQuota 将当前用户的邀请返利余额（aff_quota）转入可用额度（quota）。
func TransferAffQuota(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}

	var req struct {
		Quota int64 `json:"quota" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Quota <= 0 {
		common.Fail(c, common.CodeParamError, "转移额度必须大于 0")
		return
	}

	if err := model.TransferAffQuotaToQuota(userID, req.Quota); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	var user model.User
	if err := common.DB.Select("id", "username").First(&user, userID).Error; err == nil {
		_ = common.DB.Create(&model.Log{
			UserId: userID, Username: user.Username, CreatedAt: time.Now().Unix(),
			Type: model.LogTypeSystem, Content: "邀请返利转入可用额度", Quota: req.Quota, ModelName: "system",
		}).Error
	}

	common.OKMsg(c, "转账成功", nil)
}

// UpdateUserSetting 对齐 new-api 的 /api/user/setting，复用 UpdateSelf。
func UpdateUserSetting(c *gin.Context) {
	UpdateSelf(c)
}
