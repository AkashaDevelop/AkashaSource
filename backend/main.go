package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"time"

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

	// 生产模式：关闭 gin 调试日志
	gin.SetMode(gin.ReleaseMode)

	// 初始化 JWT 密钥（环境变量优先，否则复用/生成本地密钥文件）
	common.InitJwtSecret()

	// 初始化 Redis（不依赖数据库，永远可以跑）
	common.InitRedis()

	// Initialize Rate Limiter（纯内存实现，不依赖数据库）
	middleware.InitRateLimiter(rpm)

	// 引导式初始化：只有 dbconfig.json 存在时才连接数据库，
	// 首次启动（无配置文件）不自动用默认 SQLite 连接，等用户通过向导选择数据库
	effectiveDriver, effectiveDSN := "", ""
	if cfg, ok := common.LoadDBConfig(); ok {
		effectiveDriver, effectiveDSN = cfg.Driver, cfg.DSN
	}

	// 启动横幅（在确定 driver 之后打印）
	printBanner(port, effectiveDriver)

	log.Printf("[Init] 限流器 RPM: %d", rpm)
	if effectiveDriver == "" {
		log.Printf("[Init] 未检测到数据库配置，等待通过初始化向导配置")
	} else {
		log.Printf("[Init] 数据库驱动: %s", effectiveDriver)
		if err := common.InitDB(effectiveDriver, effectiveDSN); err != nil {
			log.Printf("[Init] 数据库连接失败，等待通过初始化向导配置: %v", err)
		} else {
			bootstrap.RunPostDBInit()
		}
	}

	// Initialize Gin
	r := gin.New()
	r.Use(gin.Recovery())
	if err := r.SetTrustedProxies(common.TrustedProxies()); err != nil {
		log.Fatalf("[Init] 受信任代理配置有误（AKASHA_TRUSTED_PROXIES）: %v", err)
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
	log.Printf("[Server] 监听 %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("[Server] 启动失败: %v", err)
	}
}

func printBanner(port int, driver string) {
	line := "───────────────────────────────────────────────────────────────"
	now := time.Now().Format("2006-01-02 15:04:05")

	fmt.Println()
	fmt.Println(line)
	fmt.Println("     _    _      _                   _           _       ")
	fmt.Println("    / \\  | | ___| |_ _ __ ___  _ __ | |_ ___  __| |_   _ ")
	fmt.Println("   / _ \\ | |/ _ \\ __| '__/ _ \\| '_ \\| __/ _ \\/ _` | | | |")
	fmt.Println("  / ___ \\| |  __/ |_| | | (_) | |_) | ||  __/ (_| | |_| |")
	fmt.Println(" /_/   \\_\\_|\\___|\\__|_|  \\___/| .__/ \\__\\___|\\__,_|\\__, |")
	fmt.Println("                              |_|                  |___/ ")
	fmt.Println()
	fmt.Printf("  [ Version ]  %s\n", common.Version)
	fmt.Printf("  [ Runtime ]  Go %s  %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  [ Listen ]   http://0.0.0.0:%d\n", port)
	fmt.Printf("  [ Driver ]   %s\n", driver)
	fmt.Printf("  [ Started ]  %s\n", now)
	fmt.Println()
	fmt.Println("  Security:  CxSec (ECDH+AES-256-GCM)  ·  QingYuan  ·  XuanJian")
	fmt.Println(line)
	fmt.Println()
}
