package common

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// JWT 是无状态的，登出后原 token 在到期前仍然有效。
// 这里维护一个「黑名单」：登出时把 token 哈希记进去，直到 token 自身过期为止。
// 有 Redis 就用 Redis（天然带 TTL、多实例共享），没有就退化为进程内 map。

var (
	blacklistStore = map[string]time.Time{}
	blacklistMu    sync.Mutex
)

func tokenBlacklistKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "jwt:blacklist:" + hex.EncodeToString(sum[:])
}

// BlacklistToken 将 token 加入黑名单，直到其自身过期时间为止
func BlacklistToken(token string, expiresAt time.Time) {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return
	}
	key := tokenBlacklistKey(token)
	if RedisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		RedisClient.Set(ctx, key, "1", ttl)
		return
	}
	blacklistMu.Lock()
	blacklistStore[key] = expiresAt
	blacklistMu.Unlock()
}

// IsTokenBlacklisted 检查 token 是否已被拉黑
func IsTokenBlacklisted(token string) bool {
	key := tokenBlacklistKey(token)
	if RedisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		exists, err := RedisClient.Exists(ctx, key).Result()
		return err == nil && exists > 0
	}
	blacklistMu.Lock()
	defer blacklistMu.Unlock()
	expiresAt, ok := blacklistStore[key]
	if !ok {
		return false
	}
	if time.Now().After(expiresAt) {
		delete(blacklistStore, key)
		return false
	}
	return true
}
