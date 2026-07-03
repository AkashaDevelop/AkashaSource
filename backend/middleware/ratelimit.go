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
