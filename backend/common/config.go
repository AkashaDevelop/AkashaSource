package common

import (
	"strconv"
	"strings"
	"sync"
)

var (
	SystemName              = "Akasha"
	RegisterEnabled         = true
	PasswordLoginEnabled    = true
	PasswordRegisterEnabled = true
	OptionMap               = make(map[string]string)
	OptionLock              sync.RWMutex
)

func UpdateOptionMap(key string, value string) {
	OptionLock.Lock()
	defer OptionLock.Unlock()
	OptionMap[key] = value
	if key == "system_name" {
		SystemName = value
	}
	if key == "register_enabled" {
		RegisterEnabled = (value == "true")
	}
	if key == "password_login_enabled" {
		PasswordLoginEnabled = (value == "true")
	}
	if key == "password_register_enabled" {
		PasswordRegisterEnabled = (value == "true")
	}

	if key == "github_client_id" {
		GitHubClientId = value
	}
	if key == "github_client_secret" {
		GitHubClientSecret = value
	}

	if key == "linuxdo_client_id" {
		LinuxDOClientId = value
	}
	if key == "linuxdo_client_secret" {
		LinuxDOClientSecret = value
	}
	if key == "linuxdo_min_trust_level" {
		LinuxDOMinTrustLevel, _ = strconv.Atoi(value)
	}

	if key == "smtp_server" {
		SMTPServer = value
	}
	if key == "smtp_port" {
		port, err := strconv.Atoi(value)
		if err == nil {
			SMTPPort = port
		}
	}
	if key == "smtp_account" {
		SMTPAccount = value
	}
	if key == "smtp_password" {
		SMTPPassword = value
	}
	if key == "smtp_from" {
		SMTPFrom = value
	}
	if key == "smtp_ssl_enabled" {
		SMTPSSLEnabled = (value == "true")
	}
	if key == "email_verification_enabled" {
		EmailVerificationEnabled = (value == "true")
	}

	if key == "turnstile_site_key" {
		TurnstileSiteKey = value
	}
	if key == "turnstile_secret_key" {
		TurnstileSecretKey = value
	}
	if key == "turnstile_check_enabled" {
		TurnstileCheckEnabled = (value == "true")
	}
	if key == "geetest_enabled" {
		GeeTestEnabled = (value == "true")
	}
	if key == "geetest_id" {
		GeeTestId = value
	}
	if key == "geetest_key" {
		GeeTestKey = value
	}
	if key == "captcha_provider" {
		CaptchaProvider = value
	}
	if key == "checkin_captcha" {
		CheckinCaptcha = (value == "true")
	}
	if key == "hcaptcha_site_key" {
		HCaptchaSiteKey = value
	}
	if key == "hcaptcha_secret_key" {
		HCaptchaSecretKey = value
	}
	if key == "hcaptcha_enabled" {
		HCaptchaEnabled = (value == "true")
	}
	if key == "recaptcha_site_key" {
		ReCaptchaSiteKey = value
	}
	if key == "recaptcha_secret_key" {
		ReCaptchaSecretKey = value
	}
	if key == "recaptcha_enabled" {
		ReCaptchaEnabled = (value == "true")
	}
	if key == "recaptcha_version" {
		ReCaptchaVersion = value
	}
	if key == "content_moderation_timeout" {
		timeout, err := strconv.Atoi(value)
		if err == nil && timeout > 0 {
			ContentModerationTimeout = timeout
		}
	}
	if key == "redis_addr" {
		RedisAddr = value
	}
	if key == "redis_password" {
		RedisPassword = value
	}
	if key == "redis_db" {
		db, err := strconv.Atoi(value)
		if err == nil {
			RedisDB = db
		}
	}
	if key == "epay_api_url" {
		EpayApiUrl = value
	}
	if key == "epay_pid" {
		EpayPid = value
	}
	if key == "epay_key" {
		EpayKey = value
	}
	if key == "epay_type" {
		EpayType = value
	}
	if key == "epay_notify_url" {
		EpayNotifyUrl = value
	}
	if key == "epay_return_url" {
		EpayReturnUrl = value
	}

	if strings.HasPrefix(key, "linuxdo_quota_level_") {
		levelStr := strings.TrimPrefix(key, "linuxdo_quota_level_")
		level, err := strconv.Atoi(levelStr)
		if err == nil && level >= 0 && level <= 5 {
			quota, err := strconv.ParseInt(value, 10, 64)
			if err == nil {
				LinuxDOLevelQuota[level] = quota
			}
		}
	}

	if key == "thinking_to_content" {
		ThinkingToContent = (value == "true")
	}

	if key == "discord_client_id" {
		DiscordClientId = value
	}
	if key == "discord_client_secret" {
		DiscordClientSecret = value
	}

	if key == "oidc_client_id" {
		OIDCClientId = value
	}
	if key == "oidc_client_secret" {
		OIDCClientSecret = value
	}
	if key == "oidc_issuer_url" {
		OIDCIssuerURL = value
	}

	if key == "telegram_bot_token" {
		TelegramBotToken = value
	}

	if key == "wechat_app_id" {
		WechatAppId = value
	}
	if key == "wechat_app_secret" {
		WechatAppSecret = value
	}

	if key == "email_domain_restriction_enabled" {
		EmailDomainRestrictionEnabled = (value == "true")
	}
	if key == "email_domain_whitelist" {
		EmailDomainWhitelist = value
	}

	// ～这四个 key 之一变了，就把倍率相关的内存态整体重新刷一遍喵；
	// image_ratio/audio_ratio/audio_completion_ratio/cache_ratio 以前这里传的是硬编码空字符串，
	// 导致每次改其他倍率都会把这四项悄悄重置回默认值，现在从 OptionMap 里正确带上～
	if key == "model_ratio" || key == "completion_ratio" || key == "group_ratio" || key == "model_price" ||
		key == "image_ratio" || key == "audio_ratio" || key == "audio_completion_ratio" || key == "cache_ratio" {
		UpdatePricing(OptionMap["model_ratio"], OptionMap["completion_ratio"], OptionMap["group_ratio"],
			OptionMap["image_ratio"], OptionMap["audio_ratio"], OptionMap["audio_completion_ratio"],
			OptionMap["cache_ratio"], OptionMap["model_price"])
	}

	if key == "group_group_ratio" {
		UpdateGroupGroupRatio(value)
	}
	if key == "group_special_usable_group" {
		UpdateGroupSpecialUsableGroup(value)
	}
	if key == "auto_groups" {
		UpdateAutoGroups(value)
	}

	if key == "cxsec_enabled" {
		CxSecEnabled = (value == "true")
	}
	if key == "qingyuan_enabled" {
		QingyuanEnabled = (value == "true")
	}
	if key == "cxsec_protected_paths" && value != "" {
		CxSecProtectedPaths = value
	}
}
