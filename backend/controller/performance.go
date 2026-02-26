package controller

import (
	"net/http"
	"runtime"
	"time"

	"STfreApi/common"

	"github.com/gin-gonic/gin"
)

func GetPerformance(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	c.JSON(http.StatusOK, gin.H{
		"goroutines":   runtime.NumGoroutine(),
		"memory_alloc": m.Alloc / 1024 / 1024,       // MB
		"memory_sys":   m.Sys / 1024 / 1024,          // MB
		"gc_cycles":    m.NumGC,
		"uptime":       time.Now().Unix() - common.StartTime,
		"go_version":   runtime.Version(),
	})
}
