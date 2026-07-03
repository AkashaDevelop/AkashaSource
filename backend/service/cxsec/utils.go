package cxsec

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"io"
	"time"
)

// unixNowMS 返回当前 Unix 毫秒时间戳
func unixNowMS() int64 {
	return time.Now().UnixMilli()
}

// unixNowSec 返回当前 Unix 秒时间戳
func unixNowSec() int64 {
	return time.Now().Unix()
}

// randBytes 生成 n 字节密码学安全随机数，出错则 panic（不可恢复）
func randBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic("cxsec: entropy source failed: " + err.Error())
	}
	return b
}

// rand32 生成 [32]byte 随机数
func rand32() [32]byte {
	var out [32]byte
	copy(out[:], randBytes(32))
	return out
}

// rand16 生成 [16]byte 随机数
func rand16() [16]byte {
	var out [16]byte
	copy(out[:], randBytes(16))
	return out
}

// rand8 生成 [8]byte 随机数
func rand8() [8]byte {
	var out [8]byte
	copy(out[:], randBytes(8))
	return out
}

// hexEncode 返回小写 hex 字符串
func hexEncode(b []byte) string {
	return hex.EncodeToString(b)
}

// hexDecode 解码 hex，失败返回 nil
func hexDecode(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil
	}
	return b
}

// zeroBytes ～把用完的密钥字节清成0，别让它在内存里赖着不走～
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// uint32ToLE 将 uint32 编码为 4 字节小端
func uint32ToLE(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}
