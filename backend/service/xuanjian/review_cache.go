package xuanjian

// 宸汐玄鉴 · 审核记忆匣 (◕‿◕)♡
//
// AI 预审是**同步阻塞**的：请求要一直等在这儿，直到审核模型回话才敢往下走。
// 于是每一次调用都实打实地加在用户感知的延迟上，还要额外付一份 token 钱。
//
// 可现实里重复内容多得惊人——用户改一版重发、SDK 自动重试、前端手滑点两下，
// 送过去的是一模一样的文本，审核模型也必然给出一模一样的结论。
// 那第二次之后就没必要再问了，记在这儿直接复用就好喵～
//
// 只缓存"结论明确"的结果：被跳过（渠道没配、超时、解析失败）的不缓存，
// 免得一次网络抖动导致后面几分钟的请求全部无审核裸奔。

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// reviewCacheTTL 审核结论的有效期
const reviewCacheTTL = 5 * time.Minute

// reviewCacheMaxEntries 缓存条目上限，超了就整体清空重来（简单有效，不用上 LRU）
const reviewCacheMaxEntries = 4096

type reviewCacheEntry struct {
	result   AIReviewResult
	cachedAt time.Time
}

type reviewCache struct {
	mu      sync.RWMutex
	entries map[string]reviewCacheEntry
}

var globalReviewCache = &reviewCache{entries: make(map[string]reviewCacheEntry)}

// reviewCacheKey 用「阶段 + 文本哈希」作为键
// 带上阶段是因为预审和复审用的提示词不同，结论不能互相串用
func reviewCacheKey(stage, text string) string {
	sum := sha256.Sum256([]byte(stage + "\x00" + text))
	return hex.EncodeToString(sum[:16])
}

// lookupReviewCache 查一条还没过期的审核结论
func lookupReviewCache(stage, text string) (AIReviewResult, bool) {
	key := reviewCacheKey(stage, text)

	globalReviewCache.mu.RLock()
	entry, ok := globalReviewCache.entries[key]
	globalReviewCache.mu.RUnlock()

	if !ok || time.Since(entry.cachedAt) > reviewCacheTTL {
		return AIReviewResult{}, false
	}
	return entry.result, true
}

// storeReviewCache 记住一条明确的审核结论
func storeReviewCache(stage, text string, result AIReviewResult) {
	// 跳过的结果不缓存——那代表"这次没审成"，不是"这内容没问题"
	if result.Skipped {
		return
	}

	key := reviewCacheKey(stage, text)

	globalReviewCache.mu.Lock()
	defer globalReviewCache.mu.Unlock()

	if len(globalReviewCache.entries) >= reviewCacheMaxEntries {
		globalReviewCache.entries = make(map[string]reviewCacheEntry, reviewCacheMaxEntries/2)
	}
	globalReviewCache.entries[key] = reviewCacheEntry{result: result, cachedAt: time.Now()}
}

// cleanupReviewCache 清掉过期条目（由 init.go 的后台清理任务定期调用）
func cleanupReviewCache() {
	cutoff := time.Now().Add(-reviewCacheTTL)
	globalReviewCache.mu.Lock()
	defer globalReviewCache.mu.Unlock()
	for key, entry := range globalReviewCache.entries {
		if entry.cachedAt.Before(cutoff) {
			delete(globalReviewCache.entries, key)
		}
	}
}
