package cxsec

// ～宸汐御安全 v2 — 所有公共类型定义放这里，别搞乱了哦～

import (
	"sync"
	"time"
)

// Session 代表一个已握手的客户端会话
// 密钥状态（epochKey 等）全部私有，外部只能通过 CurrentEpochKeys/ConsumeNonce/
// AdvanceCounter 等方法访问，方法内部统一加锁，杜绝并发数据竞争～
type Session struct {
	ID        string        // 16B hex，对外不暴露真实内容
	Hint      [8]byte       // session_hint，明文传输的索引字段
	FpHash    [32]byte      // 环境指纹哈希，混入 KDF
	CreatedAt time.Time
	TTL       time.Duration

	mu           sync.Mutex
	epochKey     [32]byte           // 当前纪元密钥（棘轮推进，旧值用完即焚）
	prevEpochKey [32]byte           // 上一纪元密钥，仅用于边界容错
	epochIndex   int64              // 当前纪元密钥对应的纪元编号
	counter      uint32             // 单调递增请求计数，防重放
	lastUsed     time.Time
	nonceSet     map[[16]byte]int64 // nonce → 到期时间戳，防重放
}

// ChallengeRecord 代表一个 PoW 挑战
type ChallengeRecord struct {
	ID          [32]byte
	ServerNonce [32]byte
	Difficulty  uint8 // 前 N 字节为 0 的要求
	CreatedAt   time.Time
	ServerPriv  []byte // 临时 ECDH 私钥，握手后销毁
	ServerPub   []byte // 临时 ECDH 公钥，发给客户端
}

// ParsedRequest 是中间件解密后的结构化请求
type ParsedRequest struct {
	Counter   uint32
	Timestamp int64  // 毫秒
	Payload   []byte // 解密后的原始业务 JSON
}

