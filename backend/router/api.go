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
	// Relay Route (OpenAI Compatible)
	router.POST("/v1/chat/completions", middleware.RateLimitMiddleware(), controller.Relay)
	router.POST("/v1/embeddings", middleware.RateLimitMiddleware(), controller.Relay)
	router.POST("/v1/audio/speech", middleware.RateLimitMiddleware(), controller.Relay)
	router.POST("/v1/audio/transcriptions", middleware.RateLimitMiddleware(), controller.Relay)
	router.POST("/v1/images/generations", middleware.RateLimitMiddleware(), controller.Relay)
	router.POST("/v1/completions", middleware.RateLimitMiddleware(), controller.Relay)
	router.POST("/v1/moderations", middleware.RateLimitMiddleware(), controller.Relay)
	router.POST("/v1/rerank", middleware.RateLimitMiddleware(), controller.Relay)
	router.GET("/v1/models", middleware.RateLimitMiddleware(), controller.ListModels)
	router.GET("/v1/models/:model", middleware.RateLimitMiddleware(), controller.RetrieveModel)
	router.GET("/v1/files", middleware.RateLimitMiddleware(), controller.FilesList)
	router.POST("/v1/files", middleware.RateLimitMiddleware(), controller.FilesCreate)
	router.GET("/v1/files/:id", middleware.RateLimitMiddleware(), controller.FilesRetrieve)
	router.GET("/v1/files/:id/content", middleware.RateLimitMiddleware(), controller.FilesContent)
	router.DELETE("/v1/files/:id", middleware.RateLimitMiddleware(), controller.FilesDelete)

	// Anthropic Messages API (Claude Code CLI compatible)
	router.POST("/v1/messages", middleware.RateLimitMiddleware(), controller.RelayMessages)

	// OpenAI Responses API (Codex CLI compatible)
	router.POST("/v1/responses", middleware.RateLimitMiddleware(), controller.RelayResponses)

	// Midjourney Relay
	router.POST("/mj/submit/imagine", middleware.RateLimitMiddleware(), midjourney.RelayMidjourney)
	router.POST("/mj/submit/action", middleware.RateLimitMiddleware(), midjourney.RelayMidjourney)
	router.POST("/mj/notify", midjourney.MidjourneyNotify)

	// Suno Relay
	router.POST("/suno/submit/*action", middleware.RateLimitMiddleware(), suno.RelaySuno)
	router.POST("/suno/notify", suno.SunoNotify)

	// OpenAI Realtime WebSocket
	router.GET("/v1/realtime", controller.RelayRealtime)

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
	router.POST("/api/payment/notify", controller.PaymentNotify)
	router.GET("/api/payment/notify", controller.PaymentNotify)

	apiRouter := router.Group("/api")
	{
		// 公开接口
		apiRouter.POST("/user/login", middleware.CriticalRateLimitMiddleware(), controller.UserLogin)
		apiRouter.POST("/user/register", middleware.CriticalRateLimitMiddleware(), controller.UserRegister)
		apiRouter.POST("/user/email/verify-code", middleware.CriticalRateLimitMiddleware(), controller.SendEmailVerifyCode)
		apiRouter.POST("/user/login/2fa", middleware.CriticalRateLimitMiddleware(), controller.TOTPLogin)
		apiRouter.POST("/user/password/reset-request", middleware.CriticalRateLimitMiddleware(), controller.PasswordResetRequest)
		apiRouter.POST("/user/password/reset-confirm", middleware.CriticalRateLimitMiddleware(), controller.PasswordResetConfirm)
		apiRouter.GET("/system/status", controller.IsSystemInitialized)
		apiRouter.GET("/pricing", controller.GetPublicPricing)

		// neko-api-key-tool compatible endpoint (self-auth via Bearer token)
		apiRouter.GET("/key/info", controller.GetKeyInfo)

		// 需要登录的接口
		authGroup := apiRouter.Group("/")
		authGroup.Use(middleware.AuthMiddleware())
		{
			// Token 管理 (用户/管理员)
			authGroup.GET("/token", controller.GetAllTokens)
			authGroup.POST("/token", controller.AddToken)
			authGroup.PUT("/token", controller.UpdateToken)
			authGroup.DELETE("/token/:id", controller.DeleteToken)

			// 日志查询 (用户)
			authGroup.GET("/log/self", controller.GetUserLogs)
			authGroup.GET("/log/self/stat", controller.GetUserLogStat)

			// 兑换码
			authGroup.POST("/user/redemption", controller.UseRedemptionCode)

			// 邀请码
			authGroup.POST("/user/invitation", controller.GenerateInvitationCode)
			authGroup.GET("/user/invitation", controller.GetUserInvitationCodes)

			// 用户自身信息
			authGroup.GET("/user/self", controller.GetSelf)
			authGroup.PUT("/user/self", controller.UpdateSelf)
			authGroup.GET("/user/dashboard", controller.GetUserDashboard)

			// 支付
			authGroup.POST("/payment/create", controller.CreatePayment)
			authGroup.GET("/payment/list", controller.ListPayments)

			// 2FA / TOTP
			authGroup.POST("/user/totp/setup", controller.TOTPSetup)
			authGroup.POST("/user/totp/enable", controller.TOTPEnable)
			authGroup.POST("/user/totp/disable", controller.TOTPDisable)

			// 签到
			authGroup.POST("/user/checkin", controller.CheckIn)
			authGroup.GET("/user/checkin", controller.GetCheckInStatus)

			// 文件管理
			authGroup.GET("/user/files", controller.UserFilesList)
			authGroup.DELETE("/user/files/:id", controller.UserFilesDelete)

			// 绘图/音乐任务
			authGroup.GET("/user/tasks/mj", controller.UserGetMJTasks)
			authGroup.GET("/user/tasks/suno", controller.UserGetSunoTasks)

			// 订阅
			authGroup.GET("/subscription/plans", controller.GetPublicSubscriptionPlans)
			authGroup.POST("/subscription/subscribe", controller.CreateSubscriptionOrder)
			authGroup.GET("/subscription/my", controller.GetMySubscriptions)
		}

		// 需要管理员权限的接口
		adminGroup := apiRouter.Group("/")
		adminGroup.Use(middleware.AuthMiddleware(), middleware.AdminAuthMiddleware())
		{
			// 仪表盘
			adminGroup.GET("/dashboard", controller.GetAdminDashboard)

			// 渠道管理 (仅管理员)
			adminGroup.GET("/channel", controller.GetAllChannels)
			adminGroup.GET("/channel/search", controller.SearchChannels)
			adminGroup.POST("/channel", controller.AddChannel)
			adminGroup.POST("/channel/batch", controller.AddChannels)
			adminGroup.PUT("/channel", controller.UpdateChannel)
			adminGroup.DELETE("/channel/:id", controller.DeleteChannel)
			adminGroup.GET("/channel/test/:id", controller.TestChannel)
			adminGroup.PATCH("/channel/:id/status", controller.ToggleChannelStatus)

			// 日志管理 (管理员)
			adminGroup.GET("/log", controller.GetAllLogs)
			adminGroup.GET("/log/stat", controller.GetLogStat)
			adminGroup.GET("/export/log", controller.ExportLogsCSV)

			// 系统设置
			adminGroup.GET("/option", controller.GetOptions)
			adminGroup.GET("/option/schema", controller.GetOptionSchema)
			adminGroup.PUT("/option", controller.UpdateOption)

			// 用户管理 (管理员)
			adminGroup.GET("/user", controller.GetAllUsers)
			adminGroup.GET("/user/search", controller.SearchUsers)
			adminGroup.POST("/user", controller.AddUser)
			adminGroup.PUT("/user", controller.UpdateUser)
			adminGroup.DELETE("/user/:id", controller.DeleteUser)
			adminGroup.PATCH("/user/:id/quota", controller.AdjustUserQuota)
			adminGroup.PATCH("/user/:id/status", controller.ToggleUserStatus)

			// 兑换码管理
			adminGroup.GET("/redemption", controller.GetAllRedemptions)
			adminGroup.POST("/redemption", controller.GenerateRedemptionCodes)
			adminGroup.PATCH("/redemption/:id/status", controller.UpdateRedemptionStatus)
			adminGroup.POST("/redemption/batch", controller.BatchRedemptionAction)
			adminGroup.GET("/export/redemption", controller.ExportRedemptionsCSV)

			// SQL 迁移
			adminGroup.GET("/migration/sql", controller.GetSQLMigrations)
			adminGroup.POST("/migration/sql/apply", controller.ApplySQLMigrations)
			adminGroup.POST("/migration/sql/rollback", controller.RollbackSQLMigrations)

			// 渠道增强
			adminGroup.POST("/channel/test-all", controller.TestAllChannels)
			adminGroup.GET("/channel/models/:id", controller.FetchChannelModels)
			adminGroup.GET("/channel/balance/:id", controller.FetchChannelBalance)
			adminGroup.GET("/export/channel", controller.ExportChannelsJSON)

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
			adminGroup.POST("/model/sync-pricing", controller.SyncPricingFromModelConfig)

			// 运维监控
			adminGroup.GET("/performance", controller.GetPerformance)
			adminGroup.DELETE("/log", controller.DeleteLogs)

			// 订阅套餐管理
			adminGroup.GET("/subscription/plan", controller.GetAllSubscriptionPlans)
			adminGroup.POST("/subscription/plan", controller.AddSubscriptionPlan)
			adminGroup.PUT("/subscription/plan", controller.UpdateSubscriptionPlan)
			adminGroup.DELETE("/subscription/plan/:id", controller.DeleteSubscriptionPlan)

			// 邀请码管理
			adminGroup.GET("/invitation", controller.AdminGetAllInvitations)
			adminGroup.POST("/invitation", controller.AdminGenerateInvitations)
			adminGroup.DELETE("/invitation/:id", controller.AdminDeleteInvitation)

			// 任务管理
			adminGroup.GET("/tasks/mj", controller.AdminGetMJTasks)
			adminGroup.GET("/tasks/suno", controller.AdminGetSunoTasks)
		}
	}
}
