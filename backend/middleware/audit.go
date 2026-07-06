package middleware

import (
	"STfreApi/model"
	"STfreApi/service"
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuditLogMiddleware ～不管是普通用户、管理员还是超管，做的每一次改动都会被这只小卫兵默默记下来哦～
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

		operatorId, _ := c.Get("id")
		usernameVal, _ := c.Get("username")
		operatorIdInt, _ := operatorId.(int)
		username, _ := usernameVal.(string)
		operatorRole := roleTierLabel(c.GetInt("role"))

		go service.RecordAudit(
			operatorIdInt,
			username,
			operatorRole,
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

// roleTierLabel ～把数字角色换算成人话档位，喂给审计日志用～
func roleTierLabel(role int) string {
	switch {
	case role >= model.RoleRoot:
		return "root"
	case role >= model.RoleAdmin:
		return "admin"
	default:
		return "user"
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
