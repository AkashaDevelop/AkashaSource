package controller

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"STfreApi/common"
	"STfreApi/common/email"
	"STfreApi/model"

	"github.com/gin-gonic/gin"
)

type resetCode struct {
	Code      string
	Email     string
	ExpiresAt time.Time
}

var (
	resetCodes     = make(map[string]*resetCode) // key: email
	resetCodesLock sync.RWMutex
	resetAttempts  = make(map[string][]time.Time) // rate limit per email
)

func PasswordResetRequest(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Rate limit: 3 per email per hour
	resetCodesLock.Lock()
	now := time.Now()
	attempts := resetAttempts[req.Email]
	var recent []time.Time
	for _, t := range attempts {
		if now.Sub(t) < time.Hour {
			recent = append(recent, t)
		}
	}
	if len(recent) >= 3 {
		resetCodesLock.Unlock()
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，请稍后再试"})
		return
	}
	resetAttempts[req.Email] = append(recent, now)
	resetCodesLock.Unlock()

	// Check user exists
	var user model.User
	if err := common.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// Don't reveal whether email exists
		c.JSON(http.StatusOK, gin.H{"message": "如果邮箱存在，验证码已发送"})
		return
	}

	// Generate 6-digit code
	code := fmt.Sprintf("%06d", rand.Intn(1000000))

	resetCodesLock.Lock()
	resetCodes[req.Email] = &resetCode{
		Code:      code,
		Email:     req.Email,
		ExpiresAt: now.Add(10 * time.Minute),
	}
	resetCodesLock.Unlock()

	// Send email
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

	resetCodesLock.RLock()
	rc, exists := resetCodes[req.Email]
	resetCodesLock.RUnlock()

	if !exists || rc.Code != req.Code || time.Now().After(rc.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码无效或已过期"})
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

	// Remove used code
	resetCodesLock.Lock()
	delete(resetCodes, req.Email)
	resetCodesLock.Unlock()

	c.JSON(http.StatusOK, gin.H{"message": "密码重置成功"})
}
