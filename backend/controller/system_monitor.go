package controller

import (
	"runtime"
	"time"

	"STfreApi/common"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

func GetSystemMonitor(c *gin.Context) {
	// CPU 使用率
	cpuPercent, err := cpu.Percent(time.Second, false)
	cpuUsage := 0.0
	if err == nil && len(cpuPercent) > 0 {
		cpuUsage = cpuPercent[0]
	}

	// 内存信息
	vmStat, err := mem.VirtualMemory()
	memoryUsage := 0.0
	memoryTotal := uint64(0)
	memoryUsed := uint64(0)
	if err == nil {
		memoryUsage = vmStat.UsedPercent
		memoryTotal = vmStat.Total
		memoryUsed = vmStat.Used
	}

	common.OK(c, gin.H{
		"cpu_usage":     cpuUsage,
		"memory_usage":  memoryUsage,
		"memory_total":  memoryTotal,
		"memory_used":   memoryUsed,
		"goroutines":    runtime.NumGoroutine(),
		"timestamp":     time.Now().Unix(),
	})
}
