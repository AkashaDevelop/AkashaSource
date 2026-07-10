package controller

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"STfreApi/common"
	"STfreApi/common/email"
	"STfreApi/model"
	"STfreApi/service"

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
		Email string `json:"email"`
		Code  string `json:"code"`
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	emailAddr := strings.TrimSpace(req.Email)
	verifyCode := strings.TrimSpace(req.Code)
	if verifyCode == "" {
		verifyCode = strings.TrimSpace(req.Token)
	}
	if emailAddr == "" || verifyCode == "" {
		common.Fail(c, common.CodeParamError, "邮箱和验证码不能为空")
		return
	}

	resetCodesLock.RLock()
	rc, exists := resetCodes[emailAddr]
	resetCodesLock.RUnlock()
	if !exists || rc.Code != verifyCode || time.Now().After(rc.ExpiresAt) {
		common.Fail(c, common.CodeParamError, "验证码无效或已过期")
		return
	}

	generatedPassword := fmt.Sprintf("%06d%06d", randomIntn(1000000), randomIntn(1000000))
	hashed, err := common.Password2Hash(generatedPassword)
	if err != nil {
		common.Fail(c, common.CodeServerError, "密码加密失败")
		return
	}
	if err = common.DB.Model(&model.User{}).Where("email = ?", emailAddr).Update("password", hashed).Error; err != nil {
		common.Fail(c, common.CodeServerError, "密码更新失败")
		return
	}

	resetCodesLock.Lock()
	delete(resetCodes, emailAddr)
	resetCodesLock.Unlock()
	common.OK(c, gin.H{"password": generatedPassword})
}

func GetRatioConfig(c *gin.Context) {
	common.OptionLock.RLock()
	expose := strings.ToLower(strings.TrimSpace(common.OptionMap["expose_ratio_config"]))
	mr := common.OptionMap[model.OptionKeyModelRatio]
	cr := common.OptionMap[model.OptionKeyCompletionRatio]
	common.OptionLock.RUnlock()
	if expose == "false" || expose == "0" || expose == "off" {
		common.Fail(c, common.CodeForbidden, "倍率配置接口未启用")
		return
	}
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

var performanceStatsResetAt = common.StartTime

func diskCacheCandidateDirs() []string {
	return []string{
		"cache",
		"tmp",
		"temp",
		filepath.Join("data", "cache"),
	}
}

func scanDiskCacheDirs() gin.H {
	type dirStat struct {
		Path      string `json:"path"`
		Exists    bool   `json:"exists"`
		FileCount int    `json:"file_count"`
		TotalSize int64  `json:"total_size"`
	}

	dirs := diskCacheCandidateDirs()
	details := make([]dirStat, 0, len(dirs))
	totalFiles := 0
	var totalSize int64
	for _, p := range dirs {
		stat := dirStat{Path: p}
		entries, err := os.ReadDir(p)
		if err != nil {
			details = append(details, stat)
			continue
		}
		stat.Exists = true
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			info, infoErr := entry.Info()
			if infoErr != nil {
				continue
			}
			stat.FileCount++
			stat.TotalSize += info.Size()
		}
		totalFiles += stat.FileCount
		totalSize += stat.TotalSize
		details = append(details, stat)
	}

	return gin.H{
		"enabled":       true,
		"supported":     true,
		"directories":   details,
		"total_files":   totalFiles,
		"total_size":    totalSize,
		"last_reset_at": performanceStatsResetAt,
	}
}

func GetPerformanceStats(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	uptime := time.Now().Unix() - common.StartTime
	affinityStats, err := getChannelAffinityCacheStats()
	if err != nil {
		affinityStats = map[string]any{
			"enabled": false,
			"error":   "渠道亲和统计不可用",
		}
	}

	stats := CompatibilityPerformanceStats{
		MemoryMB:   m.Alloc / 1024 / 1024,
		Goroutines: runtime.NumGoroutine(),
		GCCycles:   m.NumGC,
		Uptime:     fmt.Sprintf("%dd %dh %dm", uptime/86400, (uptime%86400)/3600, (uptime%3600)/60),
		GoVersion:  runtime.Version(),
		DiskCache:  scanDiskCacheDirs(),
		Affinity:   affinityStats,
	}
	common.OK(c, stats)
}

func ClearDiskCache(c *gin.Context) {
	maxAgeMinutes := 10
	if raw := strings.TrimSpace(c.Query("max_age_minutes")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			maxAgeMinutes = parsed
		}
	}
	cutoff := time.Now().Add(-time.Duration(maxAgeMinutes) * time.Minute)

	deleted := 0
	var reclaimed int64
	errors := make([]gin.H, 0)

	for _, dir := range diskCacheCandidateDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			fullPath := filepath.Join(dir, entry.Name())
			info, infoErr := entry.Info()
			if infoErr != nil {
				errors = append(errors, gin.H{"path": fullPath, "error": infoErr.Error()})
				continue
			}
			if maxAgeMinutes > 0 && info.ModTime().After(cutoff) {
				continue
			}
			size := info.Size()
			if removeErr := os.Remove(fullPath); removeErr != nil {
				errors = append(errors, gin.H{"path": fullPath, "error": removeErr.Error()})
				continue
			}
			deleted++
			reclaimed += size
		}
	}

	common.OK(c, gin.H{
		"deleted_files":   deleted,
		"reclaimed_bytes": reclaimed,
		"max_age_minutes": maxAgeMinutes,
		"error_count":     len(errors),
		"errors":          errors,
	})
}

func ResetPerformanceStats(c *gin.Context) {
	performanceStatsResetAt = time.Now().Unix()
	common.OK(c, gin.H{"reset_at": performanceStatsResetAt})
}

func ForceGC(c *gin.Context) {
	runtime.GC()
	common.OKMsg(c, "GC 已执行", nil)
}

func GetChannelAffinityCacheStats(c *gin.Context) {
	stats, err := getChannelAffinityCacheStats()
	if err != nil {
		common.Fail(c, common.CodeServerError, "获取渠道亲和缓存统计失败")
		return
	}
	common.OK(c, stats)
}

func ClearChannelAffinityCache(c *gin.Context) {
	all := strings.TrimSpace(c.Query("all"))
	ruleName := strings.TrimSpace(c.Query("rule_name"))

	if strings.EqualFold(all, "true") {
		deleted, err := clearChannelAffinityCacheAll()
		if err != nil {
			common.Fail(c, common.CodeServerError, "清理渠道亲和缓存失败")
			return
		}
		common.OK(c, gin.H{"deleted": deleted, "supported": true, "mode": "redis"})
		return
	}

	if ruleName == "" {
		common.Fail(c, common.CodeParamError, "缺少参数：rule_name，或使用 all=true 清空全部")
		return
	}

	deleted, err := clearChannelAffinityCacheByRuleName(ruleName)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	common.OK(c, gin.H{"deleted": deleted, "supported": true, "mode": "redis"})
}

func estimateChannelAffinityCacheStats(windowSeconds int64) (gin.H, error) {
	if windowSeconds <= 0 {
		windowSeconds = 3600
	}
	since := time.Now().Unix() - windowSeconds

	type row struct {
		ChannelID int
		ModelName string
		Hit       int64
		LastSeen  int64
	}

	var rows []row
	err := common.DB.Model(&model.Log{}).
		Select("channel_id, model_name, COUNT(*) as hit, MAX(created_at) as last_seen").
		Where("type = ? AND created_at >= ? AND channel_id > 0", model.LogTypeConsume, since).
		Group("channel_id, model_name").
		Order("hit DESC").
		Limit(200).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	items := make([]gin.H, 0, len(rows))
	var maxLastSeen int64
	for _, r := range rows {
		items = append(items, gin.H{
			"channel_id":   r.ChannelID,
			"model_name":   r.ModelName,
			"hit":          r.Hit,
			"last_seen_at": r.LastSeen,
		})
		if r.LastSeen > maxLastSeen {
			maxLastSeen = r.LastSeen
		}
	}

	return gin.H{
		"enabled":        true,
		"mode":           "log_estimated",
		"window_seconds": windowSeconds,
		"size":           len(items),
		"last_seen_at":   maxLastSeen,
		"items":          items,
	}, nil
}

func randomIntn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(time.Now().UnixNano() % int64(n))
}

// CheckUpdate 手动触发版本检查
func CheckUpdate(c *gin.Context) {
	info, err := service.DoVersionCheck()
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}
	common.OK(c, info)
}

// GetUpdateInfo 获取上次版本检查结果（不触发网络请求）
func GetUpdateInfo(c *gin.Context) {
	info := service.GetCachedUpdateInfo()
	if info == nil {
		common.OK(c, gin.H{"has_update": false, "last_checked": ""})
		return
	}
	common.OK(c, info)
}
