package middleware

import (
	"STfreApi/model"
	"STfreApi/service"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// auditResponseWriter 捕获响应体，用于判定业务成功/失败
type auditResponseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *auditResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// ContextKeyAuditLogged handler 已手动记录审计日志时设置此 key，中间件跳过
const ContextKeyAuditLogged = "audit_logged"

// AuditLogMiddleware 记录全站写操作（非 GET 请求）
func AuditLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet {
			c.Next()
			return
		}

		// 读取请求体
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// 捕获响应体
		arw := &auditResponseWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = arw

		c.Next()

		// handler 已手动记录则跳过
		if _, ok := c.Get(ContextKeyAuditLogged); ok {
			return
		}

		operatorId, _ := c.Get("id")
		usernameVal, _ := c.Get("username")
		operatorIdInt, _ := operatorId.(int)
		username, _ := usernameVal.(string)
		operatorRole := roleTierLabel(c.GetInt("role"))

		// 路由模板 + 路径参数
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		targetId := extractTargetId(c)
		action := mapRouteToAction(c.Request.Method, route)
		authMethod := detectAuthMethod(c)

		// 解析响应体判定业务成功
		success := parseResponseSuccess(arw.body.Bytes(), c.Writer.Status())

		go service.RecordAudit(service.AuditEntry{
			OperatorId:       operatorIdInt,
			OperatorUsername: username,
			OperatorRole:     operatorRole,
			Method:           c.Request.Method,
			Path:             c.Request.URL.Path,
			Route:            route,
			Action:           action,
			TargetType:       extractTargetType(c.Request.URL.Path),
			TargetId:         targetId,
			Success:          success,
			StatusCode:       c.Writer.Status(),
			IP:               c.ClientIP(),
			AuthMethod:       authMethod,
			RequestId:        c.GetString("request_id"),
			RawBody:          bodyBytes,
		})
	}
}

// roleTierLabel 把数字角色换算成人话档位
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

// extractTargetType 从路径里提取资源名
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

// extractTargetId 从路径参数中提取目标资源 ID
func extractTargetId(c *gin.Context) string {
	for _, key := range []string{"id", "user_id", "channel_id", "token_id", "redemption_id"} {
		if v := c.Param(key); v != "" {
			return v
		}
	}
	return ""
}

// detectAuthMethod 检测鉴权方式
func detectAuthMethod(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return "access_token"
	}
	if _, ok := c.Get("passkey_user"); ok {
		return "passkey"
	}
	if _, ok := c.Get("oauth_user"); ok {
		return "oauth"
	}
	return "session"
}

// parseResponseSuccess 解析响应体判定业务成功
// HTTP 状态码 >= 400 → 失败；否则解析 JSON 的 code 字段，0 = 成功
func parseResponseSuccess(body []byte, statusCode int) bool {
	if statusCode >= 400 {
		return false
	}
	if len(body) == 0 {
		return true // 无响应体，按 HTTP 状态码判定
	}
	var resp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return true // 非 JSON 响应，按 HTTP 状态码判定
	}
	return resp.Code == 0
}

// routeActionMap 路由模板 → 语义动作映射
var routeActionMap = map[string]string{
	// 用户管理
	"POST:/api/user":            "user.create",
	"PUT:/api/user/self":        "user.update_self",
	"PUT:/api/user/:id":         "user.update",
	"DELETE:/api/user/:id":      "user.delete",
	"POST:/api/user/manage_user": "user.manage",

	// 渠道管理
	"POST:/api/channel":         "channel.create",
	"PUT:/api/channel":          "channel.update",
	"PUT:/api/channel/:id":      "channel.update",
	"DELETE:/api/channel/:id":   "channel.delete",
	"POST:/api/channel/test/:id": "channel.test",

	// 令牌管理
	"POST:/api/token":          "token.create",
	"PUT:/api/token":           "token.update",
	"DELETE:/api/token/:id":    "token.delete",

	// 兑换码
	"POST:/api/redemption":     "redemption.create",
	"PUT:/api/redemption":      "redemption.update",
	"DELETE:/api/redemption/:id": "redemption.delete",

	// 系统设置
	"PUT:/api/option":          "option.update",
	"POST:/api/setup":          "system.setup",

	// 签到
	"POST:/api/user/check_in":  "user.checkin",

	// 充值
	"POST:/api/user/top_up":    "user.topup",

	// Passkey
	"POST:/api/user/passkey":           "passkey.register",
	"DELETE:/api/user/passkey":         "passkey.delete",

	// TOTP
	"POST:/api/user/totp/setup":  "totp.setup",
	"POST:/api/user/totp/enable": "totp.enable",
	"POST:/api/user/totp/disable": "totp.disable",

	// OAuth 绑定
	"POST:/api/user/oauth/bind/:provider":   "oauth.bind",
	"DELETE:/api/user/oauth/unbind/:provider": "oauth.unbind",
}

// mapRouteToAction 根据 method + 路由模板查找语义动作
func mapRouteToAction(method, route string) string {
	key := method + ":" + route
	if action, ok := routeActionMap[key]; ok {
		return action
	}
	// 未命中映射时用 method + target_type 生成通用动作
	target := extractTargetType(route)
	if target == "" {
		return strings.ToLower(method)
	}
	switch method {
	case "POST":
		return target + ".create"
	case "PUT", "PATCH":
		return target + ".update"
	case "DELETE":
		return target + ".delete"
	default:
		return strings.ToLower(method) + "." + target
	}
}
