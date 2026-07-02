package middleware

import (
	"STfreApi/common"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ～给每一个飞奔而来的请求都发一张小小的身份牌，方便之后追查它的足迹哦～
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestId := c.GetHeader(common.RequestIdKey)
		if requestId == "" {
			// 客户端没带牌子？那就由服务器亲自签发一个吧～
			requestId = uuid.NewString()
		}
		c.Set("request_id", requestId)
		c.Writer.Header().Set(common.RequestIdKey, requestId)
		c.Next()
	}
}
