package xuanjian

// 宸汐玄鉴 · 传令节流阀 (｡•́︿•̀｡)
//
// notifyAdmin 每被调用一次，就要查一遍全部 root 用户、再逐人发一封站内信/邮件。
// 而检测是**每个请求**都跑的——一个持续触发规则的 token，能在一分钟里把同一条告警
// 重复发上几百遍，管理员的收件箱直接被自己人 DDoS 了。
//
// 这里加一道节流：同一个 (token, finding 类型) 组合在冷却期内只放行第一条。
// 被压下去的那些不会丢——事件表里每一条都有完整记录，只是不再重复打扰人类喵～

import (
	"sync"
	"time"
)

// notifyCooldown 同一 (token, findingType) 的通知冷却时间
const notifyCooldown = 10 * time.Minute

// notifyThrottleCleanupAfter 清理多久没再触发的节流记录
const notifyThrottleCleanupAfter = time.Hour

type notifyThrottle struct {
	mu   sync.Mutex
	last map[string]time.Time
}

var globalNotifyThrottle = &notifyThrottle{last: make(map[string]time.Time)}

// allowNotify 判断这条告警现在该不该真的发出去
// 返回 true 表示放行，false 表示还在冷却期内、静默跳过
func allowNotify(tokenID int, findingType string) bool {
	key := findingType + "#" + intStr(tokenID)
	now := time.Now()

	globalNotifyThrottle.mu.Lock()
	defer globalNotifyThrottle.mu.Unlock()

	if last, ok := globalNotifyThrottle.last[key]; ok && now.Sub(last) < notifyCooldown {
		return false
	}
	globalNotifyThrottle.last[key] = now
	return true
}

// cleanupNotifyThrottle 清掉早就不再触发的节流记录，避免 map 无限长大
func cleanupNotifyThrottle() {
	cutoff := time.Now().Add(-notifyThrottleCleanupAfter)
	globalNotifyThrottle.mu.Lock()
	defer globalNotifyThrottle.mu.Unlock()
	for key, last := range globalNotifyThrottle.last {
		if last.Before(cutoff) {
			delete(globalNotifyThrottle.last, key)
		}
	}
}
