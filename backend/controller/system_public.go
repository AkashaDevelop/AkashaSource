package controller

import (
	"STfreApi/common"
	oauthctl "STfreApi/controller/oauth"
	"STfreApi/model"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
)

type SetupInfo struct {
	Status       bool   `json:"status"`
	RootInit     bool   `json:"root_init"`
	DatabaseType string `json:"database_type"`
}

type SetupRequest struct {
	Username           string `json:"username"`
	Password           string `json:"password"`
	ConfirmPassword    string `json:"confirmPassword"`
	SelfUseModeEnabled bool   `json:"SelfUseModeEnabled"`
	DemoSiteEnabled    bool   `json:"DemoSiteEnabled"`
}

type UniversalVerifyRequest struct {
	Method string `json:"method"`
	Code   string `json:"code,omitempty"`
}

const secureVerifyTimeoutSec int64 = 300

var secureVerifyStore = struct {
	sync.Mutex
	M map[int]int64
}{M: map[int]int64{}}

func randomHex(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}

func setSecureVerified(userID int) int64 {
	now := time.Now().Unix()
	expiresAt := now + secureVerifyTimeoutSec
	secureVerifyStore.Lock()
	secureVerifyStore.M[userID] = expiresAt
	secureVerifyStore.Unlock()
	return expiresAt
}

func GetSetup(c *gin.Context) {
	var rootCount int64
	common.DB.Model(&model.User{}).Where("role = ?", model.RoleRoot).Count(&rootCount)

	setup := SetupInfo{
		Status:       rootCount > 0,
		RootInit:     rootCount > 0,
		DatabaseType: common.DBDriver,
	}
	if setup.DatabaseType == "" {
		setup.DatabaseType = "unknown"
	}
	common.OK(c, setup)
}

func PostSetup(c *gin.Context) {
	var req SetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "请求参数有误")
		return
	}

	var rootCount int64
	common.DB.Model(&model.User{}).Where("role = ?", model.RoleRoot).Count(&rootCount)
	if rootCount > 0 {
		common.Fail(c, common.CodeConflict, "系统已经初始化完成")
		return
	}

	if req.Username == "" || len(req.Username) > 12 {
		common.Fail(c, common.CodeParamError, "用户名长度必须在1-12字符")
		return
	}
	if req.Password != req.ConfirmPassword {
		common.Fail(c, common.CodeParamError, "两次输入的密码不一致")
		return
	}
	if len(req.Password) < 8 {
		common.Fail(c, common.CodeParamError, "密码长度至少为8个字符")
		return
	}

	hashed, err := common.Password2Hash(req.Password)
	if err != nil {
		common.Fail(c, common.CodeServerError, "密码加密失败")
		return
	}

	root := model.User{
		Username:    req.Username,
		Password:    hashed,
		Role:        model.RoleRoot,
		Status:      model.UserStatusActive,
		DisplayName: "Root User",
		Quota:       100000000,
	}
	if err = common.DB.Create(&root).Error; err != nil {
		common.Failf(c, common.CodeServerError, "创建管理员账号失败: %s", err.Error())
		return
	}

	_ = common.DB.Save(&model.Option{Key: "SelfUseModeEnabled", Value: strconv.FormatBool(req.SelfUseModeEnabled)}).Error
	_ = common.DB.Save(&model.Option{Key: "DemoSiteEnabled", Value: strconv.FormatBool(req.DemoSiteEnabled)}).Error
	common.UpdateOptionMap("SelfUseModeEnabled", strconv.FormatBool(req.SelfUseModeEnabled))
	common.UpdateOptionMap("DemoSiteEnabled", strconv.FormatBool(req.DemoSiteEnabled))

	common.OKMsg(c, "系统初始化成功", nil)
}

func GetStatus(c *gin.Context) {
	options := make(map[string]string)
	common.OptionLock.RLock()
	for _, k := range []string{
		"system_name", "logo_url", "footer_html", "notice", "about",
		"github_client_id", "linuxdo_client_id", "discord_client_id",
		"oidc_client_id", "telegram_bot_token", "wechat_app_id",
		"turnstile_check_enabled", "turnstile_site_key", "captcha_provider",
		"geetest_enabled", "geetest_id", "checkin_captcha",
		"enable_topup", "payment_provider", "min_topup", "price",
		"email_verification_enabled",
	} {
		options[k] = common.OptionMap[k]
	}
	common.OptionLock.RUnlock()

	var rootCount int64
	common.DB.Model(&model.User{}).Where("role = ?", model.RoleRoot).Count(&rootCount)

	common.OK(c, gin.H{
		"version":            common.Version,
		"start_time":         common.StartTime,
		"email_verification": common.EmailVerificationEnabled,
		"github_oauth":       common.GitHubClientId != "",
		"github_client_id":   common.GitHubClientId,
		"linuxdo_oauth":      common.LinuxDOClientId != "",
		"linuxdo_client_id":  common.LinuxDOClientId,
		"discord_oauth":      common.DiscordClientId != "",
		"discord_client_id":  common.DiscordClientId,
		"oidc_enabled":       common.OIDCClientId != "" && common.OIDCIssuerURL != "",
		"oidc_client_id":     common.OIDCClientId,
		"telegram_oauth":     common.TelegramBotToken != "",
		"wechat_login":       common.WechatAppId != "",
		"system_name":        common.SystemName,
		"turnstile_check":    common.TurnstileCheckEnabled,
		"turnstile_site_key": common.TurnstileSiteKey,
		"setup":              rootCount > 0,
		"_qn":                "stfreapi",
		"options":            options,
	})
}

func GetNotice(c *gin.Context) {
	common.OptionLock.RLock()
	notice := common.OptionMap[model.OptionKeyNotice]
	common.OptionLock.RUnlock()
	common.OK(c, notice)
}

func GetAbout(c *gin.Context) {
	common.OptionLock.RLock()
	about := common.OptionMap[model.OptionKeyAbout]
	common.OptionLock.RUnlock()
	common.OK(c, about)
}

func GetUserAgreement(c *gin.Context) {
	common.OptionLock.RLock()
	agreement := common.OptionMap["user_agreement"]
	common.OptionLock.RUnlock()
	common.OK(c, agreement)
}

func GetPrivacyPolicy(c *gin.Context) {
	common.OptionLock.RLock()
	policy := common.OptionMap["privacy_policy"]
	common.OptionLock.RUnlock()
	common.OK(c, policy)
}

func GetHomePageContent(c *gin.Context) {
	common.OptionLock.RLock()
	content := common.OptionMap["home_page_content"]
	common.OptionLock.RUnlock()
	common.OK(c, content)
}

func GenerateOAuthCode(c *gin.Context) {
	_ = c.Query("aff") // 当前后端 OAuth 兼容层先忽略该参数
	state := oauthctl.GenerateStateToken()
	common.OK(c, state)
}

func HandleOAuth(c *gin.Context) {
	provider := c.Param("provider")
	hasCallbackParams := c.Query("code") != "" || c.Query("state") != "" || c.Query("error") != ""

	switch provider {
	case "github":
		if hasCallbackParams {
			oauthctl.GitHubCallback(c)
		} else {
			oauthctl.GitHubLogin(c)
		}
	case "linuxdo":
		if hasCallbackParams {
			oauthctl.LinuxDOCallback(c)
		} else {
			oauthctl.LinuxDOLogin(c)
		}
	case "discord":
		if hasCallbackParams {
			oauthctl.DiscordCallback(c)
		} else {
			oauthctl.DiscordLogin(c)
		}
	case "oidc":
		if hasCallbackParams {
			oauthctl.OIDCCallback(c)
		} else {
			oauthctl.OIDCLogin(c)
		}
	default:
		common.Fail(c, common.CodeParamError, "不支持的 OAuth provider")
	}
}

func UniversalVerify(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}

	var req UniversalVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}

	var user model.User
	if err := common.DB.First(&user, userID).Error; err != nil {
		common.Fail(c, common.CodeUnauthorized, "用户不存在")
		return
	}
	if user.Status != model.UserStatusActive {
		common.Fail(c, common.CodeForbidden, "用户已被禁用")
		return
	}

	has2FA := user.TOTPEnabled && user.TOTPSecret != ""
	_, passkeyErr := model.GetPasskeyByUserID(userID)
	hasPasskey := passkeyErr == nil

	if !has2FA && !hasPasskey {
		common.Fail(c, common.CodeForbidden, "用户未启用2FA或Passkey")
		return
	}

	verified := false
	switch req.Method {
	case "2fa":
		if !has2FA {
			common.Fail(c, common.CodeParamError, "用户未启用2FA")
			return
		}
		if req.Code == "" {
			common.Fail(c, common.CodeParamError, "验证码不能为空")
			return
		}
		verified = totp.Validate(req.Code, user.TOTPSecret)
	case "passkey":
		if !hasPasskey {
			common.Fail(c, common.CodeParamError, "用户未启用Passkey")
			return
		}
		verified = true
	default:
		common.Fail(c, common.CodeParamError, "不支持的验证方式")
		return
	}

	if !verified {
		common.Fail(c, common.CodeForbidden, "验证失败")
		return
	}

	expiresAt := setSecureVerified(userID)
	common.OK(c, gin.H{"verified": true, "expires_at": expiresAt})
}
