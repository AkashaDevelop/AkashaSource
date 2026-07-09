package main

import (
	"flag"
	"fmt"
	"log"

	"STfreApi/common"
	"STfreApi/middleware"
	"STfreApi/router"
	"STfreApi/service/bootstrap"
	"STfreApi/web"

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

	// 初始化 Redis（不依赖数据库，永远可以跑）
	common.InitRedis()

	// Initialize Rate Limiter（纯内存实现，不依赖数据库）
	log.Printf("初始化限流器，RPM: %d", rpm)
	middleware.InitRateLimiter(rpm)

	// 宸汐清源姊妹功能～引导式初始化：优先用向导持久化过的连接信息，
	// 没有的话才退回命令行参数；连不上也不 Fatal，让 API 先跑起来，
	// 好让向导页面能调到 /api/setup/database 把数据库接上
	effectiveDriver, effectiveDSN := driver, dsn
	if cfg, ok := common.LoadDBConfig(); ok {
		effectiveDriver, effectiveDSN = cfg.Driver, cfg.DSN
		log.Printf("检测到已保存的数据库配置，使用驱动: %s", effectiveDriver)
	}
	log.Printf("初始化数据库，驱动: %s", effectiveDriver)
	if err := common.InitDB(effectiveDriver, effectiveDSN); err != nil {
		log.Printf("数据库连接失败，等待通过初始化向导配置: %v", err)
	} else {
		bootstrap.RunPostDBInit()
	}

	// Initialize Gin
	r := gin.Default()
	if err := r.SetTrustedProxies(common.TrustedProxies()); err != nil {
		log.Fatalf("受信任代理配置有误（AKASHA_TRUSTED_PROXIES）: %v", err)
	}

	// Set API Router
	router.SetApiRouter(r)

	// Basic Ping Route
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Akasha is running",
			"driver":  effectiveDriver,
		})
	})

	// 单文件部署：把嵌入二进制的前端静态产物挂到 NoRoute 兜底，
	// 所有未命中 API 的请求交给前端（静态资源直出 / SPA 路由回退 index.html）
	web.RegisterFrontend(r)

	// Start Server
	addr := fmt.Sprintf(":%d", port)
	log.Printf("服务启动中: %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
