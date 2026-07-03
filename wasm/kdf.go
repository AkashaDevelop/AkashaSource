// +build js,wasm

package main

// ～宸汐御安全 WASM 端 KDF v2：与服务端 kdf.go 完全对称的 HKDF-SHA3 + 纪元棘轮～
// 前向安全同款逻辑搬进浏览器：拿到当前纪元密钥也算不出上一纪元密钥

import (
	"encoding/binary"
	"io"

	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/sha3"
)

const (
	kdfDomainV2        = "宸汐御安全v2\x00"
	timeWindowSecV2    = 30
	ratchetIntervalSec = 60 // 与服务端保持一致：1分钟纪元
)

// hkdfSum32 标准 HKDF-SHA3-256（Extract+Expand），输出 32 字节
func hkdfSum32(secret, salt, info []byte) [32]byte {
	r := hkdf.New(sha3.New256, secret, salt, info)
	var out [32]byte
	io.ReadFull(r, out[:]) //nolint:errcheck
	return out
}

// deriveEpoch0 从 ECDH 共享密钥派生纪元0密钥，与服务端 DeriveSessionKey 对称
func deriveEpoch0(shared []byte, fpHash [32]byte) [32]byte {
	return hkdfSum32(shared, fpHash[:], []byte(kdfDomainV2+"epoch0\x00"))
}

// advanceEpochKey 棘轮推进：单向不可逆
func advanceEpochKey(cur [32]byte) [32]byte {
	return hkdfSum32(cur[:], nil, []byte(kdfDomainV2+"ratchet\x00"))
}

func deriveRequestKey(epochKey [32]byte, timeBlock int64, counter uint32, nonce [16]byte) [32]byte {
	buf := make([]byte, 0, len(kdfDomainV2)+6+1+8+4+16)
	buf = append(buf, []byte(kdfDomainV2+"reqkey\x00")...)
	var tb [8]byte
	binary.LittleEndian.PutUint64(tb[:], uint64(timeBlock))
	buf = append(buf, tb[:]...)
	var cb [4]byte
	binary.LittleEndian.PutUint32(cb[:], counter)
	buf = append(buf, cb[:]...)
	buf = append(buf, nonce[:]...)
	return hkdfSum32(epochKey[:], nil, buf)
}

func deriveHMACKey(epochKey [32]byte, timeBlock int64) [32]byte {
	buf := make([]byte, 0, len(kdfDomainV2)+7+8)
	buf = append(buf, []byte(kdfDomainV2+"hmackey\x00")...)
	var tb [8]byte
	binary.LittleEndian.PutUint64(tb[:], uint64(timeBlock))
	buf = append(buf, tb[:]...)
	return hkdfSum32(epochKey[:], nil, buf)
}

func deriveWhiteningKey(epochKey [32]byte, sessionID string) ([32]byte, [32]byte) {
	pre := hkdfSum32(epochKey[:], nil, []byte(kdfDomainV2+"whiten_pre\x00"+sessionID))
	post := hkdfSum32(epochKey[:], nil, []byte(kdfDomainV2+"whiten_post\x00"+sessionID))
	return pre, post
}

func currentEpochV2() int64 {
	return nowSec() / ratchetIntervalSec
}

// ratchetIfNeeded 按挂钟时间把会话的当前纪元密钥推进到最新，旧密钥清零
func (sess *jsSession) ratchetIfNeeded() {
	target := currentEpochV2()
	for sess.epochIndex < target {
		zeroKey32(&sess.prevEpochKey)
		sess.prevEpochKey = sess.epochKey
		sess.epochKey = advanceEpochKey(sess.epochKey)
		sess.epochIndex++
	}
}

func zeroKey32(k *[32]byte) {
	for i := range k {
		k[i] = 0
	}
}
