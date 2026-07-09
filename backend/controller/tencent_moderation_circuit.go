package controller

// ～天御审核专用极简熔断器～
// 复刻 qingyuan/circuit_breaker.go 的三态状态机核心，但去掉 policy 维度：
// 天御审核只有一份全局配置，不存在"策略"这个维度，一个包级单例就够。
//
// 目的：审核渠道连续失败时自动打开熔断，短时间内退化为本地关键词兜底审核，
// 而不是每次都硬等超时再 fail-open（既拖慢正常请求，又在故障期完全放弃审核）。

import (
	"sync"
	"time"
)

const (
	tmsCircuitClosed   = "closed"
	tmsCircuitOpen     = "open"
	tmsCircuitHalfOpen = "half_open"

	tmsFailureThreshold = 5  // 连续失败达此次数打开熔断
	tmsCooldownSeconds  = 30 // 熔断打开后的冷却时间
)

var tmsCircuit = struct {
	mu          sync.Mutex
	state       string
	failures    int
	openedUntil time.Time
}{state: tmsCircuitClosed}

// tmsCircuitState 返回当前熔断状态；open 且冷却已过则转 half_open（放一个探测请求过去）
func tmsCircuitState() string {
	tmsCircuit.mu.Lock()
	defer tmsCircuit.mu.Unlock()
	if tmsCircuit.state == tmsCircuitOpen && time.Now().After(tmsCircuit.openedUntil) {
		tmsCircuit.state = tmsCircuitHalfOpen
	}
	return tmsCircuit.state
}

// tmsRecordSuccess 审核调用成功，清零失败计数并闭合熔断
func tmsRecordSuccess() {
	tmsCircuit.mu.Lock()
	defer tmsCircuit.mu.Unlock()
	tmsCircuit.state = tmsCircuitClosed
	tmsCircuit.failures = 0
	tmsCircuit.openedUntil = time.Time{}
}

// tmsRecordFailure 审核调用失败，累计到阈值则打开熔断
func tmsRecordFailure() {
	tmsCircuit.mu.Lock()
	defer tmsCircuit.mu.Unlock()
	tmsCircuit.failures++
	if tmsCircuit.failures >= tmsFailureThreshold {
		tmsCircuit.state = tmsCircuitOpen
		tmsCircuit.openedUntil = time.Now().Add(tmsCooldownSeconds * time.Second)
	}
}
