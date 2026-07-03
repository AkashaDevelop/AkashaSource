// +build js,wasm

package main

// ～宸汐御安全 WASM 运行时：所有加密逻辑在这里，JS 只能调用，看不到内部～
// 编译: GOOS=js GOARCH=wasm go build -o cx_runtime.wasm .

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/bits"
	"sync"
	"syscall/js"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/sha3"
)

// ─── 会话存储 ──────────────────────────────────────────────────────────────

type jsSession struct {
	epochKey     [32]byte
	prevEpochKey [32]byte
	epochIndex   int64
	fpHash       [32]byte
	counter      uint32
	sessionID    string
	nonceSet     map[[16]byte]bool
	mu           sync.Mutex
}

var (
	handleMu  sync.Mutex
	nextHandle int32 = 1
	handles    = make(map[int32]*jsSession)
)

func newHandle(sess *jsSession) int32 {
	handleMu.Lock()
	defer handleMu.Unlock()
	h := nextHandle
	nextHandle++
	handles[h] = sess
	return h
}

func getHandle(h int32) *jsSession {
	handleMu.Lock()
	defer handleMu.Unlock()
	return handles[h]
}

// ─── 导出函数 ──────────────────────────────────────────────────────────────

func main() {
	js.Global().Set("__cx_init", js.FuncOf(jsInit))
	js.Global().Set("__cx_solve_pow", js.FuncOf(jsSolvePow))
	js.Global().Set("__cx_encrypt", js.FuncOf(jsEncrypt))
	js.Global().Set("__cx_decrypt", js.FuncOf(jsDecrypt))
	js.Global().Set("__cx_destroy", js.FuncOf(jsDestroy))
	js.Global().Set("__cx_fp", js.FuncOf(jsFingerprint))
	// 阻塞，保持 WASM 运行
	select {}
}

// jsInit: 接收服务端公钥(hex) + 握手响应字节(Uint8Array)，建立会话
// 返回: sessionHandle(number) 或 -1(失败)
func jsInit(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return js.ValueOf(-1)
	}
	serverPubHex := args[0].String()
	handshakeRespJS := args[1]

	respLen := handshakeRespJS.Get("length").Int()
	if respLen < 16 {
		return js.ValueOf(-1)
	}
	resp := make([]byte, respLen)
	js.CopyBytesToGo(resp, handshakeRespJS)

	serverPubBytes, err := hex.DecodeString(serverPubHex)
	if err != nil {
		return js.ValueOf(-1)
	}

	// 解析服务端握手响应: [8B hint][4B counter_init][4B ttl]
	var hint [8]byte
	copy(hint[:], resp[:8])

	// 客户端临时 ECDH 密钥对（在握手时已生成并发送，这里重新从存储恢复）
	// 此处简化：使用 args[2] 传入的 clientPrivHex
	if len(args) < 3 {
		return js.ValueOf(-1)
	}
	clientPrivHex := args[2].String()
	clientPrivBytes, err := hex.DecodeString(clientPrivHex)
	if err != nil {
		return js.ValueOf(-1)
	}

	curve := ecdh.P256()
	clientPriv, err := curve.NewPrivateKey(clientPrivBytes)
	if err != nil {
		return js.ValueOf(-1)
	}
	serverPub, err := curve.NewPublicKey(serverPubBytes)
	if err != nil {
		return js.ValueOf(-1)
	}

	// ECDH 共享密钥
	shared, err := clientPriv.ECDH(serverPub)
	if err != nil {
		return js.ValueOf(-1)
	}

	// 派生纪元0密钥（与服务端 DeriveSessionKey 相同逻辑）
	var fpHash [32]byte
	if len(args) >= 4 {
		fpHex := args[3].String()
		fp, _ := hex.DecodeString(fpHex)
		copy(fpHash[:], fp)
	}

	epoch0 := deriveEpoch0(shared, fpHash)

	sid := fmt.Sprintf("%x%x", hint[:], epoch0[:4])
	sess := &jsSession{
		epochKey:   epoch0,
		epochIndex: currentEpochV2(),
		fpHash:     fpHash,
		counter:    binary.LittleEndian.Uint32(resp[8:12]),
		sessionID:  sid,
		nonceSet:   make(map[[16]byte]bool),
	}
	handle := newHandle(sess)
	return js.ValueOf(int(handle))
}

// jsSolvePow: 求解 PoW 挑战
// args[0]: JSON 字符串 {nonce: hex, diff: number, ts: number}
// 返回: JSON 字符串 {solution: hex8B, argon_verify: hex32B} 或 ""(失败)
func jsSolvePow(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return js.ValueOf("")
	}
	var challenge struct {
		Nonce string `json:"nonce"`
		Diff  int    `json:"diff"`
		TS    int64  `json:"ts"`
	}
	if err := json.Unmarshal([]byte(args[0].String()), &challenge); err != nil {
		return js.ValueOf("")
	}

	nonceBytes, err := hex.DecodeString(challenge.Nonce)
	if err != nil || len(nonceBytes) != 32 {
		return js.ValueOf("")
	}
	var nonce [32]byte
	copy(nonce[:], nonceBytes)

	if challenge.Diff < 1 || challenge.Diff > 31 {
		challenge.Diff = 20
	}
	// challenge.TS 已经是服务端的 timeHint（unix/60），直接用不需要再除
	timeHint := challenge.TS

	solution, ok := solvePow(nonce, uint8(challenge.Diff), timeHint)
	if !ok {
		return js.ValueOf("")
	}

	// 计算 argon2 验证字节
	input := make([]byte, 32+8+8)
	copy(input[:32], nonce[:])
	binary.LittleEndian.PutUint64(input[32:40], solution)
	binary.LittleEndian.PutUint64(input[40:48], uint64(timeHint))
	sha3Hash := sha3.Sum256(input)
	argHash := argon2.IDKey(sha3Hash[:], nonce[:8], 1, 128, 1, 32)

	var solBytes [8]byte
	binary.LittleEndian.PutUint64(solBytes[:], solution)
	result := map[string]string{
		"solution":     hex.EncodeToString(solBytes[:]),
		"argon_verify": hex.EncodeToString(argHash),
	}
	out, _ := json.Marshal(result)
	return js.ValueOf(string(out))
}

// jsEncrypt: 加密请求体
// args[0]: handle(number), args[1]: plaintext Uint8Array
// 返回: Uint8Array (完整 wire format body) 或 null
func jsEncrypt(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return js.Null()
	}
	handle := int32(args[0].Int())
	sess := getHandle(handle)
	if sess == nil {
		return js.Null()
	}

	plainLen := args[1].Get("length").Int()
	plaintext := make([]byte, plainLen)
	js.CopyBytesToGo(plaintext, args[1])

	sess.mu.Lock()
	sess.ratchetIfNeeded()
	epochKey := sess.epochKey
	counter := sess.counter
	sess.mu.Unlock()

	// 生成 nonce
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return js.Null()
	}

	timeBlock := currentTimeBlock()
	reqKey := deriveRequestKey(epochKey, timeBlock, counter, nonce)
	pre, post := deriveWhiteningKey(epochKey, sess.sessionID)

	ciphertext, tag, err := encrypt(reqKey, pre, post, nonce, counter, nowMS(), plaintext)
	if err != nil {
		return js.Null()
	}

	// 服务端 hint（从 sessionID 中提取，会话建立时存储）
	hintBytes, _ := hex.DecodeString(sess.sessionID[:16]) // 前8B hint

	// Wire format: hint(8) + nonce(16) + ciphertext + tag(16) + sig(32)
	body := make([]byte, 0, 8+16+len(ciphertext)+16+32)
	body = append(body, hintBytes...)
	body = append(body, nonce[:]...)
	body = append(body, ciphertext...)
	body = append(body, tag...)

	hmacKey := deriveHMACKey(epochKey, timeBlock)
	sig := sign(hmacKey, body)
	body = append(body, sig[:]...)

	result := js.Global().Get("Uint8Array").New(len(body))
	js.CopyBytesToJS(result, body)
	return result
}

// jsDecrypt: 解密响应体
// args[0]: handle(number), args[1]: responseBody Uint8Array
// 返回: Uint8Array (plaintext) 或 null
func jsDecrypt(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return js.Null()
	}
	handle := int32(args[0].Int())
	sess := getHandle(handle)
	if sess == nil {
		return js.Null()
	}

	respLen := args[1].Get("length").Int()
	if respLen < 16+16+32 {
		return js.Null()
	}
	respBody := make([]byte, respLen)
	js.CopyBytesToGo(respBody, args[1])

	// 响应 wire format: nonce(16) + ciphertext + tag(16) + sig(32)
	var nonce [16]byte
	copy(nonce[:], respBody[:16])
	sigStart := respLen - 32
	tagStart := sigStart - 16
	ciphertext := respBody[16:tagStart]
	tag := respBody[tagStart:sigStart]
	var sig [32]byte
	copy(sig[:], respBody[sigStart:])

	sess.mu.Lock()
	sess.ratchetIfNeeded()
	curEpoch := sess.epochKey
	prevEpoch := sess.prevEpochKey
	sess.mu.Unlock()

	timeBlock := currentTimeBlock()
	verified := false
	var verifiedEpoch [32]byte
verifyRespLoop:
	for _, ek := range [][32]byte{curEpoch, prevEpoch} {
		for _, tb := range []int64{timeBlock, timeBlock - 1} {
			hmacKey := deriveHMACKey(ek, tb)
			if verify(hmacKey, respBody[:sigStart], sig) {
				verified = true
				verifiedEpoch = ek
				break verifyRespLoop
			}
		}
	}
	if !verified {
		return js.Null()
	}

	sess.mu.Lock()
	counter := sess.counter
	sess.mu.Unlock()

	reqKey := deriveRequestKey(verifiedEpoch, timeBlock, counter, nonce)
	pre, post := deriveWhiteningKey(verifiedEpoch, sess.sessionID)
	payload, _, _, err := decryptResp(reqKey, pre, post, nonce, ciphertext, tag, 60000)
	if err != nil {
		return js.Null()
	}

	// 推进计数器
	sess.mu.Lock()
	sess.counter++
	sess.mu.Unlock()

	result := js.Global().Get("Uint8Array").New(len(payload))
	js.CopyBytesToJS(result, payload)
	return result
}

// jsDestroy: 销毁会话
func jsDestroy(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return js.Undefined()
	}
	handle := int32(args[0].Int())
	handleMu.Lock()
	delete(handles, handle)
	handleMu.Unlock()
	return js.Undefined()
}

// jsFingerprint: 计算环境指纹（在 JS 侧收集，WASM 侧哈希）
func jsFingerprint(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return js.Null()
	}
	raw := args[0].String()
	h := sha3.Sum256([]byte(raw))
	result := js.Global().Get("Uint8Array").New(32)
	js.CopyBytesToJS(result, h[:])
	return result
}

// ─── 内嵌算法实现（与服务端完全对称）─────────────────────────────────────

func solvePow(nonce [32]byte, difficulty uint8, timeHint int64) (uint64, bool) {
	if difficulty == 0 || difficulty > 31 {
		difficulty = 20
	}
	thres := uint32(1) << (32 - difficulty)
	looseThres := uint32(1) << (32 - difficulty + 2)

	for sol := uint64(0); sol < 1<<40; sol++ {
		input := make([]byte, 48)
		copy(input[:32], nonce[:])
		binary.LittleEndian.PutUint64(input[32:40], sol)
		binary.LittleEndian.PutUint64(input[40:48], uint64(timeHint))

		h := sha3.Sum256(input)
		v := binary.BigEndian.Uint32(h[:4])
		if v >= thres {
			continue
		}
		argHash := argon2.IDKey(h[:], nonce[:8], 1, 128, 1, 8)
		av := binary.BigEndian.Uint32(argHash[:4])
		if av < looseThres {
			return sol, true
		}
	}
	return 0, false
}

func currentTimeBlock() int64 {
	return nowSec() / 30
}
func nowSec() int64 {
	return int64(js.Global().Get("Date").New().Call("getTime").Int()) / 1000
}
func nowMS() int64 {
	return int64(js.Global().Get("Date").New().Call("getTime").Int())
}

// ～deriveRequestKey / deriveHMACKey / deriveWhiteningKey 已搬去 kdf.go 啦，这里不再重复啦～

func sign(hmacKey [32]byte, body []byte) [32]byte {
	// HMAC-SHA3-256
	mac := sha3.New256()
	// 内填充
	ipad := make([]byte, 32)
	opad := make([]byte, 32)
	for i := 0; i < 32; i++ {
		ipad[i] = hmacKey[i] ^ 0x36
		opad[i] = hmacKey[i] ^ 0x5c
	}
	mac.Write(ipad)
	mac.Write(body)
	inner := mac.Sum(nil)

	mac2 := sha3.New256()
	mac2.Write(opad)
	mac2.Write(inner)
	var sig [32]byte
	copy(sig[:], mac2.Sum(nil))
	return sig
}

func verify(hmacKey [32]byte, body []byte, sig [32]byte) bool {
	expected := sign(hmacKey, body)
	return bits.OnesCount64(uint64(expected[0]^sig[0])|
		uint64(expected[1]^sig[1])<<8|
		uint64(expected[2]^sig[2])<<16|
		uint64(expected[3]^sig[3])<<24) == 0 &&
		constTimeEqual(expected[:], sig[:])
}

func constTimeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

func xorWithKey(data, key []byte) []byte {
	out := make([]byte, len(data))
	kl := len(key)
	for i, b := range data {
		out[i] = b ^ key[i%kl]
	}
	return out
}

// encrypt 对称加密（与服务端 cipher.go 完全对称）
func encrypt(reqKey, pre, post [32]byte, nonce [16]byte, counter uint32, tsMS int64, plaintext []byte) (ciphertext, tag []byte, err error) {
	// 随机 padding
	padBuf := make([]byte, 1)
	rand.Read(padBuf) //nolint:errcheck
	padLen := int(padBuf[0] & 0x7F)
	padding := make([]byte, padLen)
	rand.Read(padding) //nolint:errcheck

	inner := make([]byte, 4+8+4+len(plaintext)+padLen+1)
	off := 0
	binary.LittleEndian.PutUint32(inner[off:], counter); off += 4
	binary.LittleEndian.PutUint64(inner[off:], uint64(tsMS)); off += 8
	binary.LittleEndian.PutUint32(inner[off:], uint32(len(plaintext))); off += 4
	copy(inner[off:], plaintext); off += len(plaintext)
	copy(inner[off:], padding); off += padLen
	inner[off] = byte(padLen)

	whitened := xorWithKey(inner, pre[:])

	// AES-256-GCM（使用 crypto/aes）
	sealed, err := aes256GCMEncrypt(reqKey, nonce[:12], whitened)
	if err != nil {
		return nil, nil, err
	}
	ciphertext = xorWithKey(sealed[:len(sealed)-16], post[:])
	tag = sealed[len(sealed)-16:]
	return
}

// decryptResp 解密响应（与服务端 Decrypt 对称，但不验证 counter）
func decryptResp(reqKey, pre, post [32]byte, nonce [16]byte, ciphertext, tag []byte, allowedSkewMS int64) (payload []byte, counter uint32, tsMS int64, err error) {
	deCiphered := xorWithKey(ciphertext, post[:])
	combined := append(deCiphered, tag...)
	whitened, err := aes256GCMDecrypt(reqKey, nonce[:12], combined)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("decrypt failed")
	}
	inner := xorWithKey(whitened, pre[:])

	if len(inner) < 4+8+4+1 {
		return nil, 0, 0, fmt.Errorf("inner block too short")
	}
	off := 0
	counter = binary.LittleEndian.Uint32(inner[off:]); off += 4
	tsMS = int64(binary.LittleEndian.Uint64(inner[off:])); off += 8
	payLen := int(binary.LittleEndian.Uint32(inner[off:])); off += 4

	now := nowMS()
	if diff := now - tsMS; diff < -allowedSkewMS || diff > allowedSkewMS {
		return nil, 0, 0, fmt.Errorf("timestamp out of window")
	}
	if off+payLen > len(inner)-1 {
		return nil, 0, 0, fmt.Errorf("payload length overflow")
	}
	payload = inner[off : off+payLen]
	return
}

// aes256GCMEncrypt 使用 AES-256-GCM 加密，返回 ciphertext+tag
func aes256GCMEncrypt(key [32]byte, nonce12 []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, 12)
	if err != nil {
		return nil, err
	}
	return gcm.Seal(nil, nonce12, plaintext, nil), nil
}

// aes256GCMDecrypt 使用 AES-256-GCM 解密，输入 ciphertext+tag
func aes256GCMDecrypt(key [32]byte, nonce12 []byte, combined []byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, 12)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce12, combined, nil)
}
