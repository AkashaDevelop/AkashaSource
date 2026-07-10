package router

import (
	"STfreApi/controller"
	"STfreApi/controller/midjourney"
	"STfreApi/controller/oauth"
	"STfreApi/controller/suno"
	"STfreApi/middleware"

	"github.com/gin-gonic/gin"
)

func SetApiRouter(router *gin.Engine) {
	// ～先给所有请求盖上追踪印章，后面写日志的时候才能对上号～
	router.Use(middleware.RequestIDMiddleware())
	// ～数据库还没接通之前，先把需要查表的接口都挡在门外，只留初始化向导能走～
	router.Use(middleware.RequireDatabaseReady())
	// ↓↓↓ REMOVABLE：系统授权门禁，整体移除时删掉这一行 + middleware/system_license.go 即可 ↓↓↓
	router.Use(middleware.RequireSystemLicensed())
	// ↑↑↑ REMOVABLE ↑↑↑

	// Relay Route (OpenAI Compatible)
	router.POST("/v1/chat/completions", middleware.RateLimitMiddleware(), controller.Relay)
	router.POST("/v1/embeddings", middleware.RateLimitMiddleware(), controller.Relay)
	router.POST("/v1/audio/speech", middleware.RateLimitMiddleware(), controller.Relay)
	router.POST("/v1/audio/transcriptions", middleware.RateLimitMiddleware(), controller.Relay)
	router.POST("/v1/audio/translations", middleware.RateLimitMiddleware(), controller.Relay)
	router.POST("/v1/images/generations", middleware.RateLimitMiddleware(), controller.Relay)
	router.POST("/v1/images/edits", middleware.RateLimitMiddleware(), controller.Relay)
	router.POST("/v1/edits", middleware.RateLimitMiddleware(), controller.Relay)
	router.POST("/v1/completions", middleware.RateLimitMiddleware(), controller.Relay)
	router.POST("/v1/moderations", middleware.RateLimitMiddleware(), controller.Relay)
	router.POST("/v1/rerank", middleware.RateLimitMiddleware(), controller.Relay)
	router.GET("/v1/models", middleware.RateLimitMiddleware(), controller.ListModels)
	router.GET("/v1/models/:model", middleware.RateLimitMiddleware(), controller.RetrieveModel)
	router.GET("/v1beta/models", middleware.RateLimitMiddleware(), controller.ListModelsAuth)
	router.GET("/v1beta/openai/models", middleware.RateLimitMiddleware(), controller.ListModelsAuth)
	router.GET("/v1/files", middleware.RateLimitMiddleware(), controller.FilesList)
	router.POST("/v1/files", middleware.RateLimitMiddleware(), controller.FilesCreate)
	router.GET("/v1/files/:id", middleware.RateLimitMiddleware(), controller.FilesRetrieve)
	router.GET("/v1/files/:id/content", middleware.RateLimitMiddleware(), controller.FilesContent)
	router.DELETE("/v1/files/:id", middleware.RateLimitMiddleware(), controller.FilesDelete)

	// Video API compatibility (new-api)
	router.POST("/v1/video/generations", middleware.RateLimitMiddleware(), controller.RelayTask)
	router.GET("/v1/video/generations/:task_id", middleware.RateLimitMiddleware(), controller.RelayTaskFetch)
	router.POST("/v1/videos", middleware.RateLimitMiddleware(), controller.RelayTask)
	router.GET("/v1/videos/:task_id", middleware.RateLimitMiddleware(), controller.RelayTaskFetch)
	router.GET("/v1/videos/:task_id/content", middleware.RateLimitMiddleware(), controller.VideoProxy)
	router.POST("/v1/videos/:video_id/remix", middleware.RateLimitMiddleware(), controller.RelayTask)

	// Kling video compatibility
	router.POST("/kling/v1/videos/text2video", middleware.RateLimitMiddleware(), controller.RelayTask)
	router.POST("/kling/v1/videos/image2video", middleware.RateLimitMiddleware(), controller.RelayTask)
	router.GET("/kling/v1/videos/text2video/:task_id", middleware.RateLimitMiddleware(), controller.RelayTaskFetch)
	router.GET("/kling/v1/videos/image2video/:task_id", middleware.RateLimitMiddleware(), controller.RelayTaskFetch)

	// Jimeng official compatibility
	router.POST("/jimeng", middleware.RateLimitMiddleware(), controller.RelayTask)

	// Anthropic Messages API (Claude Code CLI compatible)
	router.POST("/v1/messages", middleware.RateLimitMiddleware(), controller.RelayMessages)

	// OpenAI Responses API (Codex CLI compatible)
	router.POST("/v1/responses", middleware.RateLimitMiddleware(), controller.RelayResponses)
	router.POST("/v1/responses/compact", middleware.RateLimitMiddleware(), controller.RelayResponsesCompact)

	// Midjourney Relay (new-api compatibility)
	router.POST("/mj/submit/imagine", middleware.RateLimitMiddleware(), midjourney.RelayMidjourney)
	router.POST("/mj/submit/action", middleware.RateLimitMiddleware(), midjourney.RelayMidjourney)
	router.POST("/mj/submit/shorten", middleware.RateLimitMiddleware(), midjourney.RelayMidjourney)
	router.POST("/mj/submit/modal", middleware.RateLimitMiddleware(), midjourney.RelayMidjourney)
	router.POST("/mj/submit/change", middleware.RateLimitMiddleware(), midjourney.RelayMidjourney)
	router.POST("/mj/submit/simple-change", middleware.RateLimitMiddleware(), midjourney.RelayMidjourney)
	router.POST("/mj/submit/describe", middleware.RateLimitMiddleware(), midjourney.RelayMidjourney)
	router.POST("/mj/submit/blend", middleware.RateLimitMiddleware(), midjourney.RelayMidjourney)
	router.POST("/mj/submit/edits", middleware.RateLimitMiddleware(), midjourney.RelayMidjourney)
	router.POST("/mj/submit/video", middleware.RateLimitMiddleware(), midjourney.RelayMidjourney)
	router.POST("/mj/submit/upload-discord-images", middleware.RateLimitMiddleware(), midjourney.RelayMidjourney)
	router.POST("/mj/insight-face/swap", middleware.RateLimitMiddleware(), midjourney.RelayMidjourney)
	router.POST("/mj/task/list-by-condition", middleware.RateLimitMiddleware(), midjourney.RelayMidjourney)
	router.GET("/mj/task/:id/fetch", middleware.RateLimitMiddleware(), midjourney.RelayMidjourneyFetch)
	router.GET("/mj/task/:id/image-seed", middleware.RateLimitMiddleware(), midjourney.RelayMidjourneyFetch)
	router.POST("/mj/notify", midjourney.MidjourneyNotify)

	// Suno Relay
	router.POST("/suno/submit/*action", middleware.RateLimitMiddleware(), suno.RelaySuno)
	router.POST("/suno/fetch", middleware.RateLimitMiddleware(), suno.RelaySunoFetch)
	router.GET("/suno/fetch/:id", middleware.RateLimitMiddleware(), suno.RelaySunoFetch)
	router.POST("/suno/notify", suno.SunoNotify)

	// OpenAI Realtime WebSocket
	router.GET("/v1/realtime", controller.RelayRealtime)

	// Playground compatibility
	router.POST("/pg/chat/completions", middleware.RateLimitMiddleware(), controller.Relay)

	// OpenAI billing compatibility
	router.GET("/dashboard/billing/subscription", middleware.RateLimitMiddleware(), controller.GetSubscription)
	router.GET("/v1/dashboard/billing/subscription", middleware.RateLimitMiddleware(), controller.GetSubscription)
	router.GET("/dashboard/billing/usage", middleware.RateLimitMiddleware(), controller.GetUsage)
	router.GET("/v1/dashboard/billing/usage", middleware.RateLimitMiddleware(), controller.GetUsage)

	// Gemini Native Format (:model::action is invalid in Gin; use wildcard and parse in controller)
	router.POST("/gemini/v1beta/models/*path", controller.RelayGeminiNative)

	// OAuth Routes
	router.GET("/oauth/github", oauth.GitHubLogin)
	router.GET("/oauth/github/callback", oauth.GitHubCallback)
	router.GET("/oauth/linuxdo", oauth.LinuxDOLogin)
	router.GET("/oauth/linuxdo/callback", oauth.LinuxDOCallback)
	router.GET("/oauth/discord", oauth.DiscordLogin)
	router.GET("/oauth/discord/callback", oauth.DiscordCallback)
	router.GET("/oauth/oidc", oauth.OIDCLogin)
	router.GET("/oauth/oidc/callback", oauth.OIDCCallback)
	router.GET("/oauth/telegram/callback", oauth.TelegramCallback)
	router.GET("/oauth/wechat", oauth.WechatLogin)
	router.GET("/oauth/wechat/callback", oauth.WechatCallback)
	// ↓↓↓ REMOVABLE：系统授权门禁的 Device Flow 路由，整体移除时删掉这块即可 ↓↓↓
	// Device Flow 无需回调地址，发起和轮询都在 rootGroup 下（需登录态）
	// ↑↑↑ REMOVABLE ↑↑↑
	router.POST("/api/payment/notify", controller.PaymentNotify)
	router.GET("/api/payment/notify", controller.PaymentNotify)

	apiRouter := router.Group("/api")
	{
		// ～宸汐御安全握手端点～
		// challenge 加全局限流（防止被刷导致内存挤满挑战条目）
		apiRouter.GET("/cx/challenge", middleware.RateLimitMiddleware(), controller.CxGetChallenge)
		// ks 加严格限流（握手成本高，比 challenge 更敏感）
		apiRouter.POST("/cx/ks", middleware.CriticalRateLimitMiddleware(), controller.CxHandshake)
		// 退出登录时主动销毁会话，切断密钥链
		apiRouter.DELETE("/cx/session", controller.CxDeleteSession)
		// 配置查询：前端启动时读取，永远不加密，永远可访问
		apiRouter.GET("/cx/config", controller.CxGetConfig)

		// 公开接口
		apiRouter.GET("/setup", controller.GetSetup)
		apiRouter.POST("/setup", controller.PostSetup)
		apiRouter.POST("/setup/database", controller.PostSetupDatabase)
		apiRouter.GET("/status", controller.GetStatus)
		apiRouter.GET("/notice", controller.GetNotice)
		apiRouter.GET("/about", controller.GetAbout)
		apiRouter.GET("/user-agreement", controller.GetUserAgreement)
		apiRouter.GET("/privacy-policy", controller.GetPrivacyPolicy)
		apiRouter.GET("/home_page_content", controller.GetHomePageContent)
		apiRouter.GET("/oauth/state", middleware.CriticalRateLimitMiddleware(), controller.GenerateOAuthCode)
		apiRouter.GET("/oauth/pending", middleware.CriticalRateLimitMiddleware(), oauth.CheckOAuthPending)
		apiRouter.POST("/oauth/complete", middleware.CriticalRateLimitMiddleware(), oauth.CompleteOAuthRegistration)
		apiRouter.GET("/oauth/:provider", middleware.CriticalRateLimitMiddleware(), controller.HandleOAuth)
		apiRouter.POST("/verify", middleware.AuthMiddleware(), middleware.CriticalRateLimitMiddleware(), controller.UniversalVerify)
		apiRouter.GET("/uptime/status", controller.GetUptimeStatus)
		apiRouter.GET("/status/test", middleware.AuthMiddleware(), middleware.AdminAuthMiddleware(), controller.TestStatus)
		apiRouter.GET("/verification", middleware.CriticalRateLimitMiddleware(), controller.SendEmailVerification)
		apiRouter.GET("/reset_password", middleware.CriticalRateLimitMiddleware(), controller.SendPasswordResetEmail)
		apiRouter.POST("/user/reset", middleware.CriticalRateLimitMiddleware(), controller.ResetPassword)
		apiRouter.GET("/ratio_config", middleware.CriticalRateLimitMiddleware(), controller.GetRatioConfig)
		apiRouter.GET("/oauth/email/bind", middleware.AuthMiddleware(), middleware.CriticalRateLimitMiddleware(), controller.EmailBind)
		apiRouter.GET("/oauth/telegram/login", middleware.CriticalRateLimitMiddleware(), controller.TelegramLogin)
		apiRouter.GET("/oauth/telegram/bind", middleware.AuthMiddleware(), middleware.CriticalRateLimitMiddleware(), controller.TelegramBind)
		apiRouter.POST("/stripe/webhook", controller.StripeWebhook)
		apiRouter.POST("/creem/webhook", controller.CreemWebhook)
		apiRouter.POST("/subscription/epay/notify", controller.SubscriptionEpayNotify)
		apiRouter.GET("/subscription/epay/notify", controller.SubscriptionEpayNotify)
		apiRouter.GET("/subscription/epay/return", controller.SubscriptionEpayReturn)
		apiRouter.POST("/subscription/epay/return", controller.SubscriptionEpayReturn)

		// ～宸汐御安全默认守护这五个敏感入口，CxSec 禁用时中间件自动放行～
		apiRouter.POST("/user/login", middleware.CriticalRateLimitMiddleware(), middleware.CxSecMiddleware(), controller.UserLogin)
		apiRouter.POST("/user/register", middleware.CriticalRateLimitMiddleware(), middleware.CxSecMiddleware(), controller.UserRegister)
		apiRouter.POST("/user/email/verify-code", middleware.CriticalRateLimitMiddleware(), controller.SendEmailVerifyCode)
		apiRouter.POST("/user/login/2fa", middleware.CriticalRateLimitMiddleware(), middleware.CxSecMiddleware(), controller.TOTPLogin)
		apiRouter.POST("/user/passkey/login/begin", middleware.CriticalRateLimitMiddleware(), controller.PasskeyLoginBegin)
		apiRouter.POST("/user/passkey/login/finish", middleware.CriticalRateLimitMiddleware(), controller.PasskeyLoginFinish)
		apiRouter.POST("/user/epay/notify", controller.PaymentNotify)
		apiRouter.GET("/user/epay/notify", controller.PaymentNotify)
		apiRouter.POST("/user/password/reset-request", middleware.CriticalRateLimitMiddleware(), middleware.CxSecMiddleware(), controller.PasswordResetRequest)
		apiRouter.POST("/user/password/reset-confirm", middleware.CriticalRateLimitMiddleware(), middleware.CxSecMiddleware(), controller.PasswordResetConfirm)
		apiRouter.GET("/system/status", controller.IsSystemInitialized)
		apiRouter.GET("/pricing", controller.GetPublicPricing)

		// neko-api-key-tool compatible endpoint (self-auth via Bearer token)
		apiRouter.GET("/key/info", controller.GetKeyInfo)
		apiRouter.GET("/usage/token", controller.GetTokenUsage)

		// 需要登录的接口
		authGroup := apiRouter.Group("/")
		authGroup.Use(middleware.AuthMiddleware())
		authGroup.Use(middleware.AuditLogMiddleware()) // ～普通用户的每一步写操作也要留下小脚印哦～
		{
			// Token 管理 (用户/管理员)
			authGroup.GET("/token", controller.GetAllTokens)
			authGroup.GET("/token/search", controller.SearchTokens)
			authGroup.GET("/token/:id", controller.GetToken)
			authGroup.POST("/token", controller.AddToken)
			authGroup.PUT("/token", controller.UpdateToken)
			authGroup.DELETE("/token/:id", controller.DeleteToken)
			authGroup.POST("/token/batch", controller.DeleteTokenBatch)

			// 日志查询 (用户)
			authGroup.GET("/log/self", controller.GetUserLogs)
			authGroup.GET("/log/self/search", controller.SearchUserLogs)
			authGroup.GET("/log/self/stat", controller.GetUserLogStat)

			// 兑换码
			authGroup.POST("/user/redemption", controller.UseRedemptionCode)

			// 邀请码
			authGroup.POST("/user/invitation", controller.GenerateInvitationCode)
			authGroup.GET("/user/invitation", controller.GetUserInvitationCodes)

			// 用户自身信息
			authGroup.GET("/user/self", controller.GetSelf)
			authGroup.PUT("/user/self", controller.UpdateSelf)
			authGroup.DELETE("/user/self", controller.DeleteSelf)
			authGroup.GET("/user/logout", controller.Logout)
			authGroup.GET("/user/models", controller.GetUserModels)
			authGroup.GET("/user/token", controller.GenerateAccessToken)
			authGroup.GET("/user/groups", controller.GetUserGroups)
			authGroup.GET("/user/self/groups", controller.GetUserGroups)
			authGroup.GET("/user/dashboard", controller.GetUserDashboard)
			authGroup.GET("/user/aff", controller.GetAffCode)
			authGroup.GET("/user/topup/info", controller.GetTopUpInfo)
			authGroup.GET("/user/topup/self", controller.GetUserTopUps)
			authGroup.POST("/user/topup", controller.TopUp)
			authGroup.POST("/user/pay", controller.RequestEpay)
			authGroup.POST("/user/amount", controller.RequestAmount)
			authGroup.POST("/user/stripe/pay", controller.RequestStripePay)
			authGroup.POST("/user/stripe/amount", controller.RequestStripeAmount)
			authGroup.POST("/user/creem/pay", controller.RequestCreemPay)
			authGroup.POST("/user/aff_transfer", controller.TransferAffQuota)
			authGroup.PUT("/user/setting", controller.UpdateUserSetting)
			authGroup.GET("/user/passkey", controller.PasskeyStatus)
			authGroup.POST("/user/passkey/register/begin", controller.PasskeyRegisterBegin)
			authGroup.POST("/user/passkey/register/finish", controller.PasskeyRegisterFinish)
			authGroup.POST("/user/passkey/verify/begin", controller.PasskeyVerifyBegin)
			authGroup.POST("/user/passkey/verify/finish", controller.PasskeyVerifyFinish)
			authGroup.DELETE("/user/passkey", controller.PasskeyDelete)

			// OAuth 绑定
			authGroup.GET("/user/oauth/bindings", controller.GetUserOAuthBindings)
			authGroup.DELETE("/user/oauth/bindings/:provider_id", controller.UnbindCustomOAuth)

			// 支付
			authGroup.POST("/payment/create", controller.CreatePayment)
			authGroup.GET("/payment/list", controller.ListPayments)
			authGroup.GET("/payment/history", controller.GetUserOrderHistory)

			// 2FA / TOTP
			authGroup.GET("/user/2fa/status", controller.Get2FAStatus)
			authGroup.POST("/user/2fa/setup", controller.Setup2FA)
			authGroup.POST("/user/2fa/enable", controller.Enable2FA)
			authGroup.POST("/user/2fa/disable", controller.Disable2FA)
			authGroup.POST("/user/2fa/backup_codes", controller.RegenerateBackupCodes)
			authGroup.POST("/user/totp/setup", controller.TOTPSetup)
			authGroup.POST("/user/totp/enable", controller.TOTPEnable)
			authGroup.POST("/user/totp/disable", controller.TOTPDisable)

			// ～签到接口也要守护哦，悄悄刷签到可不行～
			authGroup.GET("/user/checkin", controller.GetCheckInStatus)
			authGroup.POST("/user/checkin", middleware.CxSecMiddleware(), controller.CheckIn)

			// 文件管理
			authGroup.GET("/user/files", controller.UserFilesList)
			authGroup.DELETE("/user/files/:id", controller.UserFilesDelete)

			// 实名认证（用户）
			authGroup.GET("/user/realname/status", controller.GetRealnameStatus)
			authGroup.POST("/user/realname/init", controller.InitRealnameVerify)
			authGroup.POST("/user/realname/query", controller.QueryRealnameResult)

			// 绘图/音乐任务
			authGroup.GET("/user/tasks/mj", controller.UserGetMJTasks)
			authGroup.GET("/user/tasks/suno", controller.UserGetSunoTasks)
			authGroup.GET("/user/tasks", controller.GetUserTasks) // 统一任务接口

			// 订阅
			authGroup.GET("/subscription/plans", controller.GetPublicSubscriptionPlans)
			authGroup.POST("/subscription/subscribe", controller.CreateSubscriptionOrder)
			authGroup.GET("/subscription/my", controller.GetMySubscriptions)
			authGroup.GET("/subscription/self", controller.GetSubscriptionSelf)
			authGroup.PUT("/subscription/self/preference", controller.UpdateSubscriptionPreference)
			// 运营数据（new-api /data 兼容）
			authGroup.GET("/data/self", controller.GetUserQuotaDates)
		}

		// 需要管理员权限的接口
		adminGroup := apiRouter.Group("/")
		adminGroup.Use(middleware.AuthMiddleware(), middleware.AdminAuthMiddleware())
		adminGroup.Use(middleware.AuditLogMiddleware()) // ～管理员的每一步写操作都要留下小脚印哦～
		{
			// 仪表盘
			adminGroup.GET("/dashboard", controller.GetAdminDashboard)
			adminGroup.GET("/admin/system/monitor", controller.GetSystemMonitor)

			// 版本更新检查
			adminGroup.GET("/system/update-check", controller.GetUpdateInfo)
			adminGroup.POST("/system/update-check", controller.CheckUpdate)

			// 渠道管理 (仅管理员)
			adminGroup.GET("/channel", controller.GetAllChannels)
			adminGroup.GET("/channel/search", controller.SearchChannels)
			adminGroup.POST("/channel", controller.AddChannel)
			adminGroup.POST("/channel/batch", controller.AddChannels)
			adminGroup.PUT("/channel", controller.UpdateChannel)
			adminGroup.DELETE("/channel/:id", controller.DeleteChannel)
			adminGroup.POST("/channel/test/:id", controller.TestChannel)
			adminGroup.POST("/channel/test-model/:id", controller.TestChannelModel)
			adminGroup.PATCH("/channel/:id/status", controller.ToggleChannelStatus)

			// 渠道亲和性
			adminGroup.GET("/admin/channel-affinity/rules", controller.GetChannelAffinityRules)
			adminGroup.POST("/admin/channel-affinity/rules", controller.SaveChannelAffinityRules)
			adminGroup.GET("/admin/channel-affinity/stats", controller.GetChannelAffinityStats)
			adminGroup.POST("/admin/channel-affinity/cache/clear", controller.ClearChannelAffinityCacheAPI)

			// 上下文净化
			adminGroup.GET("/admin/sanitization/policies", controller.GetContextSanitizationPolicies)
			adminGroup.POST("/admin/sanitization/policies", controller.CreateContextSanitizationPolicy)
			adminGroup.PUT("/admin/sanitization/policies", controller.UpdateContextSanitizationPolicy)
			adminGroup.DELETE("/admin/sanitization/policies/:id", controller.DeleteContextSanitizationPolicy)
			adminGroup.GET("/admin/sanitization/policies/:id/revisions", controller.GetContextSanitizationPolicyRevisions)
			adminGroup.POST("/admin/sanitization/policies/:id/rollback", controller.RollbackContextSanitizationPolicy)
			adminGroup.POST("/admin/sanitization/reload", controller.ReloadContextSanitizationPolicyCache)
			adminGroup.GET("/admin/sanitization/events", controller.GetContextSanitizationEvents)
			adminGroup.GET("/admin/sanitization/stats", controller.GetContextSanitizationStats)

			// 批量渠道操作
			adminGroup.POST("/admin/channels/batch-status", controller.BatchUpdateChannelStatus)
			adminGroup.POST("/admin/channels/batch-priority", controller.BatchUpdateChannelPriority)
			adminGroup.POST("/admin/channels/batch-delete", controller.BatchDeleteChannels)

			// 自定义渠道配置管理
			adminGroup.GET("/custom-channel-config", controller.GetAllCustomConfigs)
			adminGroup.GET("/custom-channel-config/:id", controller.GetCustomConfig)
			adminGroup.POST("/custom-channel-config", controller.CreateCustomConfig)
			adminGroup.PUT("/custom-channel-config", controller.UpdateCustomConfig)
			adminGroup.DELETE("/custom-channel-config/:id", controller.DeleteCustomConfig)
			adminGroup.POST("/custom-channel-config/:id/test", controller.TestCustomConfig)

			// 日志管理 (管理员)
			adminGroup.GET("/log", controller.GetAllLogs)
			adminGroup.GET("/log/search", controller.SearchAllLogs)
			adminGroup.GET("/log/stat", controller.GetLogStat)
			adminGroup.GET("/log/channel_affinity_usage_cache", controller.GetChannelAffinityUsageCacheStats)
			adminGroup.GET("/export/log", controller.ExportLogsCSV)

			// 系统设置
			adminGroup.GET("/option/schema", controller.GetOptionSchema)

			// 用户管理 (管理员)
			adminGroup.GET("/user", controller.GetAllUsers)
			adminGroup.GET("/user/search", controller.SearchUsers)
			adminGroup.POST("/user", controller.AddUser)
			adminGroup.PUT("/user", controller.UpdateUser)
			adminGroup.DELETE("/user/:id", controller.DeleteUser)
			adminGroup.PATCH("/user/:id/quota", controller.AdjustUserQuota)
			adminGroup.PATCH("/user/:id/status", controller.ToggleUserStatus)
			adminGroup.GET("/user/:id/oauth/bindings", controller.GetUserOAuthBindingsByAdmin)
			adminGroup.DELETE("/user/:id/oauth/bindings/:provider_id", controller.UnbindCustomOAuthByAdmin)
			adminGroup.GET("/user/2fa/stats", controller.Admin2FAStats)
			adminGroup.DELETE("/user/:id/2fa", controller.AdminDisable2FA)
			adminGroup.DELETE("/user/:id/passkey", controller.AdminResetPasskey)

			// 实名认证管理（管理员）
			adminGroup.GET("/admin/realname/records", controller.AdminGetRealnameRecords)
			adminGroup.GET("/admin/realname/stats", controller.AdminGetRealnameStats)
			adminGroup.DELETE("/admin/realname/:user_id", controller.AdminResetRealname)

			// 兑换码管理
			adminGroup.GET("/redemption", controller.GetAllRedemptions)
			adminGroup.GET("/redemption/search", controller.SearchRedemptions)
			adminGroup.GET("/redemption/:id", controller.GetRedemption)
			adminGroup.POST("/redemption", controller.AddRedemption)
			adminGroup.PUT("/redemption", controller.UpdateRedemption)
			adminGroup.DELETE("/redemption/invalid", controller.DeleteInvalidRedemption)
			adminGroup.DELETE("/redemption/:id", controller.DeleteRedemption)
			adminGroup.PATCH("/redemption/:id/status", controller.UpdateRedemptionStatus)
			adminGroup.POST("/redemption/batch", controller.BatchRedemptionAction)
			adminGroup.GET("/export/redemption", controller.ExportRedemptionsCSV)

			// SQL 迁移（已移至 rootGroup，此处保留注释说明）

			// 渠道增强
			adminGroup.POST("/channel/test-all", controller.TestAllChannels)
			adminGroup.GET("/channel/models/:id", controller.FetchChannelModels)
			adminGroup.GET("/channel/balance/:id", controller.FetchChannelBalance)
			adminGroup.GET("/export/channel", controller.ExportChannelsJSON)
			adminGroup.POST("/channel/codex/oauth/start", controller.StartCodexOAuth)
			adminGroup.POST("/channel/codex/oauth/complete", controller.CompleteCodexOAuth)
			adminGroup.POST("/channel/:id/codex/oauth/start", controller.StartCodexOAuthForChannel)
			adminGroup.POST("/channel/:id/codex/oauth/complete", controller.CompleteCodexOAuthForChannel)
			adminGroup.POST("/channel/:id/codex/refresh", controller.RefreshCodexChannelCredential)
			adminGroup.GET("/channel/:id/codex/usage", controller.GetCodexChannelUsage)
			adminGroup.POST("/channel/ollama/pull", controller.OllamaPullModel)
			adminGroup.POST("/channel/ollama/pull/stream", controller.OllamaPullModelStream)
			adminGroup.DELETE("/channel/ollama/delete", controller.OllamaDeleteModel)
			adminGroup.GET("/channel/ollama/version/:id", controller.OllamaVersion)
			adminGroup.POST("/channel/batch/tag", controller.BatchSetChannelTag)
			adminGroup.GET("/channel/tag/models", controller.GetTagModels)
			adminGroup.POST("/channel/copy/:id", controller.CopyChannel)
			adminGroup.POST("/channel/multi_key/manage", controller.ManageMultiKeys)
			adminGroup.POST("/channel/fetch_models", controller.FetchModels)

			// 渠道管理（new-api /channel 兼容增强）
			adminGroup.GET("/channel/models", controller.ChannelListModels)
			adminGroup.GET("/channel/models_enabled", controller.EnabledListModels)
			adminGroup.GET("/channel/:id", controller.GetChannel)
			adminGroup.GET("/channel/update_balance", controller.UpdateAllChannelsBalance)
			adminGroup.GET("/channel/update_balance/:id", controller.UpdateChannelBalance)
			adminGroup.DELETE("/channel/disabled", controller.DeleteDisabledChannel)
			adminGroup.POST("/channel/tag/disabled", controller.DisableTagChannels)
			adminGroup.POST("/channel/tag/enabled", controller.EnableTagChannels)
			adminGroup.PUT("/channel/tag", controller.EditTagChannels)
			adminGroup.POST("/channel/fix", controller.FixChannelsAbilities)

			// 运营数据工具（new-api /data /prefill_group 兼容）
			adminGroup.GET("/data", controller.GetAllQuotaDates)
			adminGroup.GET("/prefill_group", controller.GetPrefillGroups)
			adminGroup.POST("/prefill_group", controller.CreatePrefillGroup)
			adminGroup.PUT("/prefill_group", controller.UpdatePrefillGroup)
			adminGroup.DELETE("/prefill_group/:id", controller.DeletePrefillGroup)

			// 分组管理
			adminGroup.GET("/group", controller.GetAllGroups)
			adminGroup.POST("/group", controller.AddGroup)
			adminGroup.PUT("/group", controller.UpdateGroup)
			adminGroup.DELETE("/group/:id", controller.DeleteGroup)

			// 模型配置
			adminGroup.GET("/model", controller.GetAllModelConfigs)
			adminGroup.POST("/model", controller.AddModelConfig)
			adminGroup.PUT("/model", controller.UpdateModelConfig)
			adminGroup.DELETE("/model/:id", controller.DeleteModelConfig)
			adminGroup.POST("/model/batch-ratio", controller.BatchUpdateModelRatio)
			adminGroup.POST("/model/batch-apply-pricing", controller.BatchApplyModelPricing)
			adminGroup.POST("/model/sync-pricing", controller.SyncPricingFromModelConfig)
			adminGroup.POST("/model/sync-upstream", controller.SyncUpstreamPricing)

			// 运维监控
			adminGroup.DELETE("/log", controller.DeleteLogs)

			// 订阅套餐管理
			adminGroup.GET("/subscription/plan", controller.GetAllSubscriptionPlans)
			adminGroup.POST("/subscription/plan", controller.AddSubscriptionPlan)
			adminGroup.PUT("/subscription/plan", controller.UpdateSubscriptionPlan)
			adminGroup.DELETE("/subscription/plan/:id", controller.DeleteSubscriptionPlan)

			// 订阅后台管理（new-api /subscription/admin 兼容）
			adminGroup.GET("/subscription/admin/plans", controller.AdminListSubscriptionPlans)
			adminGroup.POST("/subscription/admin/plans", controller.AdminCreateSubscriptionPlan)
			adminGroup.PUT("/subscription/admin/plans/:id", controller.AdminUpdateSubscriptionPlan)
			adminGroup.PATCH("/subscription/admin/plans/:id", controller.AdminUpdateSubscriptionPlanStatus)
			adminGroup.POST("/subscription/admin/bind", controller.AdminBindSubscription)
			adminGroup.GET("/subscription/admin/users/:id/subscriptions", controller.AdminListUserSubscriptions)
			adminGroup.POST("/subscription/admin/users/:id/subscriptions", controller.AdminCreateUserSubscription)
			adminGroup.POST("/subscription/admin/user_subscriptions/:id/invalidate", controller.AdminInvalidateUserSubscription)
			adminGroup.DELETE("/subscription/admin/user_subscriptions/:id", controller.AdminDeleteUserSubscription)

			// 邀请码管理
			adminGroup.GET("/invitation", controller.AdminGetAllInvitations)
			adminGroup.POST("/invitation", controller.AdminGenerateInvitations)
			adminGroup.DELETE("/invitation/:id", controller.AdminDeleteInvitation)

			// 任务管理
			adminGroup.GET("/tasks/mj", controller.AdminGetMJTasks)
			adminGroup.GET("/tasks/suno", controller.AdminGetSunoTasks)
			adminGroup.GET("/tasks", controller.GetAllTasks) // 统一任务接口

			// 渠道账号管理（签到、余额监控）
			adminGroup.POST("/channel/checkin/trigger", controller.TriggerChannelCheckin)
			adminGroup.POST("/channel/checkin/trigger/:id", controller.CheckinSingleChannel)
			adminGroup.POST("/channel/balance/refresh", controller.TriggerChannelBalanceRefresh)
			adminGroup.POST("/channel/balance/refresh/:id", controller.RefreshSingleChannelBalance)
			adminGroup.GET("/channel/checkin/logs", controller.GetChannelCheckinLogs)
			adminGroup.GET("/channel/balance/logs", controller.GetChannelBalanceLogs)
			adminGroup.GET("/channel/scheduler/status", controller.GetSchedulerStatus)
			adminGroup.PUT("/channel/scheduler/config", controller.UpdateSchedulerConfig)
			adminGroup.POST("/channel/scheduler/start", controller.StartScheduler)
			adminGroup.POST("/channel/scheduler/stop", controller.StopScheduler)
			adminGroup.PUT("/channel/account/:id", controller.UpdateChannelAccount)
			adminGroup.POST("/channel/login/:id", controller.TestChannelLogin)
			adminGroup.GET("/channel/balance/status/:id", controller.GetChannelBalanceStatus)
			// Vendors
			adminGroup.GET("/vendors", controller.GetAllVendors)
			adminGroup.GET("/vendors/search", controller.SearchVendors)
			adminGroup.GET("/vendors/:id", controller.GetVendorMeta)
			adminGroup.POST("/vendors", controller.CreateVendorMeta)
			adminGroup.PUT("/vendors", controller.UpdateVendorMeta)
			adminGroup.DELETE("/vendors/:id", controller.DeleteVendorMeta)

			// Models Meta
			adminGroup.GET("/models/sync_upstream/preview", controller.SyncUpstreamPreview)
			adminGroup.POST("/models/sync_upstream", controller.SyncUpstreamModels)
			adminGroup.GET("/models/missing", controller.GetMissingModels)
			adminGroup.GET("/models", controller.GetAllModelsMeta)
			adminGroup.GET("/models/search", controller.SearchModelsMeta)
			adminGroup.GET("/models/:id", controller.GetModelMeta)
			adminGroup.POST("/models", controller.CreateModelMeta)
			adminGroup.PUT("/models", controller.UpdateModelMeta)
			adminGroup.DELETE("/models/:id", controller.DeleteModelMeta)

			// Deployments
			adminGroup.GET("/deployments/settings", controller.GetModelDeploymentSettings)
			adminGroup.POST("/deployments/settings/test-connection", controller.TestIoNetConnection)
			adminGroup.GET("/deployments", controller.GetAllDeployments)
			adminGroup.GET("/deployments/search", controller.SearchDeployments)
			adminGroup.POST("/deployments/test-connection", controller.TestIoNetConnection)
			adminGroup.GET("/deployments/hardware-types", controller.GetHardwareTypes)
			adminGroup.GET("/deployments/locations", controller.GetLocations)
			adminGroup.GET("/deployments/available-replicas", controller.GetAvailableReplicas)
			adminGroup.POST("/deployments/price-estimation", controller.GetPriceEstimation)
			adminGroup.GET("/deployments/check-name", controller.CheckClusterNameAvailability)
			adminGroup.POST("/deployments", controller.CreateDeployment)
			adminGroup.GET("/deployments/:id", controller.GetDeployment)
			adminGroup.GET("/deployments/:id/logs", controller.GetDeploymentLogs)
			adminGroup.GET("/deployments/:id/containers", controller.ListDeploymentContainers)
			adminGroup.GET("/deployments/:id/containers/:container_id", controller.GetContainerDetails)
			adminGroup.PUT("/deployments/:id", controller.UpdateDeployment)
			adminGroup.PUT("/deployments/:id/name", controller.UpdateDeploymentName)
			adminGroup.POST("/deployments/:id/extend", controller.ExtendDeployment)
			adminGroup.DELETE("/deployments/:id", controller.DeleteDeployment)

			// 管理员支付管理（只读接口，普通管理员可查）
			adminGroup.GET("/admin/payment/orders", controller.AdminListOrders)
			adminGroup.GET("/admin/payment/stats", controller.AdminGetOrderStats)
		}

		// Root-only 接口（对齐 new-api RootAuth 语义）
		rootGroup := apiRouter.Group("/")
		rootGroup.Use(middleware.AuthMiddleware(), middleware.RootAuthMiddleware())
		rootGroup.Use(middleware.AuditLogMiddleware()) // ～超管的高危操作也要留下小脚印哦～
		{
			// 系统设置
			rootGroup.GET("/option", controller.GetOptions)
			rootGroup.PUT("/option", controller.UpdateOption)
			// ↓↓↓ REMOVABLE：系统授权门禁的登录态接口，整体移除时删掉这两行即可 ↓↓↓
			rootGroup.GET("/system-license/status", controller.GetLicenseStatus)
			rootGroup.POST("/system-license/unbind", controller.UnbindLicense)
			rootGroup.POST("/system-license/device-code", middleware.CriticalRateLimitMiddleware(), controller.StartDeviceFlow)
			rootGroup.POST("/system-license/poll", middleware.CriticalRateLimitMiddleware(), controller.PollDeviceFlow)
			// ↑↑↑ REMOVABLE ↑↑↑
			rootGroup.GET("/option/channel_affinity_cache", controller.GetChannelAffinityCacheStats)
			rootGroup.DELETE("/option/channel_affinity_cache", controller.ClearChannelAffinityCache)

			// 操作审计（仅超管可查，避免普通管理员翻看/干扰审计记录）
			rootGroup.GET("/admin/audit-log", controller.GetAuditLogs)

			// 补单接口权限收紧到超管，防止普通管理员免费给用户充值
			rootGroup.POST("/admin/payment/orders/:id/complete", controller.AdminManualCompleteOrder)

			// 自定义 OAuth 提供商管理
			rootGroup.POST("/custom-oauth-provider/discovery", controller.FetchCustomOAuthDiscovery)
			rootGroup.GET("/custom-oauth-provider", controller.GetCustomOAuthProviders)
			rootGroup.GET("/custom-oauth-provider/:id", controller.GetCustomOAuthProvider)
			rootGroup.POST("/custom-oauth-provider", controller.CreateCustomOAuthProvider)
			rootGroup.PUT("/custom-oauth-provider/:id", controller.UpdateCustomOAuthProvider)
			rootGroup.DELETE("/custom-oauth-provider/:id", controller.DeleteCustomOAuthProvider)

			// 渠道敏感操作
			rootGroup.POST("/channel/:id/key", controller.GetChannelKey)

			// 运营数据工具（new-api /ratio_sync 兼容）
			rootGroup.GET("/ratio_sync/channels", controller.GetSyncableChannels)
			rootGroup.POST("/ratio_sync/fetch", controller.FetchUpstreamRatios)

			// 运维监控
			rootGroup.GET("/performance", controller.GetPerformance)
			rootGroup.GET("/performance/stats", controller.GetPerformanceStats)
			rootGroup.DELETE("/performance/disk_cache", controller.ClearDiskCache)
			rootGroup.POST("/performance/reset_stats", controller.ResetPerformanceStats)
			rootGroup.POST("/performance/gc", controller.ForceGC)

			// 宸汐玄鉴 - 行为风控（仅超管可访问）
			rootGroup.GET("/admin/xuanjian/config", controller.GetXJConfig)
			rootGroup.PUT("/admin/xuanjian/config", controller.UpdateXJConfig)
			rootGroup.GET("/admin/xuanjian/events", controller.GetXJEvents)
			rootGroup.GET("/admin/xuanjian/profiles", controller.GetXJProfiles)
			rootGroup.POST("/admin/xuanjian/reset/:token_id", controller.PostXJResetProfile)
			rootGroup.GET("/admin/xuanjian/rules", controller.GetXJRules)
			rootGroup.POST("/admin/xuanjian/rules", controller.PostXJRule)
			rootGroup.PUT("/admin/xuanjian/rules/:id", controller.PutXJRule)
			rootGroup.DELETE("/admin/xuanjian/rules/:id", controller.DeleteXJRule)
			rootGroup.POST("/admin/xuanjian/rules/:id/toggle", controller.PostXJRuleToggle)
			rootGroup.POST("/admin/xuanjian/rules/:id/reset", controller.PostXJRuleReset)

			// 宸汐清源 - 规则库管理（仅超管可访问）
			rootGroup.GET("/admin/qingyuan/rules", controller.AdminListQingyuanRules)
			rootGroup.GET("/admin/qingyuan/rules/:id", controller.AdminGetQingyuanRule)
			rootGroup.POST("/admin/qingyuan/rules", controller.AdminCreateQingyuanRule)
			rootGroup.PUT("/admin/qingyuan/rules/:id", controller.AdminUpdateQingyuanRule)
			rootGroup.DELETE("/admin/qingyuan/rules/:id", controller.AdminDeleteQingyuanRule)
			rootGroup.POST("/admin/qingyuan/rules/:id/toggle", controller.AdminToggleQingyuanRule)
			rootGroup.POST("/admin/qingyuan/rules/reload", controller.AdminReloadQingyuanRules)
			rootGroup.POST("/admin/qingyuan/rules/test", controller.AdminTestQingyuanRule)
			rootGroup.POST("/admin/qingyuan/rules/import", controller.AdminBatchImportQingyuanRules)
			rootGroup.GET("/admin/qingyuan/rules/export", controller.AdminExportQingyuanRules)
			rootGroup.GET("/admin/qingyuan/categories", controller.AdminListQingyuanCategories)

			// 宸汐处置登记表（制裁管理，仅超管）
			rootGroup.GET("/admin/sanctions", controller.AdminListSanctions)
			rootGroup.POST("/admin/sanctions", controller.AdminCreateSanction)
			rootGroup.DELETE("/admin/sanctions/:id", controller.AdminRevokeSanction)
		}
	}
}
