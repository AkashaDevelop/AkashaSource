package middleware

// ～宸汐御安全中间件：解包→验签→解密→注入 payload，响应走加密回路～
// v2: 密钥全部走会话纪元棘轮（EpochKeys），不再有常驻不变的 MasterKey～

import (
	"STfreApi/common"
	"STfreApi/service/cxsec"
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const allowedSkewMS = 45_000 // ±45s 时间窗

// CxSecMiddleware 对路由启用宸汐御安全保护
// ～管理员关掉或客户端传的不是 octet-stream，就当普通接口放行，不用慌～
func CxSecMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优雅降级：全局禁用 或 请求不是加密二进制格式时直接放行
		ct := c.GetHeader("Content-Type")
		if !common.CxSecEnabled || !strings.Contains(ct, "application/octet-stream") {
			c.Next()
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		minLen := cxsec.BodyHintLen + cxsec.BodyNonceLen + 1 + cxsec.BodyTagLen + cxsec.BodySigLen
		if err != nil || len(body) < minLen {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		// ── 1. 解包 ──────────────────────────────────────────
		off := 0
		var hint [8]byte
		copy(hint[:], body[off:off+cxsec.BodyHintLen])
		off += cxsec.BodyHintLen

		var nonce [16]byte
		copy(nonce[:], body[off:off+cxsec.BodyNonceLen])
		off += cxsec.BodyNonceLen

		sigStart := len(body) - cxsec.BodySigLen
		tagStart := sigStart - cxsec.BodyTagLen
		ciphertext := body[off:tagStart]
		tag := body[tagStart:sigStart]
		var sig [32]byte
		copy(sig[:], body[sigStart:])

		// ── 2. 查找会话 ──────────────────────────────────────
		sess, err := cxsec.DefaultStore.Get(hint)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 纪元密钥（当前+上一纪元，容忍棘轮推进边界）
		curEpoch, prevEpoch := sess.EpochKeys()

		// ── 3. 验御印签名（同时容忍：时间窗边界±1窗 × 纪元边界当前/上一纪元） ──
		timeBlock := cxsec.CurrentTimeBlock()
		verified := false
		var verifiedEpoch [32]byte
	verifyLoop:
		for _, ek := range [][32]byte{curEpoch, prevEpoch} {
			for _, tb := range []int64{timeBlock, timeBlock - 1} {
				hmacKey := cxsec.DeriveHMACKey(ek, tb)
				if cxsec.Verify(hmacKey, body[:sigStart], sig) {
					verified = true
					verifiedEpoch = ek
					break verifyLoop
				}
			}
		}
		if !verified {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// ── 4. nonce 去重 ─────────────────────────────────────
		if err := sess.ConsumeNonce(nonce); err != nil {
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}

		// ── 5. 解辰星密码 ─────────────────────────────────────
		expectedCounter := sess.Counter()
		pre, post := cxsec.DeriveWhiteningKey(verifiedEpoch, sess.ID)
		reqKey := cxsec.DeriveRequestKey(verifiedEpoch, timeBlock, expectedCounter, nonce)
		payload, counter, _, err := cxsec.Decrypt(reqKey, pre, post, nonce, ciphertext, tag, expectedCounter, allowedSkewMS)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// ── 6. 推进计数器 ─────────────────────────────────────
		if err := sess.AdvanceCounter(counter); err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// ── 7. 注入解密后的请求体 ─────────────────────────────
		c.Request.Body = io.NopCloser(bytes.NewReader(payload))
		c.Request.ContentLength = int64(len(payload))
		c.Set("cxsec_session", sess)

		// ── 8. 加密响应 ──────────────────────────────────────
		cxWriter := &cxSecWriter{ResponseWriter: c.Writer, sess: sess}
		c.Writer = cxWriter

		c.Next()

		cxWriter.flush()
	}
}

// cxSecWriter 拦截响应体，在 flush 时加密发出
type cxSecWriter struct {
	gin.ResponseWriter
	sess   *cxsec.Session
	buf    bytes.Buffer
	sealed bool
}

// Write ～sealed 之后再写就是白写，宁可报错也不能让数据悄悄消失哦～
func (w *cxSecWriter) Write(data []byte) (int, error) {
	if w.sealed {
		return 0, errors.New("cxsec: write after seal")
	}
	return w.buf.Write(data)
}

func (w *cxSecWriter) flush() {
	if w.sealed {
		return
	}
	w.sealed = true
	plain := w.buf.Bytes()

	curEpoch, _ := w.sess.EpochKeys()
	pre, post := cxsec.DeriveWhiteningKey(curEpoch, w.sess.ID)
	timeBlock := cxsec.CurrentTimeBlock()
	respNonce := cxsec.RandNonce()
	respCounter := w.sess.Counter()
	reqKey := cxsec.DeriveRequestKey(curEpoch, timeBlock, respCounter, respNonce)

	ciphertext, tag, err := cxsec.Encrypt(reqKey, pre, post, respNonce, respCounter, cxsec.UnixNowMS(), plain)
	if err != nil {
		w.ResponseWriter.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 组装响应: nonce(16) + ciphertext + tag(16) + sig(32)
	hmacKey := cxsec.DeriveHMACKey(curEpoch, timeBlock)
	respBody := make([]byte, 0, 16+len(ciphertext)+16+32)
	respBody = append(respBody, respNonce[:]...)
	respBody = append(respBody, ciphertext...)
	respBody = append(respBody, tag...)
	sig := cxsec.Sign(hmacKey, respBody)
	respBody = append(respBody, sig[:]...)

	w.ResponseWriter.Header().Set("Content-Type", "application/octet-stream")
	w.ResponseWriter.WriteHeaderNow()
	w.ResponseWriter.Write(respBody) //nolint:errcheck
}
