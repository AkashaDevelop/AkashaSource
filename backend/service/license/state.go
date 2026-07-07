// ⚠️ REMOVABLE MODULE — 系统授权门禁
// 独立的一次性 CSRF state，不依赖 controller/oauth 包（那边的 generateState/verifyState 是包内私有的，
// 而且这个模块要保持自成一体、方便以后整体删除），写法照抄同样的 Redis+内存兜底思路
package license

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"STfreApi/common"
)

type stateEntry struct {
	expires time.Time
}

var stateStore sync.Map

func generateState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	state := hex.EncodeToString(b)

	if common.RedisClient != nil {
		common.SetCache("license_oauth_state:"+state, true, 5*time.Minute)
	} else {
		stateStore.Store(state, stateEntry{expires: time.Now().Add(5 * time.Minute)})
	}
	return state
}

func verifyState(state string) bool {
	if state == "" {
		return false
	}
	if common.RedisClient != nil {
		var ok bool
		if !common.GetCache("license_oauth_state:"+state, &ok) {
			return false
		}
		common.RedisClient.Del(context.Background(), "license_oauth_state:"+state)
		return ok
	}
	val, loaded := stateStore.LoadAndDelete(state)
	if !loaded {
		return false
	}
	return time.Now().Before(val.(stateEntry).expires)
}
