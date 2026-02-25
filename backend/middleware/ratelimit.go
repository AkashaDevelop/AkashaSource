package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	limits sync.Map // map[string]*LimitInfo
	rate   int      // requests per minute
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
	
	// Cleanup routine to prevent memory leak
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			now := time.Now()
			GlobalRateLimiter.limits.Range(func(key, value interface{}) bool {
				info := value.(*LimitInfo)
				info.mu.Lock()
				// If reset time + window < now, it means it's expired for a while
				if now.After(info.resetTime.Add(GlobalRateLimiter.window)) {
					GlobalRateLimiter.limits.Delete(key)
				}
				info.mu.Unlock()
				return true
			})
		}
	}()
}

func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if GlobalRateLimiter == nil {
			c.Next()
			return
		}

		// Priority: Token > IP
		key := c.ClientIP()
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			key = authHeader // Use the token (or Bearer string) as key
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
		if info.count > GlobalRateLimiter.rate {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"message": "Too Many Requests",
					"type":    "rate_limit_error",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
