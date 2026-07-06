package middleware

// ～数据库还没接通的时候，别让那些查表的接口冲上去摔跟头啦～
// 引导式初始化向导完成前，common.DB 还是 nil，这里挡在最前面给一个
// 好懂的 503 提示，而不是让每个 controller 各自摔一次 nil 指针panic。
//
// 采用"黑名单"而不是"白名单"：默认放行，只拦截已知一定要查数据库的
// 路径前缀。这样以后前端构建产物 embed 进后端、多出静态资源/SPA
// fallback 路由时，天然不受影响，不需要回头再改这份名单。

import (
	"net/http"
	"strings"

	"STfreApi/common"

	"github.com/gin-gonic/gin"
)

// dbRequiredPrefixes 命中即认为该请求需要数据库支持
var dbRequiredPrefixes = []string{
	"/v1/", "/mj/", "/suno/", "/gemini/", "/jimeng", "/kling/", "/pg/", "/dashboard/",
	"/api/",
}

// dbFreePrefixes 即使命中上面的黑名单前缀，这些路径也永远放行
// （初始化向导本身、系统状态查询、CxSec 握手不依赖业务数据库）
var dbFreePrefixes = []string{
	"/api/setup", "/api/system/status", "/api/status", "/api/cx/",
}

func RequireDatabaseReady() gin.HandlerFunc {
	return func(c *gin.Context) {
		if common.DB != nil {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		for _, free := range dbFreePrefixes {
			if strings.HasPrefix(path, free) {
				c.Next()
				return
			}
		}
		for _, required := range dbRequiredPrefixes {
			if strings.HasPrefix(path, required) {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, common.R{
				Code: common.CodeServerError,
				Msg:  "数据库尚未配置，请先完成初始化向导",
			})
				return
			}
		}
		c.Next()
	}
}
