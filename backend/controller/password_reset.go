package controller

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"STfreApi/common"
	"STfreApi/common/email"
	"STfreApi/model"

	"github.com/gin-gonic/gin"
)

type resetCode struct {
	Code      string    `json:"code"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
	Attempts  int       `json:"attempts"`
}

// 内存降级存储（Redis 不可用时使用）
var (
	resetCodes     = make(map[string]*resetCode)
	resetCodesLock sync.RWMutex
	resetAttempts  = make(map[string][]time.Time)
)

const (
	resetCodeTTL     = 10 * time.Minute
	resetCodeKey     = "pwd_reset_code:"
	resetAttemptKey  = "pwd_reset_attempt:"
	resetMaxAttempts = 3
	resetCodeMaxTry  = 5
)

// genSecureCode 用密码学安全随机数生成6位数字验证码
func genSecureCode() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("crypto/rand unavailable: %w", err)
	}
	n := binary.BigEndian.Uint64(b[:]) % 1_000_000
	return fmt.Sprintf("%06d", n), nil
}

// saveResetCode 保存验证码（Redis 优先，内存降级）
func saveResetCode(email, code string) {
	rc := &resetCode{
		Code:      code,
		Email:     email,
		ExpiresAt: time.Now().Add(resetCodeTTL),
	}

	if common.RedisClient != nil {
		ctx := context.Background()
		if data, err := json.Marshal(rc); err == nil {
			common.RedisClient.Set(ctx, resetCodeKey+email, data, resetCodeTTL)
		}
		return
	}

	resetCodesLock.Lock()
	resetCodes[email] = rc
	resetCodesLock.Unlock()
}

// getResetCode 获取验证码
func getResetCode(email string) (*resetCode, bool) {
	if common.RedisClient != nil {
		ctx := context.Background()
		data, err := common.RedisClient.Get(ctx, resetCodeKey+email).Result()
		if err != nil {
			return nil, false
		}
		var rc resetCode
		if err := json.Unmarshal([]byte(data), &rc); err != nil {
			return nil, false
		}
		return &rc, true
	}

	resetCodesLock.RLock()
	rc, exists := resetCodes[email]
	resetCodesLock.RUnlock()
	return rc, exists
}

// deleteResetCode 删除验证码
func deleteResetCode(email string) {
	if common.RedisClient != nil {
		common.RedisClient.Del(context.Background(), resetCodeKey+email)
		return
	}
	resetCodesLock.Lock()
	delete(resetCodes, email)
	resetCodesLock.Unlock()
}

// incrementResetAttempts 递增尝试次数并返回当前次数
func incrementResetAttempts(email string) int {
	if common.RedisClient != nil {
		ctx := context.Background()
		key := resetCodeKey + email + ":tries"
		count, err := common.RedisClient.Incr(ctx, key).Result()
		if err != nil {
			return 0
		}
		if count == 1 {
			common.RedisClient.Expire(ctx, key, resetCodeTTL)
		}
		return int(count)
	}

	rc, exists := getResetCode(email)
	if !exists {
		return 0
	}
	rc.Attempts++
	// 内存模式需要写回
	resetCodesLock.Lock()
	resetCodes[email] = rc
	resetCodesLock.Unlock()
	return rc.Attempts
}

// checkResetRateLimit 每邮箱每小时最多3次请求
func checkResetRateLimit(email string) bool {
	if common.RedisClient != nil {
		ctx := context.Background()
		key := resetAttemptKey + email
		count, err := common.RedisClient.Incr(ctx, key).Result()
		if err != nil {
			return true // fail-open
		}
		if count == 1 {
			common.RedisClient.Expire(ctx, key, time.Hour)
		}
		return count <= int64(resetMaxAttempts)
	}

	// 内存降级
	resetCodesLock.Lock()
	defer resetCodesLock.Unlock()
	now := time.Now()
	attempts := resetAttempts[email]
	var recent []time.Time
	for _, t := range attempts {
		if now.Sub(t) < time.Hour {
			recent = append(recent, t)
		}
	}
	if len(recent) >= resetMaxAttempts {
		return false
	}
	resetAttempts[email] = append(recent, now)
	return true
}

func PasswordResetRequest(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Rate limit: 3 per email per hour
	if !checkResetRateLimit(req.Email) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，请稍后再试"})
		return
	}

	// Check user exists
	var user model.User
	if err := common.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "如果邮箱存在，验证码已发送"})
		return
	}

	code, err := genSecureCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "系统错误，请稍后再试"})
		return
	}

	saveResetCode(req.Email, code)

	subject := fmt.Sprintf("%s - 密码重置验证码", common.SystemName)
	body := fmt.Sprintf("<p>您的密码重置验证码是: <strong>%s</strong></p><p>有效期10分钟。</p>", code)
	go email.SendEmail(req.Email, subject, body)

	c.JSON(http.StatusOK, gin.H{"message": "如果邮箱存在，验证码已发送"})
}

func PasswordResetConfirm(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required"`
		Code        string `json:"code" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rc, exists := getResetCode(req.Email)
	if !exists || time.Now().After(rc.ExpiresAt) {
		deleteResetCode(req.Email)
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码无效或已过期"})
		return
	}

	tries := incrementResetAttempts(req.Email)
	if tries > resetCodeMaxTry {
		deleteResetCode(req.Email)
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码已失效，请重新申请"})
		return
	}

	if rc.Code != req.Code {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误"})
		return
	}

	// Update password
	hashed, err := common.Password2Hash(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	if err := common.DB.Model(&model.User{}).Where("email = ?", req.Email).Update("password", hashed).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码更新失败"})
		return
	}

	deleteResetCode(req.Email)
	c.JSON(http.StatusOK, gin.H{"message": "密码重置成功"})
}
