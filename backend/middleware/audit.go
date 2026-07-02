package middleware

import (
	"STfreApi/service"
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuditLogMiddleware ～管理员大人做的每一次改动，都会被这只小卫兵默默记下来哦～
func AuditLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet {
			c.Next()
			return
		}

		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		c.Next()

		adminId, _ := c.Get("id")
		usernameVal, _ := c.Get("username")
		adminIdInt, _ := adminId.(int)
		username, _ := usernameVal.(string)

		go service.RecordAdminAudit(
			adminIdInt,
			username,
			c.Request.Method,
			c.Request.URL.Path,
			extractTargetType(c.Request.URL.Path),
			c.ClientIP(),
			c.GetString("request_id"),
			c.Writer.Status(),
			bodyBytes,
		)
	}
}

// extractTargetType ～从路径里揪出这次动的是哪块小蛋糕，比如 /api/user/5 -> user～
func extractTargetType(path string) string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for _, seg := range segments {
		if seg == "" || seg == "api" || seg == "admin" {
			continue
		}
		return seg
	}
	return ""
}
