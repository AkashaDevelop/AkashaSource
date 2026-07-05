package model

import "STfreApi/common"

type Option struct {
	Key   string `json:"key" gorm:"primaryKey"`
	Value string `json:"value"`
}

const (
	OptionKeySystemName                    = "system_name"
	OptionKeyLogoUrl                       = "logo_url"
	OptionKeyFooterHtml                    = "footer_html"
	OptionKeyNotice                        = "notice"
	OptionKeyAbout                         = "about"
	OptionKeySystemUrl                     = "system_url"
	OptionKeyPrice                         = "price"
	OptionKeyMinTopup                      = "min_topup"
	OptionKeyTopupLink                     = "topup_link"
	OptionKeyChatLink                      = "chat_link"
	OptionKeyChatLink2                     = "chat_link2"
	OptionKeyEmailDomainRestrictionEnabled = "email_domain_restriction_enabled"
	OptionKeyEmailDomainWhitelist          = "email_domain_whitelist"

	// Invitation Options
	OptionKeyInvitationEnabled = "invitation_enabled" // 是否开启邀请码注册（开启后必须使用邀请码）
	OptionKeyInvitationCost    = "invitation_cost"    // 生成邀请码的成本 (Quota)
	OptionKeyInvitationReward  = "invitation_reward"  // 邀请者奖励 (Quota)
	OptionKeyNewUserReward     = "new_user_reward"    // 新用户奖励 (Quota)

	// Pricing Options
	OptionKeyModelRatio      = "model_ratio"
	OptionKeyCompletionRatio = "completion_ratio"
	OptionKeyGroupRatio      = "group_ratio"
	OptionKeyModelPrice      = "model_price"
	OptionKeyBillingPriority  = "billing_priority" // 余额+订阅双资金池优先级：subscription_first | wallet_first

	OptionKeyContentModerationEnabled  = "content_moderation_enabled"
	OptionKeyContentModerationKeywords = "content_moderation_keywords"

	OptionKeyContentModerationTimeout         = "content_moderation_timeout"
	OptionKeyContentModerationWhitelistUsers  = "content_moderation_whitelist_users"
	OptionKeyContentModerationWhitelistModels = "content_moderation_whitelist_models"
	OptionKeyContentModerationWhitelistIPs    = "content_moderation_whitelist_ips"

	// ～内容审查只认腾讯云天御这一位真身，密钥小纸条放这里保管好～
	OptionKeyTencentModerationSecretId  = "tencent_moderation_secret_id"
	OptionKeyTencentModerationSecretKey = "tencent_moderation_secret_key"
	OptionKeyTencentModerationRegion    = "tencent_moderation_region"
	OptionKeyTencentModerationBizType   = "tencent_moderation_biz_type"

	// ～日志乖乖留一留，别急着删掉它们呀～
	OptionKeyLogRetentionDays = "log_retention_days"

	OptionKeyPaymentProvider     = "payment_provider"
	OptionKeyPaymentNotifySecret = "payment_notify_secret"
	OptionKeyEpayApiUrl          = "epay_api_url"
	OptionKeyEpayPid             = "epay_pid"
	OptionKeyEpayKey             = "epay_key"
	OptionKeyEpayType            = "epay_type"
	OptionKeyEpayNotifyUrl       = "epay_notify_url"
	OptionKeyEpayReturnUrl       = "epay_return_url"
	OptionKeyEnableTopup         = "enable_topup"

	// ～Stripe 支付配置，连上国际卡需要的一套小钥匙～
	OptionKeyStripeSecretKey      = "stripe_secret_key"
	OptionKeyStripeWebhookSecret  = "stripe_webhook_secret"
	OptionKeyStripeCurrency       = "stripe_currency"
	OptionKeyStripeSuccessUrl     = "stripe_success_url"
	OptionKeyStripeCancelUrl      = "stripe_cancel_url"

	// ～Creem 支付配置，另一家国际收款好伙伴～
	OptionKeyCreemApiKey        = "creem_api_key"
	OptionKeyCreemWebhookSecret = "creem_webhook_secret"
	OptionKeyCreemProductId     = "creem_product_id"  // 已废弃，兼容旧配置保留
	OptionKeyCreemProducts      = "creem_products"    // JSON 数组，多产品目录
	OptionKeyCreemSuccessUrl    = "creem_success_url"
	OptionKeyCreemTestMode      = "creem_test_mode"

	OptionKeyRedisAddr     = "redis_addr"
	OptionKeyRedisPassword = "redis_password"
	OptionKeyRedisDB       = "redis_db"

	// Check-in Options
	OptionKeyCheckinEnabled   = "checkin_enabled"
	OptionKeyCheckinMinReward = "checkin_min_reward"
	OptionKeyCheckinMaxReward = "checkin_max_reward"

	// ～宸汐御安全通讯协议配置，守护前端关键接口哦～
	OptionKeyCxSecEnabled = "cxsec_enabled"        // 是否启用（bool, 默认 false）
	OptionKeyCxSecPaths   = "cxsec_protected_paths" // 受保护路径，逗号分隔

	// 宸汐清源 - 上下文净化（默认禁用，由超管开启）
	OptionKeyQingyuanEnabled = "qingyuan_enabled" // "true" | "false"

	// ～通知配置：余额预警和渠道告警开关放这里哦～
	OptionKeyLowBalanceThreshold  = "low_balance_threshold"  // 用户余额预警阈值（quota 单位，默认 500000 ≈ $1）
	OptionKeyChannelAlertEnabled  = "channel_alert_enabled"  // 渠道异常告警开关

	// Model Rate Limiting
	OptionKeyModelRPM = "model_rpm" // JSON: {"gpt-4": 10, "gpt-3.5-turbo": 60}

	// Thinking-to-Content
	OptionKeyThinkingToContent = "thinking_to_content"

	// OAuth - Discord
	OptionKeyDiscordClientId     = "discord_client_id"
	OptionKeyDiscordClientSecret = "discord_client_secret"

	// OIDC
	OptionKeyOIDCClientId     = "oidc_client_id"
	OptionKeyOIDCClientSecret = "oidc_client_secret"
	OptionKeyOIDCIssuerURL    = "oidc_issuer_url"

	// Telegram
	OptionKeyTelegramBotToken = "telegram_bot_token"

	// WeChat Open Platform
	OptionKeyWechatAppId     = "wechat_app_id"
	OptionKeyWechatAppSecret = "wechat_app_secret"

	// Passkey
	OptionKeyPasskeyEnabled          = "passkey_enabled"
	OptionKeyPasskeyRPID             = "passkey_rp_id"
	OptionKeyPasskeyRPDisplayName    = "passkey_display_name"
	OptionKeyPasskeyOrigins          = "passkey_origins"
	OptionKeyPasskeyAllowInsecure    = "passkey_allow_insecure"
	OptionKeyPasskeyUserVerification = "passkey_user_verification"
	OptionKeyPasskeyAttachment       = "passkey_attachment"

	// 宸汐玄鉴 - 行为风控模块（仅超管可配置）
	OptionKeyXuanJianEnabled = "xuanjian_enabled" // "true" | "false"
	OptionKeyXuanJianPolicy  = "xuanjian_policy"  // JSON 策略配置
)

func InitOptions() {
	common.DB.AutoMigrate(&Option{})
	common.DB.AutoMigrate(&User{})
	common.DB.AutoMigrate(&Token{})
	common.DB.AutoMigrate(&Channel{})
	common.DB.AutoMigrate(&Log{})
	common.DB.AutoMigrate(&Redemption{})
	common.DB.AutoMigrate(&Invitation{})
	common.DB.AutoMigrate(&MidjourneyTask{})
	common.DB.AutoMigrate(&StoredFile{})
	common.DB.AutoMigrate(&PaymentOrder{})
	common.DB.AutoMigrate(&SunoTask{})
	common.DB.AutoMigrate(&SubscriptionPlan{})
	common.DB.AutoMigrate(&UserSubscription{})
	common.DB.AutoMigrate(&PasskeyCredential{})
	common.DB.AutoMigrate(&CustomOAuthProvider{})
	common.DB.AutoMigrate(&UserOAuthBinding{})
	common.DB.AutoMigrate(&Vendor{})
	common.DB.AutoMigrate(&ModelMeta{})
	common.DB.AutoMigrate(&Deployment{})
	common.DB.AutoMigrate(&PrefillGroup{})
	common.DB.AutoMigrate(&ChannelCheckinLog{})
	common.DB.AutoMigrate(&ChannelBalanceLog{})
	ApplyMigrations()
	_ = ApplySQLFileMigrations()

	// Load options from DB
	var options []Option
	common.DB.Find(&options)
	for _, option := range options {
		common.UpdateOptionMap(option.Key, option.Value)
	}

	// Initialize pricing if not present in DB
	if _, ok := common.OptionMap[OptionKeyModelRatio]; !ok {
		common.UpdatePricing("", "", "", "", "", "") // Load defaults
		// Save defaults to DB
		// Note: We should check if they exist before creating to avoid duplicates if Find failed but DB has them
		// But since we did Find above, if map is empty, DB is likely empty or key missing.
		// Use FirstOrCreate or similar

		mrJSON := common.ModelRatio2JSONString()
		crJSON := common.CompletionRatio2JSONString()
		grJSON := common.GroupRatio2JSONString()
		mpJSON := common.ModelPrice2JSONString()

		common.DB.FirstOrCreate(&Option{Key: OptionKeyModelRatio, Value: mrJSON}, Option{Key: OptionKeyModelRatio})
		common.DB.FirstOrCreate(&Option{Key: OptionKeyCompletionRatio, Value: crJSON}, Option{Key: OptionKeyCompletionRatio})
		common.DB.FirstOrCreate(&Option{Key: OptionKeyGroupRatio, Value: grJSON}, Option{Key: OptionKeyGroupRatio})
		common.DB.FirstOrCreate(&Option{Key: OptionKeyModelPrice, Value: mpJSON}, Option{Key: OptionKeyModelPrice})
	}

	// billing_priority 默认值：订阅优先
	if _, ok := common.OptionMap[OptionKeyBillingPriority]; !ok {
		common.DB.FirstOrCreate(&Option{Key: OptionKeyBillingPriority, Value: "subscription_first"}, Option{Key: OptionKeyBillingPriority})
		common.UpdateOptionMap(OptionKeyBillingPriority, "subscription_first")
	}
}
