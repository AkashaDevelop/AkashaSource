package xuanjian

// 宸汐玄鉴·启动与清理 (づ｡◕‿‿◕｡)づ
// 在 bootstrap.go 里调用 Init() 就行啦，
// 后台清理任务会自动每10分钟跑一次，
// 不活跃的画像乖乖清掉，内存不会无限涨哦。

import (
	"log"
	"time"
)

// Init 初始化宸汐玄鉴模块（从 DB options 加载配置）
func Init() {
	LoadConfig()

	// 首次启动播种内置规则（无论模块是否启用，都要让超管在管理页里看得到规则列表）
	if err := SeedBuiltinRules(); err != nil {
		log.Printf("[xuanjian] 播种内置规则失败: %v", err)
	}
	if err := ReloadRuleCache(); err != nil {
		log.Printf("[xuanjian] 加载规则缓存失败: %v", err)
	}

	cfg, enabled := GetConfig()
	if !enabled || cfg.Mode == ModeOff {
		log.Printf("[xuanjian] 宸汐玄鉴已禁用（超管可在系统设置中开启）")
		return
	}

	log.Printf("[xuanjian] 宸汐玄鉴已启动，模式: %s", cfg.Mode)

	// 启动后台画像清理任务
	go startCleanup()
}

func startCleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		CleanupExpired()
	}
}

// GetVersion 返回模块版本
func GetVersion() string {
	return "1.0.0"
}
