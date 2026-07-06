package controller

import (
	"STfreApi/common"
	"STfreApi/model"

	"github.com/gin-gonic/gin"
)

func GetOptions(c *gin.Context) {
	var options []model.Option
	if err := common.DB.Find(&options).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取配置失败")
		return
	}
	common.OK(c, options)
}

func GetOptionSchema(c *gin.Context) {
	schema := []gin.H{
		{"key": model.OptionKeySystemName, "label": "系统名称", "type": "text", "group": "系统"},
		{"key": model.OptionKeySystemUrl, "label": "系统地址", "type": "text", "group": "系统"},
		{"key": model.OptionKeyLogoUrl, "label": "Logo 地址", "type": "text", "group": "系统"},
		{"key": model.OptionKeyFooterHtml, "label": "页脚内容", "type": "textarea", "group": "系统"},
		{"key": model.OptionKeyNotice, "label": "公告内容", "type": "textarea", "group": "系统"},
		{"key": model.OptionKeyThinkingToContent, "label": "Thinking 转 Content", "type": "boolean", "group": "系统"},
		{"key": model.OptionKeyPrice, "label": "默认价格", "type": "number", "group": "计费"},
		{"key": model.OptionKeyMinTopup, "label": "最低充值", "type": "number", "group": "计费"},
		{"key": model.OptionKeyModelRatio, "label": "模型倍率 (JSON)", "type": "textarea", "group": "计费"},
		{"key": model.OptionKeyCompletionRatio, "label": "补全倍率 (JSON)", "type": "textarea", "group": "计费"},
		{"key": model.OptionKeyModelRPM, "label": "模型速率限制 (JSON)", "type": "textarea", "group": "限流"},
		{"key": model.OptionKeyPaymentProvider, "label": "支付渠道 (epay/stripe/creem)", "type": "text", "group": "支付"},
		{"key": model.OptionKeyEpayApiUrl, "label": "易支付 API 地址", "type": "text", "group": "支付"},
		{"key": model.OptionKeyEpayPid, "label": "易支付 PID", "type": "text", "group": "支付"},
		{"key": model.OptionKeyEpayKey, "label": "易支付 KEY", "type": "password", "group": "支付"},
		{"key": model.OptionKeyEpayType, "label": "易支付通道类型", "type": "text", "group": "支付"},
		{"key": model.OptionKeyEpayNotifyUrl, "label": "易支付回调地址", "type": "text", "group": "支付"},
		{"key": model.OptionKeyEpayReturnUrl, "label": "易支付同步返回地址", "type": "text", "group": "支付"},
		{"key": model.OptionKeyStripeSecretKey, "label": "Stripe Secret Key", "type": "password", "group": "支付"},
		{"key": model.OptionKeyStripeWebhookSecret, "label": "Stripe Webhook Secret", "type": "password", "group": "支付"},
		{"key": model.OptionKeyStripeCurrency, "label": "Stripe 货币 (默认 usd)", "type": "text", "group": "支付"},
		{"key": model.OptionKeyStripeSuccessUrl, "label": "Stripe 支付成功跳转地址", "type": "text", "group": "支付"},
		{"key": model.OptionKeyStripeCancelUrl, "label": "Stripe 支付取消跳转地址", "type": "text", "group": "支付"},
		{"key": model.OptionKeyCreemApiKey, "label": "Creem API Key", "type": "password", "group": "支付"},
		{"key": model.OptionKeyCreemWebhookSecret, "label": "Creem Webhook Secret", "type": "password", "group": "支付"},
		{"key": model.OptionKeyCreemProductId, "label": "Creem Product ID", "type": "text", "group": "支付"},
		{"key": model.OptionKeyCreemSuccessUrl, "label": "Creem 支付成功跳转地址", "type": "text", "group": "支付"},
		{"key": model.OptionKeyCreemTestMode, "label": "Creem 测试模式", "type": "boolean", "group": "支付"},
		{"key": model.OptionKeyContentModerationEnabled, "label": "内容审查开关", "type": "boolean", "group": "风控"},
		{"key": model.OptionKeyContentModerationKeywords, "label": "敏感词", "type": "textarea", "group": "风控"},
		{"key": model.OptionKeyContentModerationTimeout, "label": "审查超时(秒)", "type": "number", "group": "风控"},
		{"key": model.OptionKeyContentModerationWhitelistUsers, "label": "审查白名单用户ID", "type": "textarea", "group": "风控"},
		{"key": model.OptionKeyContentModerationWhitelistModels, "label": "审查白名单模型", "type": "textarea", "group": "风控"},
		{"key": model.OptionKeyContentModerationWhitelistIPs, "label": "审查白名单IP", "type": "textarea", "group": "风控"},
		{"key": model.OptionKeyTencentModerationSecretId, "label": "腾讯云 SecretId", "type": "text", "group": "风控"},
		{"key": model.OptionKeyTencentModerationSecretKey, "label": "腾讯云 SecretKey", "type": "password", "group": "风控"},
		{"key": model.OptionKeyTencentModerationRegion, "label": "腾讯云地域（默认 ap-guangzhou）", "type": "text", "group": "风控"},
		{"key": model.OptionKeyTencentModerationBizType, "label": "腾讯云审核策略 BizType", "type": "text", "group": "风控"},
		{"key": model.OptionKeyLogRetentionDays, "label": "日志留存天数（≥180天，满足《网络安全法》要求）", "type": "number", "group": "日志"},
		{"key": model.OptionKeyRedisAddr, "label": "Redis 地址", "type": "text", "group": "缓存"},
		{"key": model.OptionKeyRedisPassword, "label": "Redis 密码", "type": "password", "group": "缓存"},
		{"key": model.OptionKeyRedisDB, "label": "Redis DB", "type": "number", "group": "缓存"},
		{"key": model.OptionKeyCheckinEnabled, "label": "签到开关", "type": "boolean", "group": "签到"},
		{"key": model.OptionKeyCheckinMinReward, "label": "签到最小奖励", "type": "number", "group": "签到"},
		{"key": model.OptionKeyCheckinMaxReward, "label": "签到最大奖励", "type": "number", "group": "签到"},
		{"key": "checkin_captcha", "label": "签到人机验证", "type": "boolean", "group": "签到"},
		{"key": model.OptionKeyLowBalanceThreshold, "label": "余额预警阈值（quota 单位，默认 500000 ≈ $1）", "type": "number", "group": "通知"},
		{"key": model.OptionKeyChannelAlertEnabled, "label": "渠道异常告警开关（需管理员配置通知方式）", "type": "boolean", "group": "通知"},
		{"key": model.OptionKeyInvitationEnabled, "label": "邀请码注册开关", "type": "boolean", "group": "邀请"},
		{"key": model.OptionKeyInvitationCost, "label": "生成邀请码成本", "type": "number", "group": "邀请"},
		{"key": model.OptionKeyInvitationReward, "label": "邀请者奖励", "type": "number", "group": "邀请"},
		{"key": model.OptionKeyNewUserReward, "label": "新用户奖励", "type": "number", "group": "邀请"},
		{"key": "github_client_id", "label": "GitHub Client ID", "type": "text", "group": "OAuth"},
		{"key": "github_client_secret", "label": "GitHub Client Secret", "type": "password", "group": "OAuth"},
		{"key": "linuxdo_client_id", "label": "LinuxDO Client ID", "type": "text", "group": "OAuth"},
		{"key": "linuxdo_client_secret", "label": "LinuxDO Client Secret", "type": "password", "group": "OAuth"},
		{"key": model.OptionKeyDiscordClientId, "label": "Discord Client ID", "type": "text", "group": "OAuth"},
		{"key": model.OptionKeyDiscordClientSecret, "label": "Discord Client Secret", "type": "password", "group": "OAuth"},
		{"key": model.OptionKeyOIDCClientId, "label": "OIDC Client ID", "type": "text", "group": "OAuth"},
		{"key": model.OptionKeyOIDCClientSecret, "label": "OIDC Client Secret", "type": "password", "group": "OAuth"},
		{"key": model.OptionKeyOIDCIssuerURL, "label": "OIDC Issuer URL", "type": "text", "group": "OAuth"},
		{"key": model.OptionKeyTelegramBotToken, "label": "Telegram Bot Token", "type": "password", "group": "OAuth"},
		{"key": "smtp_server", "label": "SMTP 服务器", "type": "text", "group": "邮件"},
		{"key": "smtp_port", "label": "SMTP 端口", "type": "number", "group": "邮件"},
		{"key": "smtp_account", "label": "SMTP 账号", "type": "text", "group": "邮件"},
		{"key": "smtp_password", "label": "SMTP 密码", "type": "password", "group": "邮件"},
		{"key": "smtp_from", "label": "发件人地址", "type": "text", "group": "邮件"},
		{"key": "turnstile_check_enabled", "label": "Turnstile 开关", "type": "boolean", "group": "安全"},
		{"key": "turnstile_site_key", "label": "Turnstile Site Key", "type": "text", "group": "安全"},
		{"key": "turnstile_secret_key", "label": "Turnstile Secret Key", "type": "password", "group": "安全"},
		{"key": "captcha_provider", "label": "验证码提供商", "type": "text", "group": "安全"},
		{"key": "geetest_enabled", "label": "极验开关", "type": "boolean", "group": "安全"},
		{"key": "geetest_id", "label": "极验 Captcha ID", "type": "text", "group": "安全"},
		{"key": "geetest_key", "label": "极验 Captcha Key", "type": "password", "group": "安全"},
		{"key": "hcaptcha_enabled", "label": "hCaptcha 开关", "type": "boolean", "group": "安全"},
		{"key": "hcaptcha_site_key", "label": "hCaptcha Site Key", "type": "text", "group": "安全"},
		{"key": "hcaptcha_secret_key", "label": "hCaptcha Secret Key", "type": "password", "group": "安全"},
		{"key": "recaptcha_enabled", "label": "reCAPTCHA 开关", "type": "boolean", "group": "安全"},
		{"key": "recaptcha_version", "label": "reCAPTCHA 版本", "type": "text", "group": "安全"},
		{"key": "recaptcha_site_key", "label": "reCAPTCHA Site Key", "type": "text", "group": "安全"},
		{"key": "recaptcha_secret_key", "label": "reCAPTCHA Secret Key", "type": "password", "group": "安全"},
	}
	values := map[string]string{}
	common.OptionLock.RLock()
	for _, item := range schema {
		key := item["key"].(string)
		values[key] = common.OptionMap[key]
	}
	common.OptionLock.RUnlock()
	common.OK(c, gin.H{"schema": schema, "values": values})
}

func UpdateOption(c *gin.Context) {
	var options []model.Option
	if err := c.ShouldBindJSON(&options); err != nil {
		common.Fail(c, common.CodeParamError, "参数解析失败")
		return
	}
	for _, option := range options {
		if err := common.DB.Save(&option).Error; err != nil {
			common.Failf(c, common.CodeServerError, "配置更新失败: %s", option.Key)
			return
		}
		common.UpdateOptionMap(option.Key, option.Value)
	}
	common.OKMsg(c, "配置保存成功", nil)
}

func IsSystemInitialized(c *gin.Context) {
	var count int64
	common.DB.Model(&model.User{}).Where("role >= ?", model.RoleAdmin).Count(&count)
	options := make(map[string]string)
	common.OptionLock.RLock()
	for _, k := range []string{
		"system_name", "logo_url", "footer_html", "notice", "about",
		"github_client_id", "linuxdo_client_id", "discord_client_id",
		"oidc_client_id", "telegram_bot_token", "wechat_app_id",
		"turnstile_check_enabled", "turnstile_site_key",
		"captcha_provider", "geetest_enabled", "geetest_id", "checkin_captcha",
		"hcaptcha_enabled", "hcaptcha_site_key",
		"recaptcha_enabled", "recaptcha_site_key", "recaptcha_version",
		"enable_topup", "payment_provider", "min_topup", "price",
		"email_verification_enabled",
		"register_enabled",
		"password_login_enabled",
		"password_register_enabled",
		"passkey_enabled",
		"quota_display_type", "quota_display_symbol", "quota_display_rate",
	} {
		options[k] = common.OptionMap[k]
	}
	common.OptionLock.RUnlock()
	common.OK(c, gin.H{"initialized": count > 0, "options": options})
}

func GetPublicPricing(c *gin.Context) {
	// Get model configs from database
	var configs []model.ModelConfig
	if err := common.DB.Where("enabled = ?", true).Find(&configs).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取模型配置失败")
		return
	}

	// Convert to array format
	type ModelPrice struct {
		Model               string  `json:"model"`
		InputRatio          float64 `json:"input_ratio"`
		OutputRatio         float64 `json:"output_ratio"`
		UpstreamInputPrice  float64 `json:"upstream_input_price"`
		UpstreamOutputPrice float64 `json:"upstream_output_price"`
		ActualInputPrice    float64 `json:"actual_input_price"`
		ActualOutputPrice   float64 `json:"actual_output_price"`
	}

	models := make([]ModelPrice, 0, len(configs))
	for _, cfg := range configs {
		models = append(models, ModelPrice{
			Model:               cfg.ModelName,
			InputRatio:          cfg.InputRatio,
			OutputRatio:         cfg.OutputRatio,
			UpstreamInputPrice:  cfg.UpstreamInputPrice,
			UpstreamOutputPrice: cfg.UpstreamOutputPrice,
			ActualInputPrice:    cfg.UpstreamInputPrice * cfg.InputRatio,
			ActualOutputPrice:   cfg.UpstreamOutputPrice * cfg.OutputRatio,
		})
	}

	common.OK(c, gin.H{"models": models})
}
