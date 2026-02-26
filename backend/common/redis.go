package common

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis() {
	if RedisAddr == "" {
		return
	}
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     RedisAddr,
		Password: RedisPassword,
		DB:       RedisDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := RedisClient.Ping(ctx).Err(); err != nil {
		RedisClient = nil
	}
}

func GetCache(key string, dest any) bool {
	if RedisClient == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	val, err := RedisClient.Get(ctx, key).Result()
	if err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false
	}
	return true
}

func SetCache(key string, value any, ttl time.Duration) {
	if RedisClient == nil {
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	RedisClient.Set(ctx, key, string(data), ttl)
}
