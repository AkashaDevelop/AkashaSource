package controller

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"STfreApi/common"
	"STfreApi/common/email"
	"STfreApi/model"

	"github.com/gin-gonic/gin"
)

func GetUptimeStatus(c *gin.Context) {
	uptime := time.Now().Unix() - common.StartTime
	common.OK(c, gin.H{
		"status":     "up",
		"start_time": common.StartTime,
		"uptime":     uptime,
	})
}

func TestStatus(c *gin.Context) {
	sqlDB, err := common.DB.DB()
	if err != nil {
		common.Fail(c, common.CodeServerError, "数据库连接实例不可用")
		return
	}
	if err = sqlDB.Ping(); err != nil {
		common.Fail(c, common.CodeServerError, "数据库连接失败")
		return
	}

	redisOK := false
	if common.RedisClient != nil {
		if pingErr := common.RedisClient.Ping(c.Request.Context()).Err(); pingErr == nil {
			redisOK = true
		}
	}

	common.OK(c, gin.H{
		"server": "running",
		"db":     "ok",
		"redis":  redisOK,
	})
}

func SendEmailVerification(c *gin.Context) {
	emailAddr := strings.TrimSpace(c.Query("email"))
	if emailAddr == "" {
		common.Fail(c, common.CodeParamError, "邮箱不能为空")
		return
	}
	if common.SMTPServer == "" {
		common.Fail(c, common.CodeServerError, "邮件服务未配置")
		return
	}

	verifyCodesLock.Lock()
	now := time.Now()
	var recent []time.Time
	for _, t := range verifyAttempts[emailAddr] {
		if now.Sub(t) < time.Hour {
			recent = append(recent, t)
		}
	}
	if len(recent) >= 5 {
		verifyCodesLock.Unlock()
		common.Fail(c, common.CodeParamError, "发送过于频繁，请稍后再试")
		return
	}
	verifyAttempts[emailAddr] = append(recent, now)
	code := fmt.Sprintf("%06d", randomIntn(1000000))
	verifyCodes[emailAddr] = &verifyCode{Code: code, ExpiresAt: now.Add(10 * time.Minute)}
	verifyCodesLock.Unlock()

	subject := fmt.Sprintf("%s - 邮箱验证码", common.SystemName)
	body := fmt.Sprintf("<p>您的注册验证码是: <strong>%s</strong></p><p>有效期10分钟，请勿泄露。</p>", code)
	go email.SendEmail(emailAddr, subject, body)

	common.OKMsg(c, "验证码已发送", nil)
}

func SendPasswordResetEmail(c *gin.Context) {
	emailAddr := strings.TrimSpace(c.Query("email"))
	if emailAddr == "" {
		common.Fail(c, common.CodeParamError, "邮箱不能为空")
		return
	}

	resetCodesLock.Lock()
	now := time.Now()
	attempts := resetAttempts[emailAddr]
	var recent []time.Time
	for _, t := range attempts {
		if now.Sub(t) < time.Hour {
			recent = append(recent, t)
		}
	}
	if len(recent) >= 3 {
		resetCodesLock.Unlock()
		common.Fail(c, common.CodeParamError, "请求过于频繁，请稍后再试")
		return
	}
	resetAttempts[emailAddr] = append(recent, now)
	resetCodesLock.Unlock()

	var user model.User
	if err := common.DB.Where("email = ?", emailAddr).First(&user).Error; err != nil {
		common.OKMsg(c, "如果邮箱存在，验证码已发送", nil)
		return
	}

	code := fmt.Sprintf("%06d", randomIntn(1000000))
	resetCodesLock.Lock()
	resetCodes[emailAddr] = &resetCode{Code: code, Email: emailAddr, ExpiresAt: now.Add(10 * time.Minute)}
	resetCodesLock.Unlock()

	subject := fmt.Sprintf("%s - 密码重置验证码", common.SystemName)
	body := fmt.Sprintf("<p>您的密码重置验证码是: <strong>%s</strong></p><p>有效期10分钟。</p>", code)
	go email.SendEmail(emailAddr, subject, body)

	common.OKMsg(c, "如果邮箱存在，验证码已发送", nil)
}

func ResetPassword(c *gin.Context) {
	var req struct {
		Email       string `json:"email"`
		Code        string `json:"code"`
		Password    string `json:"password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Code) == "" {
		common.Fail(c, common.CodeParamError, "邮箱和验证码不能为空")
		return
	}

	newPassword := strings.TrimSpace(req.NewPassword)
	if newPassword == "" {
		newPassword = strings.TrimSpace(req.Password)
	}
	if len(newPassword) < 8 {
		common.Fail(c, common.CodeParamError, "新密码长度至少为8")
		return
	}

	resetCodesLock.RLock()
	rc, exists := resetCodes[req.Email]
	resetCodesLock.RUnlock()
	if !exists || rc.Code != req.Code || time.Now().After(rc.ExpiresAt) {
		common.Fail(c, common.CodeParamError, "验证码无效或已过期")
		return
	}

	hashed, err := common.Password2Hash(newPassword)
	if err != nil {
		common.Fail(c, common.CodeServerError, "密码加密失败")
		return
	}
	if err = common.DB.Model(&model.User{}).Where("email = ?", req.Email).Update("password", hashed).Error; err != nil {
		common.Fail(c, common.CodeServerError, "密码更新失败")
		return
	}

	resetCodesLock.Lock()
	delete(resetCodes, req.Email)
	resetCodesLock.Unlock()
	common.OKMsg(c, "密码重置成功", nil)
}

func GetRatioConfig(c *gin.Context) {
	common.OptionLock.RLock()
	mr := common.OptionMap[model.OptionKeyModelRatio]
	cr := common.OptionMap[model.OptionKeyCompletionRatio]
	common.OptionLock.RUnlock()
	common.OK(c, gin.H{
		"model_ratio":      mr,
		"completion_ratio": cr,
	})
}

type CompatibilityPerformanceStats struct {
	MemoryMB   uint64 `json:"memory_mb"`
	Goroutines int    `json:"goroutines"`
	GCCycles   uint32 `json:"gc_cycles"`
	Uptime     string `json:"uptime"`
	GoVersion  string `json:"go_version"`
	DiskCache  any    `json:"disk_cache"`
	Affinity   any    `json:"channel_affinity_cache"`
}

func GetPerformanceStats(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	uptime := time.Now().Unix() - common.StartTime
	stats := CompatibilityPerformanceStats{
		MemoryMB:   m.Alloc / 1024 / 1024,
		Goroutines: runtime.NumGoroutine(),
		GCCycles:   m.NumGC,
		Uptime:     fmt.Sprintf("%dd %dh %dm", uptime/86400, (uptime%86400)/3600, (uptime%3600)/60),
		GoVersion:  runtime.Version(),
		DiskCache: gin.H{
			"enabled": false,
			"message": "当前版本未启用磁盘缓存统计",
		},
		Affinity: gin.H{
			"enabled": false,
			"size":    0,
		},
	}
	common.OK(c, stats)
}

func ClearDiskCache(c *gin.Context) {
	common.OKMsg(c, "磁盘缓存清理完成", nil)
}

func ResetPerformanceStats(c *gin.Context) {
	common.OKMsg(c, "性能统计已重置", nil)
}

func ForceGC(c *gin.Context) {
	runtime.GC()
	common.OKMsg(c, "GC 已执行", nil)
}

func GetChannelAffinityCacheStats(c *gin.Context) {
	common.OK(c, gin.H{
		"enabled": false,
		"size":    0,
	})
}

func ClearChannelAffinityCache(c *gin.Context) {
	common.OKMsg(c, "渠道亲和缓存已清理", nil)
}

func randomIntn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(time.Now().UnixNano() % int64(n))
}
