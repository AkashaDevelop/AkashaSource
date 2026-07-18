package middleware

// 🌸 分组三维速率限制：RPM（每分钟请求）/ TPM（每分钟 token）/ RPD（每日请求）～
// 对齐 new-api 的按分组限流体系，Redis 优先、内存兜底

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"STfreApi/common"
	"STfreApi/service"
)

// ～带自定义窗口的计数器（内存版）～
type windowCounter struct {
	count int64
	reset int64 // Unix 秒
	mu    sync.Mutex
}

var groupCounters sync.Map

func init() {
	// ～每 10 分钟清扫过期计数器，别让内存悄悄长胖～
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			now := time.Now().Unix()
			groupCounters.Range(func(key, value interface{}) bool {
				wc := value.(*windowCounter)
				wc.mu.Lock()
				if now > wc.reset+86400 {
					groupCounters.Delete(key)
				}
				wc.mu.Unlock()
				return true
			})
		}
	}()
}

// addToWindow 在窗口内累加 amount 并返回累加后的值；窗口过期自动重置～
func addToWindow(key string, amount int64, window time.Duration) int64 {
	if common.RedisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		count, err := common.RedisClient.IncrBy(ctx, key, amount).Result()
		if err != nil {
			return 0 // Redis 出错时放行，别把正常流量误伤了
		}
		if count == amount {
			common.RedisClient.Expire(ctx, key, window)
		}
		return count
	}

	val, _ := groupCounters.LoadOrStore(key, &windowCounter{reset: time.Now().Add(window).Unix()})
	wc := val.(*windowCounter)
	wc.mu.Lock()
	defer wc.mu.Unlock()
	now := time.Now().Unix()
	if now > wc.reset {
		wc.count = 0
		wc.reset = time.Now().Add(window).Unix()
	}
	wc.count += amount
	return wc.count
}

// peekWindow 只读当前窗口累计值，不改动～
func peekWindow(key string) int64 {
	if common.RedisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		v, err := common.RedisClient.Get(ctx, key).Int64()
		if err != nil {
			return 0
		}
		return v
	}
	val, ok := groupCounters.Load(key)
	if !ok {
		return 0
	}
	wc := val.(*windowCounter)
	wc.mu.Lock()
	defer wc.mu.Unlock()
	if time.Now().Unix() > wc.reset {
		return 0
	}
	return atomic.LoadInt64(&wc.count)
}

// ～按天分桶的 key，天变了自然换新桶～
func dayKey(prefix, group string) string {
	return fmt.Sprintf("%s:%s:%d", prefix, group, time.Now().Unix()/86400)
}

func minuteKey(prefix, group string) string {
	return fmt.Sprintf("%s:%s", prefix, group)
}

// GroupRateLimitMiddleware 按分组三维限流：RPM / TPM / RPD，任一超限即拦截～
// 返回 false 表示超限（调用方负责写响应）。各维度 <=0 都表示不限。
func GroupRateLimitMiddleware(groupName string) bool {
	if groupName == "" {
		return true
	}
	rpm, tpm, rpd := service.GetGroupRateLimits(groupName)
	if rpm <= 0 && tpm <= 0 && rpd <= 0 {
		return true
	}

	// RPD：每日请求数
	if rpd > 0 {
		if addToWindow(dayKey("grpd", groupName), 1, 24*time.Hour) > int64(rpd) {
			return false
		}
	}

	// RPM：每分钟请求数
	if rpm > 0 {
		if addToWindow(minuteKey("grpm", groupName), 1, time.Minute) > int64(rpm) {
			return false
		}
	}

	// TPM：每分钟 token 数（token 用量由 RecordGroupTokens 在请求完成后累加，
	// 这里只检查当前分钟已消耗量是否已经爆表）
	if tpm > 0 {
		if peekWindow(minuteKey("gtpm", groupName)) >= int64(tpm) {
			return false
		}
	}

	return true
}

// RecordGroupTokens 请求完成后把本次消耗的 token 数记入分组的 TPM 窗口～
// 在 RecordConsumeLog 里调用。
func RecordGroupTokens(groupName string, tokens int) {
	if groupName == "" || tokens <= 0 {
		return
	}
	// 只有配置了 TPM 的分组才需要记账，省一次无谓的计数～
	_, tpm, _ := service.GetGroupRateLimits(groupName)
	if tpm <= 0 {
		return
	}
	addToWindow(minuteKey("gtpm", groupName), int64(tokens), time.Minute)
}
