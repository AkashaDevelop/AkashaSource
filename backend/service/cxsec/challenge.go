package cxsec

// ～算力证明：SHA3-256 + Argon2id 双重哈希，防爬虫/批量攻击～
// 适配老旧设备：Argon2id 内存量可配置，默认 64KB（较保守）

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"io"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/sha3"
)

const (
	PowDifficultyDefault = 20    // ～生产强度：约百万次尝试起步，脚本刷子哭晕在厕所～
	PowArgon2Memory      = 128   // 128KB，64KB的两倍，碰撞成本翻番，老旧设备仍可接受
	PowArgon2Iterations  = 1
	PowArgon2Threads     = 1
	PowTTLSec            = 120   // 挑战有效期2分钟（含时间因子自动过期）
)

// GenChallenge 生成 PoW 挑战
func GenChallenge(difficulty uint8) (nonce [32]byte, target []byte) {
	io.ReadFull(rand.Reader, nonce[:]) //nolint:errcheck
	// target = difficulty 位 0 → 前 difficulty/8 个字节为0，最后不足一字节用位掩码
	targetLen := 4
	target = make([]byte, targetLen)
	// 简单表示：difficulty 表示前 difficulty 个 bit 为0
	// 用4字节 uint32 表示阈值：threshold = 2^(32-difficulty)
	if difficulty >= 32 {
		difficulty = 31
	}
	thres := uint32(1) << (32 - difficulty)
	binary.BigEndian.PutUint32(target, thres)
	return
}

// VerifyPoW 验证客户端提交的 PoW 解答
// solution: uint64 客户端找到的解
// timeHint: 时间因子（floor(unix/60)），防止旧挑战被复用
func VerifyPoW(nonce [32]byte, solution uint64, difficulty uint8, timeHint int64) bool {
	// 组装求解输入
	input := make([]byte, 32+8+8)
	copy(input[:32], nonce[:])
	binary.LittleEndian.PutUint64(input[32:40], solution)
	binary.LittleEndian.PutUint64(input[40:48], uint64(timeHint))

	// Step 1: SHA3-256 初筛
	sha3Hash := sha3.Sum256(input)
	thres := uint32(1) << (32 - difficulty)
	if difficulty < 32 && binary.BigEndian.Uint32(sha3Hash[:4]) >= thres {
		return false
	}

	// Step 2: Argon2id 内存困难验证（防 ASIC/脚本批量求解）
	argHash := argon2.IDKey(sha3Hash[:], nonce[:8], PowArgon2Iterations, PowArgon2Memory, PowArgon2Threads, 8)
	// argon2 输出前4字节需 < threshold/4（比 SHA3 宽松一些，避免太慢）
	looseThres := uint32(1) << (32 - difficulty + 2)
	return binary.BigEndian.Uint32(argHash[:4]) < looseThres
}

// CurrentPowTimeHint 当前 PoW 时间因子（每60s变化，挑战自动过期）
func CurrentPowTimeHint() int64 {
	return unixNowSec() / 60
}

// FingerprintVerify 简单校验环境指纹（确保是浏览器环境发来的）
// fpToken = SHA3-256(browser_data) 前32字节，服务端只做基础格式校验
// ～不是全零、也不是清一色同一个字节糊弄过去的，才算是像样的哈希值哦～
func FingerprintVerify(fpToken []byte) bool {
	if len(fpToken) != 32 {
		return false
	}
	var zero [32]byte
	if subtle.ConstantTimeCompare(fpToken, zero[:]) == 1 {
		return false
	}
	// 粗略熵检测：真实 SHA3 输出的字节分布很散，若 32 字节里不同值少于 8 种，
	// 大概率是"全1字节"这类平凡构造，直接拒绝
	seen := make(map[byte]struct{}, 32)
	for _, b := range fpToken {
		seen[b] = struct{}{}
	}
	return len(seen) >= 8
}
