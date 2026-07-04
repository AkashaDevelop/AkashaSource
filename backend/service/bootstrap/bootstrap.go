package bootstrap

// 宸汐清源姊妹模块～数据库连上之后才能开工的那些活儿，都收在这里啦 (・ω・)ノ
// 之所以单独拎出来，是因为它既要在正常启动时跑一遍，也要在"引导式初始化"向导
// 把数据库连上的那一刻被 controller 层再跑一遍——用 sync.Once 兜底，
// 不管从哪条路径触发，这堆初始化只会真正执行一次。

import (
	"log"
	"sync"

	"STfreApi/model"
	"STfreApi/service"
	"STfreApi/service/cxsec"
	"STfreApi/service/qingyuan"
)

var once sync.Once

// RunPostDBInit 执行所有依赖数据库连接的初始化步骤
// 可以在正常启动流程（main.go）里调用，也可以在引导向导的
// "数据库连接成功"处理器里调用，重复调用只会生效一次
func RunPostDBInit() {
	once.Do(func() {
		// Load Options
		model.InitOptions()

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
			log.Printf("宸汐清源模块初始化失败: %v", err)
			return
		}
		features := qingyuan.GetFeatures()
		enabledCount := 0
		for _, enabled := range features {
			if enabled {
				enabledCount++
			}
		}
		log.Printf("宸汐清源已启用 %d 个功能 (版本: %s)", enabledCount, qingyuan.GetVersion())
	})
}
