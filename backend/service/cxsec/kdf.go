package cxsec

// ～宸汐KDF v2：标准 HKDF-SHA3 派生 + 纪元棘轮（前向安全）～
// 会话建立后不再长期持有"万能主密钥"：只保留当前纪元密钥，
// 每 RatchetIntervalSec 秒单向棘轮推进一次，旧纪元密钥用完即焚，
// 即使服务器内存在某一刻被拿到，也只能看见"现在"，看不见"过去"。

import (
	"crypto/ecdh"
	"encoding/binary"
	"io"

	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/sha3"
)

const (
	timeWindowSec      = 30 // 30秒 HMAC 签名时间窗（细粒度，防重放）
	RatchetIntervalSec = 60  // 1分钟纪元棘轮：泄露最多暴露60s流量，告别"10分钟窗口"
	kdfDomain         = "宸汐御安全v2\x00" // 域分隔符，v2 标记协议已破坏性升级
)

// hkdfSum32 标准 HKDF-SHA3-256（Extract+Expand），输出 32 字节
func hkdfSum32(secret, salt, info []byte) [32]byte {
	r := hkdf.New(sha3.New256, secret, salt, info)
	var out [32]byte
	io.ReadFull(r, out[:]) //nolint:errcheck // hkdf.Reader 对固定长度读取不会出错
	return out
}

// DeriveSessionKey 从 ECDH 共享密钥派生纪元0密钥（会话根密钥用完即焚，不长期保留）
// 输入: ECDH 私钥、对端公钥、环境指纹哈希
// 输出: 32B 纪元0密钥
func DeriveSessionKey(priv *ecdh.PrivateKey, peerPub *ecdh.PublicKey, fpHash [32]byte) ([32]byte, error) {
	shared, err := priv.ECDH(peerPub)
	if err != nil {
		return [32]byte{}, err
	}
	// HKDF-Extract 用指纹哈希做盐，HKDF-Expand 派生纪元0密钥
	epoch0 := hkdfSum32(shared, fpHash[:], []byte(kdfDomain+"epoch0\x00"))
	zeroBytes(shared)
	return epoch0, nil
}

// AdvanceEpochKey 棘轮推进：由当前纪元密钥单向派生下一纪元密钥
// 单向不可逆——拿到 epoch(n) 无法算出 epoch(n-1)，这就是前向安全的核心
func AdvanceEpochKey(cur [32]byte) [32]byte {
	next := hkdfSum32(cur[:], nil, []byte(kdfDomain+"ratchet\x00"))
	return next
}

// DeriveRequestKey 为单次请求派生加密密钥
// 绑定: 当前纪元密钥 + 时间窗 + 请求计数 + 请求 nonce
func DeriveRequestKey(epochKey [32]byte, timeBlock int64, counter uint32, nonce [16]byte) [32]byte {
	buf := make([]byte, 0, len(kdfDomain)+6+1+8+4+16)
	buf = append(buf, []byte(kdfDomain+"reqkey\x00")...)
	var tb [8]byte
	binary.LittleEndian.PutUint64(tb[:], uint64(timeBlock))
	buf = append(buf, tb[:]...)
	var cb [4]byte
	binary.LittleEndian.PutUint32(cb[:], counter)
	buf = append(buf, cb[:]...)
	buf = append(buf, nonce[:]...)
	return hkdfSum32(epochKey[:], nil, buf)
}

// DeriveHMACKey 为御印签名派生 HMAC 密钥（与加密密钥分离）
func DeriveHMACKey(epochKey [32]byte, timeBlock int64) [32]byte {
	buf := make([]byte, 0, len(kdfDomain)+7+8)
	buf = append(buf, []byte(kdfDomain+"hmackey\x00")...)
	var tb [8]byte
	binary.LittleEndian.PutUint64(tb[:], uint64(timeBlock))
	buf = append(buf, tb[:]...)
	return hkdfSum32(epochKey[:], nil, buf)
}

// DeriveWhiteningKey 为辰星密码的密钥白化层派生额外密钥
func DeriveWhiteningKey(epochKey [32]byte, sessionID string) ([32]byte, [32]byte) {
	pre := hkdfSum32(epochKey[:], nil, []byte(kdfDomain+"whiten_pre\x00"+sessionID))
	post := hkdfSum32(epochKey[:], nil, []byte(kdfDomain+"whiten_post\x00"+sessionID))
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

// CurrentEpoch 返回当前纪元编号（每 RatchetIntervalSec 秒推进一次）
func CurrentEpoch() int64 {
	return unixNowSec() / RatchetIntervalSec
}
