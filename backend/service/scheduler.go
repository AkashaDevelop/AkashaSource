package service

import (
	"STfreApi/common"
	"log"
	"sync"
	"time"
)

type SchedulerConfig struct {
	CheckinCron         string
	BalanceRefreshCron  string
	CheckinIntervalHours int
}

var (
	schedulerOnce       sync.Once
	schedulerRunning    bool
	schedulerMutex      sync.Mutex
	checkinTicker       *time.Ticker
	balanceTicker       *time.Ticker
	stopChan            chan struct{}
	currentConfig       SchedulerConfig
)

func InitScheduler() {
	schedulerOnce.Do(func() {
		schedulerRunning = false
		stopChan = make(chan struct{})
		currentConfig = SchedulerConfig{
			CheckinCron:         "0 8 * * *",
			BalanceRefreshCron:  "0 */6 * * *",
			CheckinIntervalHours: 24,
		}
		loadSchedulerConfig()
		startScheduler()
	})
}

func loadSchedulerConfig() {
	common.OptionLock.RLock()
	defer common.OptionLock.RUnlock()

	if cron, ok := common.OptionMap["checkin_cron"]; ok && cron != "" {
		currentConfig.CheckinCron = cron
	}
	if cron, ok := common.OptionMap["balance_refresh_cron"]; ok && cron != "" {
		currentConfig.BalanceRefreshCron = cron
	}
	if hours, ok := common.OptionMap["checkin_interval_hours"]; ok && hours != "" {
		if h := parseIntDefault(hours, 24); h > 0 {
			currentConfig.CheckinIntervalHours = h
		}
	}
}

func parseIntDefault(s string, def int) int {
	if _, err := time.ParseDuration(s + "h"); err == nil {
		if n, err := parseInt(s); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n, nil
}

func startScheduler() {
	schedulerMutex.Lock()
	defer schedulerMutex.Unlock()

	if schedulerRunning {
		return
	}

	schedulerRunning = true
	log.Println("[调度器] 启动渠道签到和余额监控调度器...")

	go runScheduler()
}

func StartScheduler() {
	startScheduler()
}

func runScheduler() {
	checkinInterval := time.Duration(currentConfig.CheckinIntervalHours) * time.Hour
	balanceInterval := 6 * time.Hour

	checkinTicker = time.NewTicker(checkinInterval)
	balanceTicker = time.NewTicker(balanceInterval)

	defer func() {
		checkinTicker.Stop()
		balanceTicker.Stop()
	}()

	log.Printf("[调度器] 签到间隔: %v, 余额刷新间隔: %v", checkinInterval, balanceInterval)

	for {
		select {
		case <-stopChan:
			log.Println("[调度器] 调度器已停止")
			return
		case <-checkinTicker.C:
			log.Println("[调度器] 执行定时签到任务...")
			go func() {
				results, err := CheckinAllChannels()
				if err != nil {
					log.Printf("[调度器] 签到任务失败: %v", err)
					return
				}
				success := 0
				failed := 0
				for _, r := range results {
					if r["success"] == true {
						success++
					} else {
						failed++
					}
				}
				log.Printf("[调度器] 签到任务完成: 成功 %d, 失败 %d", success, failed)
			}()
		case <-balanceTicker.C:
			log.Println("[调度器] 执行定时余额刷新任务...")
			go func() {
				results, err := RefreshAllChannelBalances()
				if err != nil {
					log.Printf("[调度器] 余额刷新任务失败: %v", err)
					return
				}
				success := 0
				failed := 0
				for _, r := range results {
					if r["success"] == true {
						success++
					} else {
						failed++
					}
				}
				log.Printf("[调度器] 余额刷新任务完成: 成功 %d, 失败 %d", success, failed)
			}()
		}
	}
}

func StopScheduler() {
	schedulerMutex.Lock()
	defer schedulerMutex.Unlock()

	if !schedulerRunning {
		return
	}

	close(stopChan)
	schedulerRunning = false
	stopChan = make(chan struct{})
}

func UpdateSchedulerConfig(config SchedulerConfig) {
	schedulerMutex.Lock()
	defer schedulerMutex.Unlock()

	if config.CheckinCron != "" {
		currentConfig.CheckinCron = config.CheckinCron
	}
	if config.BalanceRefreshCron != "" {
		currentConfig.BalanceRefreshCron = config.BalanceRefreshCron
	}
	if config.CheckinIntervalHours > 0 {
		currentConfig.CheckinIntervalHours = config.CheckinIntervalHours
	}

	if schedulerRunning {
		checkinInterval := time.Duration(currentConfig.CheckinIntervalHours) * time.Hour
		checkinTicker.Reset(checkinInterval)
		log.Printf("[调度器] 更新签到间隔: %v", checkinInterval)
	}
}

func GetSchedulerStatus() map[string]interface{} {
	schedulerMutex.Lock()
	defer schedulerMutex.Unlock()

	return map[string]interface{}{
		"running":                schedulerRunning,
		"checkin_cron":           currentConfig.CheckinCron,
		"balance_refresh_cron":   currentConfig.BalanceRefreshCron,
		"checkin_interval_hours": currentConfig.CheckinIntervalHours,
	}
}

func TriggerCheckin() (map[string]interface{}, error) {
	results, err := CheckinAllChannels()
	if err != nil {
		return nil, err
	}
	success := 0
	failed := 0
	skipped := 0
	for _, r := range results {
		if r["success"] == true {
			success++
		} else {
			if msg, ok := r["message"].(string); ok && (msg == "今日已签到" || msg == "站点不支持签到接口") {
				skipped++
			} else {
				failed++
			}
		}
	}
	return map[string]interface{}{
		"total":   len(results),
		"success": success,
		"failed":  failed,
		"skipped": skipped,
		"results": results,
	}, nil
}

func TriggerBalanceRefresh() (map[string]interface{}, error) {
	results, err := RefreshAllChannelBalances()
	if err != nil {
		return nil, err
	}
	success := 0
	failed := 0
	for _, r := range results {
		if r["success"] == true {
			success++
		} else {
			failed++
		}
	}
	return map[string]interface{}{
		"total":   len(results),
		"success": success,
		"failed":  failed,
		"results": results,
	}, nil
}
