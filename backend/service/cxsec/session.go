package cxsec

// ～会话状态管理：nonce 黑名单 + 计数器 + 并发安全～

import (
	"crypto/ecdh"
	"errors"
	"sync"
	"time"
)

const (
	SessionTTL      = 8 * time.Hour
	NonceBlacklistTTL = 120 * time.Second
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

	masterKey, err := DeriveSessionKey(priv, peerPub, fpHash)
	if err != nil {
		return nil, err
	}

	hint := rand8()
	sess := &Session{
		ID:        hexEncode(randBytes(16)),
		MasterKey: masterKey,
		FpHash:    fpHash,
		Counter:   0,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		TTL:       SessionTTL,
		NonceSet:  make(map[[16]byte]int64),
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

// ConsumeNonce 检查 nonce 是否已使用，未使用则记录（防重放）
func (sess *Session) ConsumeNonce(nonce [16]byte) error {
	// 清理过期 nonce
	now := unixNowSec()
	for n, exp := range sess.NonceSet {
		if now > exp {
			delete(sess.NonceSet, n)
		}
	}
	if len(sess.NonceSet) >= MaxNoncesPerSession {
		return errors.New("cxsec: nonce store full")
	}
	if _, used := sess.NonceSet[nonce]; used {
		return errors.New("cxsec: nonce replayed")
	}
	sess.NonceSet[nonce] = now + int64(NonceBlacklistTTL.Seconds())
	return nil
}

// AdvanceCounter 验证并推进请求计数器（必须严格递增）
func (sess *Session) AdvanceCounter(counter uint32) error {
	if counter != sess.Counter {
		return errors.New("cxsec: counter mismatch")
	}
	sess.Counter++
	sess.LastUsed = time.Now()
	return nil
}
