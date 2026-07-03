package cxsec

// ～宸汐御安全 v1 — 所有公共类型定义放这里，别搞乱了哦～

import "time"

// Session 代表一个已握手的客户端会话
type Session struct {
	ID          string    // 16B hex，对外不暴露真实内容
	Hint        [8]byte   // session_hint，明文传输的索引字段
	MasterKey   [32]byte  // ECDH 派生的主密钥，绝不离开内存
	FpHash      [32]byte  // 环境指纹哈希，混入 KDF
	Counter     uint32    // 单调递增请求计数，防重放
	CreatedAt   time.Time
	LastUsed    time.Time
	TTL         time.Duration
	NonceSet    map[[16]byte]int64 // nonce → 到期时间戳，防重放
}

// ChallengeRecord 代表一个 PoW 挑战
type ChallengeRecord struct {
	ID          [32]byte
	ServerNonce [32]byte
	Difficulty  uint8     // 前 N 字节为 0 的要求
	CreatedAt   time.Time
	ServerPriv  []byte    // 临时 ECDH 私钥，握手后销毁
	ServerPub   []byte    // 临时 ECDH 公钥，发给客户端
}

// ParsedRequest 是中间件解密后的结构化请求
type ParsedRequest struct {
	Counter   uint32
	Timestamp int64  // 毫秒
	Payload   []byte // 解密后的原始业务 JSON
}
