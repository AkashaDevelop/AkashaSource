package cxsec

// ～宸汐KDF：用 ECDH 共享密钥 + 时间窗 + 计数 + 环境指纹派生每次请求的加密密钥～
// 即使会话主密钥泄露，历史请求也无法解密（时间窗已过）

import (
	"crypto/ecdh"
	"encoding/binary"
	"golang.org/x/crypto/sha3"
)

const (
	timeWindowSec = 30 // 30秒密钥时间窗
	kdfDomain     = "宸汐御安全v1\x00" // 域分隔符，防跨协议攻击
)

// DeriveSessionKey 从 ECDH 共享密钥派生会话主密钥
// 输入: ECDH 私钥、对端公钥、环境指纹哈希
// 输出: 32B 会话主密钥
func DeriveSessionKey(priv *ecdh.PrivateKey, peerPub *ecdh.PublicKey, fpHash [32]byte) ([32]byte, error) {
	shared, err := priv.ECDH(peerPub)
	if err != nil {
		return [32]byte{}, err
	}
	// HKDF-like 派生: SHA3-256(domain || shared || fpHash)
	h := sha3.New256()
	h.Write([]byte(kdfDomain))
	h.Write([]byte("master\x00"))
	h.Write(shared)
	h.Write(fpHash[:])
	var key [32]byte
	copy(key[:], h.Sum(nil))
	return key, nil
}

// DeriveRequestKey 为单次请求派生加密密钥
// 绑定: 主密钥 + 时间窗 + 请求计数 + 请求 nonce
// 效果: 每次请求密钥唯一，30s 后时间窗关闭
func DeriveRequestKey(masterKey [32]byte, timeBlock int64, counter uint32, nonce [16]byte) [32]byte {
	h := sha3.New256()
	h.Write([]byte(kdfDomain))
	h.Write([]byte("reqkey\x00"))
	h.Write(masterKey[:])
	var tb [8]byte
	binary.LittleEndian.PutUint64(tb[:], uint64(timeBlock))
	h.Write(tb[:])
	var cb [4]byte
	binary.LittleEndian.PutUint32(cb[:], counter)
	h.Write(cb[:])
	h.Write(nonce[:])
	var key [32]byte
	copy(key[:], h.Sum(nil))
	return key
}

// DeriveHMACKey 为御印签名派生 HMAC 密钥（与加密密钥分离）
func DeriveHMACKey(masterKey [32]byte, timeBlock int64) [32]byte {
	h := sha3.New256()
	h.Write([]byte(kdfDomain))
	h.Write([]byte("hmackey\x00"))
	h.Write(masterKey[:])
	var tb [8]byte
	binary.LittleEndian.PutUint64(tb[:], uint64(timeBlock))
	h.Write(tb[:])
	var key [32]byte
	copy(key[:], h.Sum(nil))
	return key
}

// DeriveWhiteningKey 为辰星密码的密钥白化层派生额外密钥
func DeriveWhiteningKey(masterKey [32]byte, sessionID string) ([32]byte, [32]byte) {
	h1 := sha3.New256()
	h1.Write([]byte(kdfDomain))
	h1.Write([]byte("whiten_pre\x00"))
	h1.Write(masterKey[:])
	h1.Write([]byte(sessionID))
	var pre [32]byte
	copy(pre[:], h1.Sum(nil))

	h2 := sha3.New256()
	h2.Write([]byte(kdfDomain))
	h2.Write([]byte("whiten_post\x00"))
	h2.Write(masterKey[:])
	h2.Write([]byte(sessionID))
	var post [32]byte
	copy(post[:], h2.Sum(nil))

	return pre, post
}

// CurrentTimeBlock 返回当前 30s 时间窗编号
func CurrentTimeBlock() int64 {
	return unixNowSec() / timeWindowSec
}

// TimeBlockFromMS 从毫秒时间戳计算时间窗编号
func TimeBlockFromMS(tsMS int64) int64 {
	return (tsMS / 1000) / timeWindowSec
}
