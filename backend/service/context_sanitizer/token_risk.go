package context_sanitizer

import (
	"sync"
	"time"
)

// TokenRiskTracker 追踪 token 级别的风险累积
// 设计文档 4.10 & 6.7: Token 级风险累积与自适应对抗
type TokenRiskTracker struct {
	mu            sync.RWMutex
	risks         map[int]*tokenRisk // key: tokenId
	slidingWindow time.Duration
	decayRate     float64
}

type tokenRisk struct {
	tokenId         int
	riskFloor       int       // 当前风险基线分
	triggerCount    int       // 触发次数
	lastTriggerTime time.Time // 最后触发时间
	lastDecayTime   time.Time // 最后衰减时间
	probeDetected   bool      // 是否检测到规则探测
	payloadHashes   []string  // 最近 payload 的哈希 (用于探测检测)
}

var globalTokenRiskTracker = &TokenRiskTracker{
	risks:         make(map[int]*tokenRisk),
	slidingWindow: 5 * time.Minute,
	decayRate:     0.5,
}

// RecordTokenTrigger 记录 token 触发检测事件
func RecordTokenTrigger(tokenId int, payloadHash string, riskScore int) int {
	if tokenId <= 0 {
		return 0
	}

	tracker := globalTokenRiskTracker
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	risk := tracker.risks[tokenId]
	if risk == nil {
		risk = &tokenRisk{
			tokenId:       tokenId,
			lastDecayTime: time.Now(),
			payloadHashes: make([]string, 0),
		}
		tracker.risks[tokenId] = risk
	}

	now := time.Now()

	// 滑动窗口外的触发清零
	if now.Sub(risk.lastTriggerTime) > tracker.slidingWindow {
		risk.triggerCount = 0
		risk.payloadHashes = nil
	}

	// 自动衰减 (每 5 分钟衰减 50%)
	if now.Sub(risk.lastDecayTime) > tracker.slidingWindow {
		risk.riskFloor = int(float64(risk.riskFloor) * tracker.decayRate)
		risk.lastDecayTime = now
	}

	// 记录触发
	risk.triggerCount++
	risk.lastTriggerTime = now

	// 提升风险基线 (线性增长)
	riskFloorIncrement := 5
	if riskScore >= 80 {
		riskFloorIncrement = 10
	}
	risk.riskFloor += riskFloorIncrement

	// 风险基线上限
	maxRiskFloor := 50
	if risk.riskFloor > maxRiskFloor {
		risk.riskFloor = maxRiskFloor
	}

	// 探测检测: 短时间内多个不同 payload 变体
	if payloadHash != "" {
		risk.payloadHashes = append(risk.payloadHashes, payloadHash)
		if len(risk.payloadHashes) > 20 {
			risk.payloadHashes = risk.payloadHashes[1:] // 保留最近 20 个
		}

		// 5 分钟内 10+ 个不同 payload
		if len(risk.payloadHashes) >= 10 {
			uniqueCount := countUnique(risk.payloadHashes)
			if uniqueCount >= 8 {
				risk.probeDetected = true
			}
		}
	}

	return risk.riskFloor
}

// GetTokenRiskFloor 获取 token 当前的风险基线分
func GetTokenRiskFloor(tokenId int) int {
	if tokenId <= 0 {
		return 0
	}

	tracker := globalTokenRiskTracker
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	risk := tracker.risks[tokenId]
	if risk == nil {
		return 0
	}

	// 检查是否需要衰减
	now := time.Now()
	if now.Sub(risk.lastDecayTime) > tracker.slidingWindow {
		// 在读锁中不能修改,返回衰减后的值
		return int(float64(risk.riskFloor) * tracker.decayRate)
	}

	return risk.riskFloor
}

// GetTokenTriggerCount 获取 token 在滑动窗口内的触发次数
func GetTokenTriggerCount(tokenId int) int {
	if tokenId <= 0 {
		return 0
	}

	tracker := globalTokenRiskTracker
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	risk := tracker.risks[tokenId]
	if risk == nil {
		return 0
	}

	// 滑动窗口外的触发不计数
	if time.Since(risk.lastTriggerTime) > tracker.slidingWindow {
		return 0
	}

	return risk.triggerCount
}

// IsTokenProbing 判断 token 是否在进行规则探测
func IsTokenProbing(tokenId int) bool {
	if tokenId <= 0 {
		return false
	}

	tracker := globalTokenRiskTracker
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	risk := tracker.risks[tokenId]
	if risk == nil {
		return false
	}

	return risk.probeDetected
}

// ShouldEscalateMode 判断是否应该升级该 token 的检测模式
func ShouldEscalateMode(tokenId int, threshold int) bool {
	if tokenId <= 0 {
		return false
	}

	triggerCount := GetTokenTriggerCount(tokenId)
	if threshold <= 0 {
		threshold = 10 // 默认阈值
	}

	return triggerCount >= threshold
}

// CleanupExpiredRisks 清理过期的风险记录 (后台任务)
func CleanupExpiredRisks() {
	tracker := globalTokenRiskTracker
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	now := time.Now()
	expiredTokens := make([]int, 0)

	for tokenId, risk := range tracker.risks {
		// 1 小时未触发则清理
		if now.Sub(risk.lastTriggerTime) > time.Hour {
			expiredTokens = append(expiredTokens, tokenId)
		}
	}

	for _, tokenId := range expiredTokens {
		delete(tracker.risks, tokenId)
	}
}

// GetTokenRiskStats 获取 token 风险统计 (用于监控)
func GetTokenRiskStats(tokenId int) map[string]interface{} {
	if tokenId <= 0 {
		return nil
	}

	tracker := globalTokenRiskTracker
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	risk := tracker.risks[tokenId]
	if risk == nil {
		return map[string]interface{}{
			"risk_floor":     0,
			"trigger_count":  0,
			"probe_detected": false,
		}
	}

	return map[string]interface{}{
		"risk_floor":        risk.riskFloor,
		"trigger_count":     risk.triggerCount,
		"last_trigger_time": risk.lastTriggerTime.Unix(),
		"probe_detected":    risk.probeDetected,
		"payload_count":     len(risk.payloadHashes),
	}
}

// countUnique 计算切片中唯一元素数量
func countUnique(slice []string) int {
	unique := make(map[string]bool)
	for _, item := range slice {
		unique[item] = true
	}
	return len(unique)
}

// InitTokenRiskCleanupJob 初始化后台清理任务
func InitTokenRiskCleanupJob() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			CleanupExpiredRisks()
		}
	}()
}
