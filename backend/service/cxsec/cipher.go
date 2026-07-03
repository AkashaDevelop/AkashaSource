package cxsec

// ～辰星密码：AES-256-GCM + 白化层
// nonce 由调用方传入，同一个 nonce 既用于 KDF 派生，也用于 GCM IV（前12B）
// 这样密文绑定到密钥派生的 nonce，逆向者无法独立分析两者～

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
)

const (
	// Wire format: 8B hint | 16B nonce | ciphertext | 16B tag | 32B sig
	BodyHintLen  = 8
	BodyNonceLen = 16
	BodyTagLen   = 16
	BodySigLen   = 32

	// 密文内部: 4B counter | 8B ts_ms | 4B paylen | payload | padding | 1B padlen
	innerCounterLen = 4
	innerTSLen      = 8
	innerPayLenLen  = 4
	innerPadMax     = 127
)

// Encrypt 加密内部块。nonce 由调用方提供（先 RandNonce，再 DeriveRequestKey，再传入此处）
func Encrypt(requestKey [32]byte, whiteningPre, whiteningPost [32]byte,
	nonce [16]byte, counter uint32, tsMS int64, plaintext []byte) (ciphertext, tag []byte, err error) {

	// 随机 padding，混淆流量长度特征 (0~127B)
	padBytes := randBytes(1)
	padLen := int(padBytes[0] & 0x7F)
	padding := randBytes(padLen)

	// 内部明文: [4B counter][8B ts_ms][4B paylen][payload][padding][1B padlen]
	totalInner := innerCounterLen + innerTSLen + innerPayLenLen + len(plaintext) + padLen + 1
	inner := make([]byte, totalInner)
	off := 0
	binary.LittleEndian.PutUint32(inner[off:], counter); off += innerCounterLen
	binary.LittleEndian.PutUint64(inner[off:], uint64(tsMS)); off += innerTSLen
	binary.LittleEndian.PutUint32(inner[off:], uint32(len(plaintext))); off += innerPayLenLen
	copy(inner[off:], plaintext); off += len(plaintext)
	copy(inner[off:], padding); off += padLen
	inner[off] = byte(padLen)

	// 密钥白化（输入）: inner XOR pre_key（循环）
	whitened := xorWithKey(inner, whiteningPre[:])

	// AES-256-GCM，使用 nonce 前12B 作为 GCM IV
	block, err := aes.NewCipher(requestKey[:])
	if err != nil {
		return
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, 12)
	if err != nil {
		return
	}
	sealed := gcm.Seal(nil, nonce[:12], whitened, nil)

	// 输出白化: 密文(不含tag) XOR post_key
	ciphertext = xorWithKey(sealed[:len(sealed)-16], whiteningPost[:])
	tag = sealed[len(sealed)-16:]
	return
}

// Decrypt 解密并验证 counter 和时间戳
func Decrypt(requestKey [32]byte, whiteningPre, whiteningPost [32]byte,
	nonce [16]byte, ciphertext, tag []byte,
	expectedCounter uint32, allowedSkewMS int64) (payload []byte, counter uint32, tsMS int64, err error) {

	// 反向输出白化
	deCiphered := xorWithKey(ciphertext, whiteningPost[:])

	block, err := aes.NewCipher(requestKey[:])
	if err != nil {
		return
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, 12)
	if err != nil {
		return
	}
	combined := append(deCiphered, tag...) //nolint:gocritic
	whitened, err := gcm.Open(nil, nonce[:12], combined, nil)
	if err != nil {
		err = errors.New("cxsec: decrypt failed")
		return
	}

	// 反向输入白化
	inner := xorWithKey(whitened, whiteningPre[:])

	if len(inner) < innerCounterLen+innerTSLen+innerPayLenLen+1 {
		err = errors.New("cxsec: inner block too short")
		return
	}
	off := 0
	counter = binary.LittleEndian.Uint32(inner[off:]); off += innerCounterLen
	tsMS = int64(binary.LittleEndian.Uint64(inner[off:])); off += innerTSLen
	payLen := int(binary.LittleEndian.Uint32(inner[off:])); off += innerPayLenLen

	if counter != expectedCounter {
		err = errors.New("cxsec: counter mismatch")
		return
	}
	nowMS := unixNowMS()
	if diff := nowMS - tsMS; diff < -allowedSkewMS || diff > allowedSkewMS {
		err = errors.New("cxsec: timestamp out of window")
		return
	}
	if off+payLen > len(inner)-1 {
		err = errors.New("cxsec: payload length overflow")
		return
	}
	payload = inner[off : off+payLen]
	return
}

// xorWithKey 循环 XOR
func xorWithKey(data, key []byte) []byte {
	out := make([]byte, len(data))
	kl := len(key)
	for i, b := range data {
		out[i] = b ^ key[i%kl]
	}
	return out
}

// RandNonce 生成随机16B nonce（供外部调用）
func RandNonce() [16]byte {
	var n [16]byte
	copy(n[:], randBytes(16))
	return n
}

// UnixNowMS 导出给中间件使用
func UnixNowMS() int64 { return unixNowMS() }
