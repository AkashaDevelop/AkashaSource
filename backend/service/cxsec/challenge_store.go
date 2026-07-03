package cxsec

// ～挑战存储：内存 Map + 定期清理，60s 内挑战单次有效～

import (
	"crypto/ecdh"
	"sync"
	"time"
)

// ChallengeEntry 导出，供 controller 访问
type ChallengeEntry struct {
	Nonce      [32]byte
	ServerPriv *ecdh.PrivateKey
	ExpiresAt  time.Time
}

var (
	challengeMu    sync.Mutex
	challengeStore = make(map[[32]byte]*ChallengeEntry)
)

// StoreChallenge 存储挑战，返回 challengeID hex
func StoreChallenge(nonce [32]byte, priv *ecdh.PrivateKey) string {
	cid := rand32()
	entry := &ChallengeEntry{
		Nonce:      nonce,
		ServerPriv: priv,
		ExpiresAt:  time.Now().Add(time.Duration(PowTTLSec) * time.Second),
	}
	challengeMu.Lock()
	challengeStore[cid] = entry
	challengeMu.Unlock()
	return hexEncode(cid[:])
}

// ConsumeChallenge 取出并删除挑战（单次有效）
func ConsumeChallenge(cid [32]byte) (*ChallengeEntry, bool) {
	challengeMu.Lock()
	defer challengeMu.Unlock()
	entry, ok := challengeStore[cid]
	if !ok {
		return nil, false
	}
	delete(challengeStore, cid)
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	return entry, true
}

// PurgeExpiredChallenges 定期清理，在 main 启动一个 goroutine 调用
func PurgeExpiredChallenges() {
	for {
		time.Sleep(2 * time.Minute)
		now := time.Now()
		challengeMu.Lock()
		for k, v := range challengeStore {
			if now.After(v.ExpiresAt) {
				delete(challengeStore, k)
			}
		}
		challengeMu.Unlock()
	}
}
