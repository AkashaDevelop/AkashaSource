package controller

import (
	"fmt"
	"runtime"
	"time"

	"STfreApi/common"

	"github.com/gin-gonic/gin"
)

func GetPerformance(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Now().Unix() - common.StartTime
	uptimeStr := fmt.Sprintf("%dd %dh %dm", uptime/86400, (uptime%86400)/3600, (uptime%3600)/60)

	common.OK(c, gin.H{
		"goroutines": runtime.NumGoroutine(),
		"memory_mb":  m.Alloc / 1024 / 1024,
		"gc_cycles":  m.NumGC,
		"uptime":     uptimeStr,
		"go_version": runtime.Version(),
	})
}
