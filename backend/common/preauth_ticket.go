package common

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// 2FA 登录的第二步（验证码/备份码）之前必须先证明"这是刚刚密码校验通过的那次登录"，
// 否则攻击者只要知道 user_id 就能跳过密码、直接硬撞验证码/备份码。
// 这里参考 token_blacklist.go / service/passkey/session.go 的套路：
// 有 Redis 就用 Redis（一次性 GET+DEL，天然支持多实例），没有就退化为进程内 map。

const preAuthTicketTTL = 5 * time.Minute
const preAuthKeyPrefix = "preauth:"

var (
	preAuthStore = map[string]preAuthValue{}
	preAuthMu    sync.Mutex
)

type preAuthValue struct {
	UserId    int
	ExpiresAt time.Time
}

// IssuePreAuthTicket 在密码校验通过、还需要二次验证时签发一次性票据
func IssuePreAuthTicket(userId int) string {
	ticket := "pre_" + GetUUID()
	key := preAuthKeyPrefix + ticket
	value := preAuthValue{UserId: userId, ExpiresAt: time.Now().Add(preAuthTicketTTL)}

	if RedisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		raw, _ := json.Marshal(value)
		RedisClient.Set(ctx, key, raw, preAuthTicketTTL)
		return ticket
	}

	preAuthMu.Lock()
	preAuthStore[key] = value
	preAuthMu.Unlock()
	return ticket
}

// ConsumePreAuthTicket 校验并一次性消费票据（取出即删，防止被反复拿去撞库），
// 返回票据是否有效且确实绑定到 expectUserId
func ConsumePreAuthTicket(ticket string, expectUserId int) bool {
	if ticket == "" {
		return false
	}
	key := preAuthKeyPrefix + ticket

	if RedisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		raw, err := RedisClient.Get(ctx, key).Bytes()
		if err != nil {
			return false
		}
		RedisClient.Del(ctx, key)
		var v preAuthValue
		if json.Unmarshal(raw, &v) != nil {
			return false
		}
		return v.UserId == expectUserId
	}

	preAuthMu.Lock()
	defer preAuthMu.Unlock()
	v, ok := preAuthStore[key]
	if !ok {
		return false
	}
	delete(preAuthStore, key)
	if time.Now().After(v.ExpiresAt) {
		return false
	}
	return v.UserId == expectUserId
}
