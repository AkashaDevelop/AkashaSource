package middleware

import (
	"STfreApi/common"
	"STfreApi/model"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 验证 JWT Token
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, common.R{Code: common.CodeUnauthorized, Msg: "未提供认证令牌"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, common.R{Code: common.CodeUnauthorized, Msg: "认证令牌格式错误"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 复用 common.ParseToken，避免在中间件重复定义 Claims 与解析逻辑
		claims, err := common.ParseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, common.R{Code: common.CodeUnauthorized, Msg: "无效的认证令牌"})
			c.Abort()
			return
		}

		if common.IsTokenBlacklisted(tokenString) {
			c.JSON(http.StatusUnauthorized, common.R{Code: common.CodeUnauthorized, Msg: "令牌已失效，请重新登录"})
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("id", claims.UserId)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("jwt_token", tokenString)
		if claims.ExpiresAt != nil {
			c.Set("jwt_expires_at", claims.ExpiresAt.Time)
		}
		c.Next()
	}
}

// AdminAuthMiddleware 验证是否为管理员
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, common.R{Code: common.CodeUnauthorized, Msg: "未授权"})
			c.Abort()
			return
		}

		if role.(int) < model.RoleAdmin {
			c.JSON(http.StatusForbidden, common.R{Code: common.CodeForbidden, Msg: "权限不足"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RootAuthMiddleware 验证是否为超级管理员
func RootAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, common.R{Code: common.CodeUnauthorized, Msg: "未授权"})
			c.Abort()
			return
		}

		if role.(int) < model.RoleRoot {
			c.JSON(http.StatusForbidden, common.R{Code: common.CodeForbidden, Msg: "权限不足"})
			c.Abort()
			return
		}
		c.Next()
	}
}
