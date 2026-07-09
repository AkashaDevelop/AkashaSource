package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"STfreApi/common"
	"STfreApi/service"
	"STfreApi/service/sanction"

	"github.com/gin-gonic/gin"
)

// hashKey ～把又长又臭的 token 摘要成32字节小卡片，Redis 存的更省心～
func hashKey(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

type RateLimiter struct {
	limits sync.Map
	rate   int
	window time.Duration
}

type LimitInfo struct {
	count     int
	resetTime time.Time
	mu        sync.Mutex
}

var GlobalRateLimiter *RateLimiter

func InitRateLimiter(rpm int) {
	GlobalRateLimiter = &RateLimiter{
		rate:   rpm,
		window: time.Minute,
	}
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			now := time.Now()
			GlobalRateLimiter.limits.Range(func(key, value interface{}) bool {
				info := value.(*LimitInfo)
				info.mu.Lock()
				if now.After(info.resetTime.Add(GlobalRateLimiter.window)) {
					GlobalRateLimiter.limits.Delete(key)
				}
				info.mu.Unlock()
				return true
			})
		}
	}()
}

func checkLimit(key string, limit int) bool {
	if common.RedisClient != nil {
		return redisCheckLimit(key, limit, GlobalRateLimiter.window)
	}
	val, _ := GlobalRateLimiter.limits.LoadOrStore(key, &LimitInfo{
		resetTime: time.Now().Add(GlobalRateLimiter.window),
	})
	info := val.(*LimitInfo)

	info.mu.Lock()
	defer info.mu.Unlock()

	if time.Now().After(info.resetTime) {
		info.count = 0
		info.resetTime = time.Now().Add(GlobalRateLimiter.window)
	}

	info.count++
	return info.count <= limit
}

func redisCheckLimit(key string, limit int, window time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	count, err := common.RedisClient.Incr(ctx, key).Result()
	if err != nil {
		return true
	}
	if count == 1 {
		common.RedisClient.Expire(ctx, key, window)
	}
	return count <= int64(limit)
}

func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// ～最前置：IP 黑名单拦截，恶意流量进门第一道就挡掉，比查库/转发都早～
		if banned, reason := sanction.IsIPBanned(c.ClientIP()); banned {
			msg := "访问被拒绝"
			if reason != "" {
				msg = "访问被拒绝: " + reason
			}
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"message": msg,
					"type":    "forbidden",
				},
			})
			c.Abort()
			return
		}

		if GlobalRateLimiter == nil {
			c.Next()
			return
		}

		ip := c.ClientIP()
		ipLimit := GlobalRateLimiter.rate
		if !checkLimit("ip:"+ip, ipLimit) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"message": "请求过于频繁（IP 限流）",
					"type":    "rate_limit_error",
				},
			})
			c.Abort()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			tokenKey := hashKey(strings.TrimPrefix(authHeader, "Bearer "))
			tokenLimit := GlobalRateLimiter.rate * 3
			if !checkLimit("token:"+tokenKey, tokenLimit) {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": gin.H{
						"message": "请求过于频繁（令牌限流）",
						"type":    "rate_limit_error",
					},
				})
				c.Abort()
				return
			}
		}

		userId, exists := c.Get("id")
		if exists {
			userKey := fmt.Sprint(userId)
			userLimit := GlobalRateLimiter.rate * 5
			if !checkLimit("user:"+userKey, userLimit) {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": gin.H{
						"message": "请求过于频繁（用户限流）",
						"type":    "rate_limit_error",
					},
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// ModelRateLimitMiddleware checks per-model rate limits
func ModelRateLimitMiddleware(modelName string) bool {
	if GlobalRateLimiter == nil {
		return true
	}
	common.OptionLock.RLock()
	modelRPMStr := common.OptionMap["model_rpm"]
	common.OptionLock.RUnlock()

	if modelRPMStr == "" {
		return true
	}

	var modelRPM map[string]int
	if err := json.Unmarshal([]byte(modelRPMStr), &modelRPM); err != nil {
		return true
	}

	limit, ok := modelRPM[modelName]
	if !ok {
		return true
	}

	return checkLimit("model:"+modelName, limit)
}

// GroupRateLimitMiddleware 按 model.Group.QPM 给分组单独限流，QPM<=0 表示这个分组不限～
func GroupRateLimitMiddleware(groupName string) bool {
	if GlobalRateLimiter == nil || groupName == "" {
		return true
	}
	qpm := service.GetGroupQPM(groupName)
	if qpm <= 0 {
		return true
	}
	return checkLimit("group:"+groupName, qpm)
}

// CheckTokenSanction 检查 Token 级处置（停用/降速/固定低RPM），命中拦截则写响应并返回 false。
// 在 relay controller 里 token 解析完成后、转发渠道前调用，与 ModelRateLimitMiddleware 同级。
func CheckTokenSanction(c *gin.Context, tokenID int) bool {
	if tokenID <= 0 {
		return true
	}
	ts := sanction.GetTokenSanction(tokenID)
	if ts == nil {
		return true
	}

	// 停用（suspend_token / disable_token）：直接 403
	if ts.Disabled {
		msg := "令牌已被停用"
		if ts.Reason != "" {
			msg = "令牌已被停用: " + ts.Reason
		}
		c.JSON(http.StatusForbidden, gin.H{
			"error": gin.H{"message": msg, "type": "forbidden"},
		})
		c.Abort()
		return false
	}

	if GlobalRateLimiter == nil {
		return true
	}

	// 固定低 RPM 惩罚优先于降速倍率（rpm_limit 是绝对值，更严格）
	limit := 0
	switch {
	case ts.RPMLimit > 0:
		limit = ts.RPMLimit
	case ts.ThrottleFactor > 0 && ts.ThrottleFactor < 1.0:
		// 在令牌原有限额（rate*3）基础上按倍率降速
		limit = int(float64(GlobalRateLimiter.rate*3) * ts.ThrottleFactor)
		if limit < 1 {
			limit = 1
		}
	default:
		return true
	}

	if !checkLimit(fmt.Sprintf("sanction_throttle:%d", tokenID), limit) {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": gin.H{
				"message": "请求过于频繁（风控降速处置中）",
				"type":    "rate_limit_error",
			},
		})
		c.Abort()
		return false
	}
	return true
}

// CriticalRateLimitMiddleware limits critical operations (login, register, password reset) to 5/min per IP
func CriticalRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if GlobalRateLimiter == nil {
			c.Next()
			return
		}
		ip := c.ClientIP()
		if !checkLimit("critical:"+ip, 5) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "操作过于频繁，请稍后再试",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
