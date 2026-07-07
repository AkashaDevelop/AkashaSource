// ⚠️ REMOVABLE MODULE — 系统授权门禁全局拦截
// 整体移除时：删掉这个文件 + router/api.go 里注册这个中间件的那一行即可
package middleware

import (
	"net/http"
	"strings"

	"STfreApi/common"
	"STfreApi/service/license"

	"github.com/gin-gonic/gin"
)

// licenseFreePrefixes 即使系统未授权，这些路径也永远放行——
// 初始化向导/公开状态/CxSec 握手是既有的必须豁免项；登录相关路径是为了让站长
// 能先登进后台，再去完成 GitHub 组织授权，不然会死锁进不去
var licenseFreePrefixes = []string{
	"/api/setup", "/api/system/status", "/api/status", "/api/cx/",
	"/api/user/login", "/api/user/self", "/oauth/",
	"/api/system-license/",
}

func RequireSystemLicensed() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !license.FeatureEnabled() || license.IsAuthorized() {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		for _, p := range licenseFreePrefixes {
			if strings.HasPrefix(path, p) {
				c.Next()
				return
			}
		}

		// ～和 RequireDatabaseReady 保持一致：这种系统级硬拦截跳出"永远 200 + 业务码"的常规约定，
		// 直接给真实的 503，方便监控/反代一眼识别出这是系统级故障而不是普通业务错误～
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, common.R{
			Code: common.CodeForbidden,
			Msg:  "系统未完成授权，请联系站长在后台完成 GitHub 组织授权",
		})
	}
}
