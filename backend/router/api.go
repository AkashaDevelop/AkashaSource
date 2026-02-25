package router

import (
	"STfreApi/controller"
	"STfreApi/controller/midjourney"
	"STfreApi/controller/oauth"
	"STfreApi/middleware"

	"github.com/gin-gonic/gin"
)

func SetApiRouter(router *gin.Engine) {
	// Relay Route (OpenAI Compatible)
	router.POST("/v1/chat/completions", middleware.RateLimitMiddleware(), controller.Relay)
	router.POST("/v1/embeddings", middleware.RateLimitMiddleware(), controller.Relay)

	// Midjourney Relay
	router.POST("/mj/submit/imagine", middleware.RateLimitMiddleware(), midjourney.RelayMidjourney)
	router.POST("/mj/submit/action", middleware.RateLimitMiddleware(), midjourney.RelayMidjourney)
	router.POST("/mj/notify", midjourney.MidjourneyNotify)

	// OAuth Routes
	router.GET("/oauth/github", oauth.GitHubLogin)
	router.GET("/oauth/github/callback", oauth.GitHubCallback)
	router.GET("/oauth/linuxdo", oauth.LinuxDOLogin)
	router.GET("/oauth/linuxdo/callback", oauth.LinuxDOCallback)

	apiRouter := router.Group("/api")
	{
		// 公开接口
		apiRouter.POST("/user/login", controller.UserLogin)
		apiRouter.POST("/user/register", controller.UserRegister)
		apiRouter.GET("/system/status", controller.IsSystemInitialized)

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

			// 兑换码
			authGroup.POST("/user/redemption", controller.UseRedemptionCode)

			// 邀请码
			authGroup.POST("/user/invitation", controller.GenerateInvitationCode)
			authGroup.GET("/user/invitation", controller.GetUserInvitationCodes)

			// 用户自身信息
			authGroup.GET("/user/self", controller.GetSelf)
			authGroup.PUT("/user/self", controller.UpdateSelf)
			authGroup.GET("/user/dashboard", controller.GetUserDashboard)
		}

		// 需要管理员权限的接口
		adminGroup := apiRouter.Group("/")
		adminGroup.Use(middleware.AuthMiddleware(), middleware.AdminAuthMiddleware())
		{
			// 仪表盘
			adminGroup.GET("/dashboard", controller.GetAdminDashboard)

			// 渠道管理 (仅管理员)
			adminGroup.GET("/channel", controller.GetAllChannels)
			adminGroup.POST("/channel", controller.AddChannel)
			adminGroup.POST("/channel/batch", controller.AddChannels)
			adminGroup.PUT("/channel", controller.UpdateChannel)
			adminGroup.DELETE("/channel/:id", controller.DeleteChannel)
			adminGroup.GET("/channel/test/:id", controller.TestChannel)

			// 日志管理 (管理员)
			adminGroup.GET("/log", controller.GetAllLogs)

			// 系统设置
			adminGroup.GET("/option", controller.GetOptions)
			adminGroup.PUT("/option", controller.UpdateOption)

			// 用户管理 (管理员)
			adminGroup.GET("/user", controller.GetAllUsers)
			adminGroup.POST("/user", controller.AddUser)
			adminGroup.PUT("/user", controller.UpdateUser)
			adminGroup.DELETE("/user/:id", controller.DeleteUser)

			// 兑换码管理
			adminGroup.GET("/redemption", controller.GetAllRedemptions)
			adminGroup.POST("/redemption", controller.GenerateRedemptionCodes)
		}
	}
}
