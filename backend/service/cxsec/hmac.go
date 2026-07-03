package cxsec

// ～御印签名：魔改 HMAC-SHA3，内外填充随时间窗变化，30s 后签名完全失效～

import (
	"crypto/hmac"
	"crypto/subtle"
	"golang.org/x/crypto/sha3"
)

// Sign 计算御印签名，覆盖 Body 前缀（hint+nonce+ciphertext+tag）
// hmacKey 由 DeriveHMACKey 得到（含时间窗因子）
func Sign(hmacKey [32]byte, body []byte) [32]byte {
	// 标准 HMAC-SHA3-256 足够，时间变化已在 key 中体现
	mac := hmac.New(sha3.New256, hmacKey[:])
	mac.Write(body)
	var sig [32]byte
	copy(sig[:], mac.Sum(nil))
	return sig
}

// Verify 验证御印签名（恒定时间比较，防 timing attack）
func Verify(hmacKey [32]byte, body []byte, sig [32]byte) bool {
	expected := Sign(hmacKey, body)
	return subtle.ConstantTimeCompare(expected[:], sig[:]) == 1
}
