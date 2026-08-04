package xuanjian

// 宸汐玄鉴·启动与清理 (づ｡◕‿‿◕｡)づ
// 在 bootstrap.go 里调用 Init() 就行啦，
// 后台清理任务会自动每10分钟跑一次，
// 不活跃的画像乖乖清掉，内存不会无限涨哦。

import (
	"log"
	"sync"
	"time"
)

// cleanupOnce 保证清理任务只启动一次（配置热更新会重复调用 Init 相关路径）
var cleanupOnce sync.Once

// Init 初始化宸汐玄鉴模块（从 DB options 加载配置）
func Init() {
	LoadConfig()

	// 首次启动播种内置规则（无论模块是否启用，都要让超管在管理页里看得到规则列表）
	if err := SeedBuiltinRules(); err != nil {
		log.Printf("[xuanjian] 播种内置规则失败: %v", err)
	}
	// 存量规则库校准（新装库不会触发，见 rules_recalibrate.go）
	RecalibrateRules()
	if err := ReloadRuleCache(); err != nil {
		log.Printf("[xuanjian] 加载规则缓存失败: %v", err)
	}

	// ～2026.8.4 修正：清理任务以前挂在"模块已启用"分支的最后一行，
	// 于是启动时是关闭状态、之后管理员在后台打开的话，清理协程永远不会起——
	// 画像只进不出，内存一路涨到天上去 (⊙_⊙;)
	// 现在无条件启动：模块没开时画像本来就是空的，多跑一个空转的 ticker 不花什么钱～
	cleanupOnce.Do(func() { go startCleanup() })

	cfg, enabled := GetConfig()
	if !enabled || cfg.Mode == ModeOff {
		log.Printf("[xuanjian] 宸汐玄鉴已禁用（超管可在系统设置中开启）")
		return
	}

	log.Printf("[xuanjian] 宸汐玄鉴已启动，模式: %s", cfg.Mode)
}

func startCleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		CleanupExpired()
		cleanupNotifyThrottle()
		cleanupReviewCache()
	}
}

// GetVersion 返回模块版本
func GetVersion() string {
	return "1.0.0"
}
