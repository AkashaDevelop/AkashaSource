package controller

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"STfreApi/common"
	"STfreApi/common/email"

	"github.com/gin-gonic/gin"
)

type verifyCode struct {
	Code      string
	ExpiresAt time.Time
}

var (
	verifyCodes     = make(map[string]*verifyCode) // key: email
	verifyCodesLock sync.RWMutex
	verifyAttempts  = make(map[string][]time.Time)
)

func SendEmailVerifyCode(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if common.SMTPServer == "" {
		common.Fail(c, common.CodeServerError, "邮件服务未配置")
		return
	}

	verifyCodesLock.Lock()
	now := time.Now()
	var recent []time.Time
	for _, t := range verifyAttempts[req.Email] {
		if now.Sub(t) < time.Hour {
			recent = append(recent, t)
		}
	}
	if len(recent) >= 5 {
		verifyCodesLock.Unlock()
		common.Fail(c, common.CodeParamError, "发送过于频繁，请稍后再试")
		return
	}
	verifyAttempts[req.Email] = append(recent, now)
	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	verifyCodes[req.Email] = &verifyCode{Code: code, ExpiresAt: now.Add(10 * time.Minute)}
	verifyCodesLock.Unlock()

	subject := fmt.Sprintf("%s - 邮箱验证码", common.SystemName)
	body := fmt.Sprintf("<p>您的注册验证码是: <strong>%s</strong></p><p>有效期10分钟，请勿泄露。</p>", code)
	go email.SendEmail(req.Email, subject, body)

	common.OKMsg(c, "验证码已发送", nil)
}

func CheckEmailVerifyCode(emailAddr, code string) bool {
	verifyCodesLock.RLock()
	vc, ok := verifyCodes[emailAddr]
	verifyCodesLock.RUnlock()
	if !ok || vc.Code != code || time.Now().After(vc.ExpiresAt) {
		return false
	}
	verifyCodesLock.Lock()
	delete(verifyCodes, emailAddr)
	verifyCodesLock.Unlock()
	return true
}
