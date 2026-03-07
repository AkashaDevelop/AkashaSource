package cachex

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type HybridCache struct {
	namespace    string
	redis        *redis.Client
	redisEnabled func() bool
	memory       map[string]cacheEntry
	mu           sync.RWMutex
	maxEntries   int
	defaultTTL   time.Duration
}

type cacheEntry struct {
	value    int
	expireAt int64
}

func NewHybridCache(namespace string, redis *redis.Client, redisEnabled func() bool, maxEntries int, defaultTTL time.Duration) *HybridCache {
	cache := &HybridCache{
		namespace:    namespace,
		redis:        redis,
		redisEnabled: redisEnabled,
		memory:       make(map[string]cacheEntry),
		maxEntries:   maxEntries,
		defaultTTL:   defaultTTL,
	}
	go cache.cleanup()
	return cache
}

func (c *HybridCache) Get(key string) (int, bool) {
	fullKey := c.namespace + ":" + key

	if c.redisEnabled != nil && c.redisEnabled() && c.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		val, err := c.redis.Get(ctx, fullKey).Result()
		if err == nil {
			var result int
			if json.Unmarshal([]byte(val), &result) == nil {
				return result, true
			}
		}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.memory[key]
	if !ok || time.Now().Unix() > entry.expireAt {
		return 0, false
	}
	return entry.value, true
}

func (c *HybridCache) Set(key string, value int, ttl time.Duration) {
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

	c.memory[key] = cacheEntry{
		value:    value,
		expireAt: time.Now().Add(ttl).Unix(),
	}
}

func (c *HybridCache) evictOldest() {
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

func (c *HybridCache) cleanup() {
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

func (c *HybridCache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.memory))
	for k := range c.memory {
		keys = append(keys, k)
	}
	return keys
}

func (c *HybridCache) DeleteMany(keys []string) {
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
