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
	OptionKeyInvitationEnabled       = "invitation_enabled"        // 是否开启邀请码注册（开启后必须使用邀请码）
	OptionKeyRegisterEnabled         = "register_enabled"          // 是否开放注册（默认 true）
	OptionKeyPasswordLoginEnabled    = "password_login_enabled"    // 是否允许密码登录（默认 true）
	OptionKeyPasswordRegisterEnabled = "password_register_enabled" // 是否允许密码注册（默认 true）
	OptionKeyInvitationCost          = "invitation_cost"           // 生成邀请码的成本 (Quota)
	OptionKeyInvitationReward        = "invitation_reward"         // 邀请者奖励 (Quota)
	OptionKeyNewUserReward           = "new_user_reward"           // 新用户奖励 (Quota)

	// Pricing Options
	OptionKeyModelRatio      = "model_ratio"
	OptionKeyCompletionRatio = "completion_ratio"
	OptionKeyGroupRatio      = "group_ratio"
	OptionKeyModelPrice      = "model_price"
	// ～图像/音频/缓存倍率，配合模型定价管理页一起用～
	OptionKeyImageRatio           = "image_ratio"
	OptionKeyAudioRatio           = "audio_ratio"
	OptionKeyAudioCompletionRatio = "audio_completion_ratio"
	OptionKeyCacheRatio           = "cache_ratio"
	OptionKeyBillingPriority      = "billing_priority" // 余额+订阅双资金池优先级：subscription_first | wallet_first

	// ～分组架构对齐 new-api 的三件套：用户分组特殊倍率、特殊可用分组规则、auto 分组候选列表～
	OptionKeyGroupGroupRatio         = "group_group_ratio"
	OptionKeyGroupSpecialUsableGroup = "group_special_usable_group"
	OptionKeyAutoGroups              = "auto_groups"

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
	OptionKeyStripeSecretKey     = "stripe_secret_key"
	OptionKeyStripeWebhookSecret = "stripe_webhook_secret"
	OptionKeyStripeCurrency      = "stripe_currency"
	OptionKeyStripeSuccessUrl    = "stripe_success_url"
	OptionKeyStripeCancelUrl     = "stripe_cancel_url"

	// ～Creem 支付配置，另一家国际收款好伙伴～
	OptionKeyCreemApiKey        = "creem_api_key"
	OptionKeyCreemWebhookSecret = "creem_webhook_secret"
	OptionKeyCreemProductId     = "creem_product_id" // 已废弃，兼容旧配置保留
	OptionKeyCreemProducts      = "creem_products"   // JSON 数组，多产品目录
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
	OptionKeyCxSecEnabled = "cxsec_enabled"         // 是否启用（bool, 默认 false）
	OptionKeyCxSecPaths   = "cxsec_protected_paths" // 受保护路径，逗号分隔

	// 宸汐清源 - 上下文净化（默认禁用，由超管开启）
	OptionKeyQingyuanEnabled = "qingyuan_enabled" // "true" | "false"

	// ～通知配置：余额预警和渠道告警开关放这里哦～
	OptionKeyLowBalanceThreshold = "low_balance_threshold" // 用户余额预警阈值（quota 单位，默认 500000 ≈ $1）
	OptionKeyChannelAlertEnabled = "channel_alert_enabled" // 渠道异常告警开关

	// Model Rate Limiting
	OptionKeyModelRPM = "model_rpm" // JSON: {"gpt-4": 10, "gpt-3.5-turbo": 60}

	// Thinking-to-Content
	OptionKeyThinkingToContent = "thinking_to_content"

	// LinuxDO
	OptionKeyLinuxDOMinTrustLevel = "linuxdo_min_trust_level"

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

	// Captcha Providers - hCaptcha & Google reCAPTCHA
	OptionKeyHCaptchaSiteKey    = "hcaptcha_site_key"
	OptionKeyHCaptchaSecretKey  = "hcaptcha_secret_key"
	OptionKeyHCaptchaEnabled    = "hcaptcha_enabled"
	OptionKeyReCaptchaSiteKey   = "recaptcha_site_key"
	OptionKeyReCaptchaSecretKey = "recaptcha_secret_key"
	OptionKeyReCaptchaEnabled   = "recaptcha_enabled"
	OptionKeyReCaptchaVersion   = "recaptcha_version" // v2 or v3

	// 宸汐玄鉴 - 行为风控模块（仅超管可配置）
	OptionKeyXuanJianEnabled = "xuanjian_enabled" // "true" | "false"
	OptionKeyXuanJianPolicy  = "xuanjian_policy"  // JSON 策略配置

	// 实名认证扩展模块（仅超管可配置）
	OptionKeyRealnameEnabled               = "realname_enabled"   // "true" | "false"
	OptionKeyRealnameScenarios             = "realname_scenarios" // JSON: ["model_call","recharge","double_blind"]
	OptionKeyRealnameProvider              = "realname_provider"  // "aliyun" | "tencent"
	OptionKeyRealnameAliyunAccessKeyId     = "realname_aliyun_access_key_id"
	OptionKeyRealnameAliyunAccessKeySecret = "realname_aliyun_access_key_secret"
	OptionKeyRealnameAliyunRegion          = "realname_aliyun_region"   // 默认 cn-hangzhou
	OptionKeyRealnameAliyunSceneId         = "realname_aliyun_scene_id" // 认证场景ID（阿里云控制台创建）

	// 版本更新检查
	OptionKeyVersionCheckEnabled       = "version_check_enabled"        // "true" | "false"
	OptionKeyVersionCheckIntervalHours = "version_check_interval_hours" // 检查间隔（小时）
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
	common.DB.AutoMigrate(&RealnameAuth{})
	common.DB.AutoMigrate(&NotificationLimit{})
	common.DB.AutoMigrate(&QingyuanRule{})
	common.DB.AutoMigrate(&QingyuanRuleCategory{})
	common.DB.AutoMigrate(&Sanction{})
	common.DB.AutoMigrate(&AuditLog{})
	// ～这几张表之前只在 ApplyMigrations() 的一次性版本迁移里出现过，
	// 老数据库跑完那一次版本号就不会再重新迁移了，后续给它们加字段会跟 AuditLog 一样报错，
	// 挪到这里保证每次启动都跟 model 定义对齐喵～
	common.DB.AutoMigrate(&Group{})
	common.DB.AutoMigrate(&CheckIn{})
	common.DB.AutoMigrate(&ModelConfig{})
	common.DB.AutoMigrate(&ContextSanitizationPolicy{})
	common.DB.AutoMigrate(&ContextSanitizationPolicyRevision{})
	common.DB.AutoMigrate(&ContextSanitizationEvent{})
	common.DB.AutoMigrate(&CustomChannelConfig{})
	common.DB.AutoMigrate(&XuanJianEvent{})
	common.DB.AutoMigrate(&XuanJianRule{})
	common.DB.AutoMigrate(&Ability{})
	SeedQingyuanRules()

	// Load options from DB
	var options []Option
	common.DB.Find(&options)
	for _, option := range options {
		common.UpdateOptionMap(option.Key, option.Value)
	}

	// Initialize pricing if not present in DB
	if _, ok := common.OptionMap[OptionKeyModelRatio]; !ok {
		common.UpdatePricing("", "", "", "", "", "", "", "") // Load defaults
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

	// auto_groups 默认值：["default"]，管理员可以在设置里改成自己的候选分组顺序～
	if _, ok := common.OptionMap[OptionKeyAutoGroups]; !ok {
		agJSON := common.AutoGroups2JSONString()
		common.DB.FirstOrCreate(&Option{Key: OptionKeyAutoGroups, Value: agJSON}, Option{Key: OptionKeyAutoGroups})
		common.UpdateOptionMap(OptionKeyAutoGroups, agJSON)
	}

	// 登录注册开关默认值（首次启动写入 DB，管理员可在设置页修改）
	common.DB.FirstOrCreate(&Option{Key: OptionKeyRegisterEnabled, Value: "true"}, Option{Key: OptionKeyRegisterEnabled})
	common.DB.FirstOrCreate(&Option{Key: OptionKeyPasswordLoginEnabled, Value: "true"}, Option{Key: OptionKeyPasswordLoginEnabled})
	common.DB.FirstOrCreate(&Option{Key: OptionKeyPasswordRegisterEnabled, Value: "true"}, Option{Key: OptionKeyPasswordRegisterEnabled})
	if v, ok := common.OptionMap[OptionKeyRegisterEnabled]; !ok || v == "" {
		common.UpdateOptionMap(OptionKeyRegisterEnabled, "true")
	}
	if v, ok := common.OptionMap[OptionKeyPasswordLoginEnabled]; !ok || v == "" {
		common.UpdateOptionMap(OptionKeyPasswordLoginEnabled, "true")
	}
	if v, ok := common.OptionMap[OptionKeyPasswordRegisterEnabled]; !ok || v == "" {
		common.UpdateOptionMap(OptionKeyPasswordRegisterEnabled, "true")
	}

	// 版本更新检查默认值
	if v, ok := common.OptionMap[OptionKeyVersionCheckEnabled]; !ok || v == "" {
		common.DB.FirstOrCreate(&Option{Key: OptionKeyVersionCheckEnabled, Value: "true"}, Option{Key: OptionKeyVersionCheckEnabled})
		common.UpdateOptionMap(OptionKeyVersionCheckEnabled, "true")
	}
	if v, ok := common.OptionMap[OptionKeyVersionCheckIntervalHours]; !ok || v == "" {
		common.DB.FirstOrCreate(&Option{Key: OptionKeyVersionCheckIntervalHours, Value: "24"}, Option{Key: OptionKeyVersionCheckIntervalHours})
		common.UpdateOptionMap(OptionKeyVersionCheckIntervalHours, "24")
	}
}
