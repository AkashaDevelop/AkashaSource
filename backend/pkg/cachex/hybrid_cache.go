package cachex

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// HybridCache 是 Redis 优先、进程内 map 兜底的混合缓存。
// 泛型参数 T 允许缓存任意可 JSON 序列化的值类型。
type HybridCache[T any] struct {
	namespace    string
	redis        *redis.Client
	redisEnabled func() bool
	memory       map[string]cacheEntry[T]
	mu           sync.RWMutex
	maxEntries   int
	defaultTTL   time.Duration
}

type cacheEntry[T any] struct {
	value    T
	expireAt int64
}

func NewHybridCache[T any](namespace string, redis *redis.Client, redisEnabled func() bool, maxEntries int, defaultTTL time.Duration) *HybridCache[T] {
	cache := &HybridCache[T]{
		namespace:    namespace,
		redis:        redis,
		redisEnabled: redisEnabled,
		memory:       make(map[string]cacheEntry[T]),
		maxEntries:   maxEntries,
		defaultTTL:   defaultTTL,
	}
	go cache.cleanup()
	return cache
}

func (c *HybridCache[T]) Get(key string) (T, bool) {
	fullKey := c.namespace + ":" + key

	if c.redisEnabled != nil && c.redisEnabled() && c.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		val, err := c.redis.Get(ctx, fullKey).Result()
		if err == nil {
			var result T
			if json.Unmarshal([]byte(val), &result) == nil {
				return result, true
			}
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.memory[key]
	if !ok || time.Now().Unix() > entry.expireAt {
		var zero T
		return zero, false
	}
	return entry.value, true
}

func (c *HybridCache[T]) Set(key string, value T, ttl time.Duration) {
	fullKey := c.namespace + ":" + key

	if c.redisEnabled != nil && c.redisEnabled() && c.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		data, _ := json.Marshal(value)
		c.redis.Set(ctx, fullKey, string(data), ttl)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.memory) >= c.maxEntries {
		c.evictOldest()
	}

	c.memory[key] = cacheEntry[T]{
		value:    value,
		expireAt: time.Now().Add(ttl).Unix(),
	}
}

func (c *HybridCache[T]) evictOldest() {
	var oldestKey string
	var oldestTime int64 = time.Now().Unix()

	for k, v := range c.memory {
		if v.expireAt < oldestTime {
			oldestTime = v.expireAt
			oldestKey = k
		}
	}

	if oldestKey != "" {
		delete(c.memory, oldestKey)
	}
}

func (c *HybridCache[T]) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now().Unix()
		for k, v := range c.memory {
			if now > v.expireAt {
				delete(c.memory, k)
			}
		}
		c.mu.Unlock()
	}
}

func (c *HybridCache[T]) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.memory))
	for k := range c.memory {
		keys = append(keys, k)
	}
	return keys
}

func (c *HybridCache[T]) DeleteMany(keys []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, k := range keys {
		delete(c.memory, k)
	}

	if c.redisEnabled != nil && c.redisEnabled() && c.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		fullKeys := make([]string, len(keys))
		for i, k := range keys {
			fullKeys[i] = c.namespace + ":" + k
		}
		c.redis.Del(ctx, fullKeys...)
	}
}
