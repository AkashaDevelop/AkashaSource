package passkey

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"STfreApi/common"

	webauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

const sessionTTL = 3 * time.Minute

var (
	sessionStore = map[string]sessionValue{}
	sessionMu    sync.Mutex
)

type sessionValue struct {
	Data      webauthn.SessionData
	ExpiresAt time.Time
}

func saveSession(key string, data *webauthn.SessionData) (string, error) {
	if data == nil {
		return "", fmt.Errorf("会话数据为空")
	}
	sessionID := uuid.NewString()
	value := sessionValue{Data: *data, ExpiresAt: time.Now().Add(sessionTTL)}
	if common.RedisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		raw, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		if err := common.RedisClient.Set(ctx, key+sessionID, raw, sessionTTL).Err(); err != nil {
			return "", err
		}
		return sessionID, nil
	}
	sessionMu.Lock()
	sessionStore[key+sessionID] = value
	sessionMu.Unlock()
	return sessionID, nil
}

func popSession(key, sessionID string) (*webauthn.SessionData, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("缺少 session_id")
	}
	fullKey := key + sessionID
	if common.RedisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		raw, err := common.RedisClient.Get(ctx, fullKey).Bytes()
		if err != nil {
			return nil, fmt.Errorf("passkey 会话不存在或已过期")
		}
		var value sessionValue
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("passkey 会话无效")
		}
		common.RedisClient.Del(ctx, fullKey)
		return &value.Data, nil
	}

	sessionMu.Lock()
	defer sessionMu.Unlock()
	value, ok := sessionStore[fullKey]
	if !ok {
		return nil, fmt.Errorf("passkey 会话不存在或已过期")
	}
	delete(sessionStore, fullKey)
	if time.Now().After(value.ExpiresAt) {
		return nil, fmt.Errorf("passkey 会话已过期")
	}
	return &value.Data, nil
}

func SaveLoginSession(data *webauthn.SessionData) (string, error) {
	return saveSession("passkey:login:", data)
}

func PopLoginSession(sessionID string) (*webauthn.SessionData, error) {
	return popSession("passkey:login:", sessionID)
}

func SaveRegisterSession(data *webauthn.SessionData) (string, error) {
	return saveSession("passkey:register:", data)
}

func PopRegisterSession(sessionID string) (*webauthn.SessionData, error) {
	return popSession("passkey:register:", sessionID)
}
