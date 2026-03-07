package service

import (
	"runtime"
	"time"
)

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

	return SystemStats{
		CPUUsage:    0, // CPU usage requires external library
		MemoryUsage: float64(m.Alloc) / float64(m.Sys) * 100,
		MemoryTotal: m.Sys,
		MemoryUsed:  m.Alloc,
		Goroutines:  runtime.NumGoroutine(),
		Timestamp:   time.Now().Unix(),
	}
}
