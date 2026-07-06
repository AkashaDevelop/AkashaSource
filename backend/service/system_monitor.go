package service

import (
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

// 后台定期采样 CPU 使用率，避免在请求路径上阻塞。
var (
	cpuUsage   float64
	cpuUsageMu sync.RWMutex
)

func init() {
	go func() {
		// 首次采样需要间隔，gopsutil 通过两次采样间隔计算使用率
		_, _ = cpu.Percent(time.Second, false)
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			if percent, err := cpu.Percent(time.Second, false); err == nil && len(percent) > 0 {
				cpuUsageMu.Lock()
				cpuUsage = percent[0]
				cpuUsageMu.Unlock()
			}
		}
	}()
}

type SystemStats struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	MemoryTotal uint64  `json:"memory_total"`
	MemoryUsed  uint64  `json:"memory_used"`
	Goroutines  int     `json:"goroutines"`
	Timestamp   int64   `json:"timestamp"`
}

func GetSystemStats() SystemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	cpuUsageMu.RLock()
	cpu := cpuUsage
	cpuUsageMu.RUnlock()

	return SystemStats{
		CPUUsage:    cpu,
		MemoryUsage: float64(m.Alloc) / float64(m.Sys) * 100,
		MemoryTotal: m.Sys,
		MemoryUsed:  m.Alloc,
		Goroutines:  runtime.NumGoroutine(),
		Timestamp:   time.Now().Unix(),
	}
}
