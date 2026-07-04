package main

import (
	"flag"
	"fmt"
	"log"

	"STfreApi/common"
	"STfreApi/middleware"
	"STfreApi/model"
	"STfreApi/router"
	"STfreApi/service"
	"STfreApi/service/cxsec"
	"STfreApi/service/qingyuan"

	"github.com/gin-gonic/gin"
)

func main() {
	// Parse command line arguments
	var port int
	var driver string
	var dsn string
	var rpm int

	flag.IntVar(&port, "port", 8080, "Server port")
	flag.StringVar(&driver, "driver", "sqlite", "Database driver (sqlite, mysql, postgres)")
	flag.StringVar(&dsn, "dsn", "akasha.db", "Database DSN")
	flag.IntVar(&rpm, "rpm", 60, "Rate limit (requests per minute)")
	flag.Parse()

	// 初始化 JWT 密钥（环境变量优先，否则复用/生成本地密钥文件）
	common.InitJwtSecret()

	// Initialize Database
	log.Printf("初始化数据库，驱动: %s", driver)
	common.InitDB(driver, dsn)

	// Load Options
	model.InitOptions()

	// 初始化 Redis
	common.InitRedis()

	// Initialize Rate Limiter
	log.Printf("初始化限流器，RPM: %d", rpm)
	middleware.InitRateLimiter(rpm)

	// Start Channel Health Check
	log.Printf("启动渠道健康检查...")
	service.StartChannelCheck()

	// 启动日志队列
	service.InitLogQueue()

	// 启动订阅过期检查
	service.InitSubscriptionExpiry()

	// 启动渠道签到和余额监控调度器
	log.Printf("启动渠道签到和余额监控调度器...")
	service.InitScheduler()

	// 启动宸汐御安全挑战清理 goroutine
	go cxsec.PurgeExpiredChallenges()

	// 唤醒「宸汐清源」上下文净化结界～守护每一次对话不被杂念侵扰哦 (｡•ᴗ•｡)
	log.Printf("初始化宸汐清源模块...")
	if err := qingyuan.Init(); err != nil {
		log.Fatalf("宸汐清源模块初始化失败: %v", err)
	}
	features := qingyuan.GetFeatures()
	enabledCount := 0
	for _, enabled := range features {
		if enabled {
			enabledCount++
		}
	}
	log.Printf("宸汐清源已启用 %d 个功能 (版本: %s)", enabledCount, qingyuan.GetVersion())

	// Initialize Gin
	r := gin.Default()

	// Set API Router
	router.SetApiRouter(r)

	// Basic Ping Route
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Akasha is running",
			"driver":  driver,
		})
	})

	// Start Server
	addr := fmt.Sprintf(":%d", port)
	log.Printf("服务启动中: %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
