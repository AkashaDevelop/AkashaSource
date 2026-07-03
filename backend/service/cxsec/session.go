package cxsec

// ～会话状态管理：纪元棘轮 + nonce 黑名单 + 计数器，全部并发安全～

import (
	"crypto/ecdh"
	"errors"
	"sync"
	"time"
)

const (
	SessionTTL          = 8 * time.Hour
	NonceBlacklistTTL   = 120 * time.Second
	MaxNoncesPerSession = 200_000
)

// SessionStore 线程安全的会话存储
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session // key = hint hex
}

var DefaultStore = &SessionStore{
	sessions: make(map[string]*Session),
}

// Create 创建并存储新会话
func (s *SessionStore) Create(priv *ecdh.PrivateKey, peerPub *ecdh.PublicKey, fpToken []byte) (*Session, error) {
	var fpHash [32]byte
	copy(fpHash[:], fpToken)

	epoch0, err := DeriveSessionKey(priv, peerPub, fpHash)
	if err != nil {
		return nil, err
	}

	hint := rand8()
	sess := &Session{
		ID:         hexEncode(randBytes(16)),
		FpHash:     fpHash,
		CreatedAt:  time.Now(),
		TTL:        SessionTTL,
		epochKey:   epoch0,
		epochIndex: CurrentEpoch(),
		counter:    0,
		lastUsed:   time.Now(),
		nonceSet:   make(map[[16]byte]int64),
	}
	copy(sess.Hint[:], hint[:])

	s.mu.Lock()
	s.sessions[hexEncode(hint[:])] = sess
	s.mu.Unlock()
	return sess, nil
}

// Get 按 hint 查找会话
func (s *SessionStore) Get(hint [8]byte) (*Session, error) {
	key := hexEncode(hint[:])
	s.mu.RLock()
	sess, ok := s.sessions[key]
	s.mu.RUnlock()
	if !ok {
		return nil, errors.New("cxsec: session not found")
	}
	if time.Since(sess.CreatedAt) > sess.TTL {
		s.mu.Lock()
		delete(s.sessions, key)
		s.mu.Unlock()
		return nil, errors.New("cxsec: session expired")
	}
	return sess, nil
}

// Delete 按 hint 主动销毁会话（退出登录时调用，立即切断密钥链）
func (s *SessionStore) Delete(hint [8]byte) {
	key := hexEncode(hint[:])
	s.mu.Lock()
	delete(s.sessions, key)
	s.mu.Unlock()
}
// 旧纪元密钥用完即焚：即便之后内存被拿到，也只能看见"现在"，看不见"过去"
func (sess *Session) ratchetIfNeeded() {
	target := CurrentEpoch()
	for sess.epochIndex < target {
		old := sess.prevEpochKey
		sess.prevEpochKey = sess.epochKey
		sess.epochKey = AdvanceEpochKey(sess.epochKey)
		sess.epochIndex++
		zeroBytes(old[:])
	}
}

// EpochKeys 返回当前纪元密钥与上一纪元密钥（供纪元边界容错校验）
func (sess *Session) EpochKeys() (cur, prev [32]byte) {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	sess.ratchetIfNeeded()
	return sess.epochKey, sess.prevEpochKey
}

// Counter 读取当前请求计数器
func (sess *Session) Counter() uint32 {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	return sess.counter
}

// ConsumeNonce 检查 nonce 是否已使用，未使用则记录（防重放）
func (sess *Session) ConsumeNonce(nonce [16]byte) error {
	sess.mu.Lock()
	defer sess.mu.Unlock()

	now := unixNowSec()
	for n, exp := range sess.nonceSet {
		if now > exp {
			delete(sess.nonceSet, n)
		}
	}
	if len(sess.nonceSet) >= MaxNoncesPerSession {
		return errors.New("cxsec: nonce store full")
	}
	if _, used := sess.nonceSet[nonce]; used {
		return errors.New("cxsec: nonce replayed")
	}
	sess.nonceSet[nonce] = now + int64(NonceBlacklistTTL.Seconds())
	return nil
}

// AdvanceCounter 验证并推进请求计数器（必须严格递增）
func (sess *Session) AdvanceCounter(counter uint32) error {
	sess.mu.Lock()
	defer sess.mu.Unlock()

	if counter != sess.counter {
		return errors.New("cxsec: counter mismatch")
	}
	sess.counter++
	sess.lastUsed = time.Now()
	return nil
}
