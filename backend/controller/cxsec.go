package controller

// ～宸汐御安全握手端点：PoW 挑战 + ECDH 密钥协商～

import (
	"STfreApi/common"
	"STfreApi/service/cxsec"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CxGetChallenge GET /api/cx/challenge
// 下发 PoW 挑战 + 服务端临时 ECDH 公钥
func CxGetChallenge(c *gin.Context) {
	// 生成临时 ECDH P-256 密钥对
	curve := ecdh.P256()
	priv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "entropy failure"})
		return
	}

	nonce, _ := cxsec.GenChallenge(cxsec.PowDifficultyDefault)

	// 存储挑战记录（60s 有效）
	cid := cxsec.StoreChallenge(nonce, priv)

	c.JSON(http.StatusOK, gin.H{
		"cid":   cid,
		"nonce": hex.EncodeToString(nonce[:]),
		"srv":   hex.EncodeToString(priv.PublicKey().Bytes()),
		"ts":    cxsec.CurrentPowTimeHint(),
		"diff":  cxsec.PowDifficultyDefault,
		"algo":  "sha3+argon2id",
		"ttl":   60,
	})
}

// CxHandshake POST /api/cx/ks
// 接收 PoW 解答 + 客户端 ECDH 公钥，建立加密会话
func CxHandshake(c *gin.Context) {
	const bodyLen = 32 + 33 + 8 + 32 + 32 // cid+pub+solution+argon_verify+fp

	raw := make([]byte, bodyLen)
	if _, err := io.ReadFull(c.Request.Body, raw); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	off := 0
	var cid [32]byte
	copy(cid[:], raw[off:off+32])
	off += 32

	clientPubBytes := raw[off : off+33]
	off += 33

	var solution [8]byte
	copy(solution[:], raw[off:off+8])
	off += 8

	// argon2 验证字节（客户端预计算，服务端可选验证）
	// argonVerify := raw[off : off+32]
	off += 32

	fpToken := raw[off : off+32]

	// 取出挑战
	challenge, ok := cxsec.ConsumeChallenge(cid)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}

	// 验证 PoW
	solutionU64 := uint64(solution[0]) | uint64(solution[1])<<8 | uint64(solution[2])<<16 |
		uint64(solution[3])<<24 | uint64(solution[4])<<32 | uint64(solution[5])<<40 |
		uint64(solution[6])<<48 | uint64(solution[7])<<56

	if !cxsec.VerifyPoW(challenge.Nonce, solutionU64, cxsec.PowDifficultyDefault, cxsec.CurrentPowTimeHint()) {
		c.Status(http.StatusForbidden)
		return
	}

	// 验证环境指纹
	if !cxsec.FingerprintVerify(fpToken) {
		c.Status(http.StatusForbidden)
		return
	}

	// ECDH 建立会话
	curve := ecdh.P256()
	peerPub, err := curve.NewPublicKey(clientPubBytes)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	sess, err := cxsec.DefaultStore.Create(challenge.ServerPriv, peerPub, fpToken)
	if err != nil {
		common.Fail(c, common.CodeServerError, "session creation failed")
		return
	}

	// 响应：会话 hint(8B) + 计数器初值(4B) + 会话有效期(4B)
	// 全二进制，不用 JSON
	resp := make([]byte, 8+4+4)
	copy(resp[:8], sess.Hint[:])
	binary.LittleEndian.PutUint32(resp[8:12], 0)     // counter init = 0
	binary.LittleEndian.PutUint32(resp[12:16], 28800) // TTL = 8h = 28800s

	c.Data(http.StatusOK, "application/octet-stream", resp)
}
