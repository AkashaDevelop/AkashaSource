// 系统授权门禁 — 授权文件存储
// 授权文件使用 AES-256-GCM 加密存储，密钥从白盒派生，磁盘上无明文 JSON。
// HMAC 签名内嵌在加密 payload 中，解密后校验签名，双重保护。
package license

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const licenseFileName = "license.dat"

// licenseData 授权文件明文结构
type licenseData struct {
	GithubLogin         string `json:"github_login"`
	Fingerprint         string `json:"fingerprint"`
	BoundAt             int64  `json:"bound_at"`
	LastCheck           int64  `json:"last_check"`
	RevalidateFailCount int64  `json:"revalidate_fail_count"`
	Signature           string `json:"signature"`
}

// 加密文件格式: [32B nonce] [ciphertext+tag]
const fileNonceSize = 32

var (
	licenseMu     sync.RWMutex
	cachedLicense *licenseData
	cacheLoaded   bool
)

func licenseFilePath() string {
	// 优先使用工作目录，兼容 go run 开发模式和编译后的二进制
	if wd, err := os.Getwd(); err == nil {
		return filepath.Join(wd, licenseFileName)
	}
	exe, err := os.Executable()
	if err != nil {
		return licenseFileName
	}
	return filepath.Join(filepath.Dir(exe), licenseFileName)
}

// deriveFileKey 从白盒 HMAC 密钥派生授权文件加密密钥
// 使用 SHA256(hmacSecret || file-domain) 确保密钥与二进制绑定
//
//go:noinline
func deriveFileKey() []byte {
	secret := getSecretHmacSecret()
	h := sha256.New()
	h.Write(secret)
	h.Write([]byte{0x6c, 0x69, 0x63, 0x2e, 0x66, 0x69, 0x6c, 0x65}) // "lic.file" XOR'd
	return h.Sum(nil) // 32 bytes = AES-256
}

// encryptLicense 用 AES-256-GCM 加密授权数据
func encryptLicense(plaintext []byte) ([]byte, error) {
	key := deriveFileKey()
	defer zeroBytes(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, fileNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce[:gcm.NonceSize()], plaintext, nil)

	// 输出: nonce(32B) + ciphertext
	result := make([]byte, 0, fileNonceSize+len(ciphertext))
	result = append(result, nonce...)
	result = append(result, ciphertext...)
	return result, nil
}

// decryptLicense 用 AES-256-GCM 解密授权数据
func decryptLicense(blob []byte) ([]byte, error) {
	if len(blob) < fileNonceSize {
		return nil, fmt.Errorf("授权文件格式无效")
	}

	key := deriveFileKey()
	defer zeroBytes(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := blob[:gcm.NonceSize()]
	ciphertext := blob[fileNonceSize:]

	return gcm.Open(nil, nonce, ciphertext, nil)
}

// readLicenseFile 从磁盘读取、解密、验证授权文件
func readLicenseFile() *licenseData {
	licenseMu.RLock()
	if cacheLoaded {
		result := cachedLicense
		licenseMu.RUnlock()
		return result
	}
	licenseMu.RUnlock()

	licenseMu.Lock()
	defer licenseMu.Unlock()

	if cacheLoaded {
		return cachedLicense
	}

	blob, err := os.ReadFile(licenseFilePath())
	if err != nil {
		cacheLoaded = true
		return nil
	}

	plaintext, err := decryptLicense(blob)
	if err != nil {
		// 解密失败 = 文件被篡改或密钥不匹配
		cacheLoaded = true
		return nil
	}

	var ld licenseData
	if err := json.Unmarshal(plaintext, &ld); err != nil {
		cacheLoaded = true
		return nil
	}
	zeroBytes(plaintext)

	if !verifySignature(ld.GithubLogin, ld.Fingerprint, ld.BoundAt, ld.Signature) {
		cacheLoaded = true
		return nil
	}

	cachedLicense = &ld
	cacheLoaded = true
	return cachedLicense
}

// writeLicenseFile 签名 + 加密 + 原子写入
func writeLicenseFile(ld *licenseData) error {
	ld.Signature = computeSignature(ld.GithubLogin, ld.Fingerprint, ld.BoundAt)

	plaintext, err := json.Marshal(ld)
	if err != nil {
		return fmt.Errorf("序列化授权文件失败: %w", err)
	}

	blob, err := encryptLicense(plaintext)
	zeroBytes(plaintext)
	if err != nil {
		return fmt.Errorf("加密授权文件失败: %w", err)
	}

	path := licenseFilePath()
	tmpPath := path + ".tmp"

	if err := os.WriteFile(tmpPath, blob, 0600); err != nil {
		return fmt.Errorf("写入授权文件失败(%s): %w", path, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("重命名授权文件失败: %w", err)
	}

	licenseMu.Lock()
	cachedLicense = ld
	cacheLoaded = true
	licenseMu.Unlock()

	return nil
}

// deleteLicenseFile 删除授权文件
func deleteLicenseFile() error {
	licenseMu.Lock()
	defer licenseMu.Unlock()
	cacheLoaded = true
	cachedLicense = nil
	return os.Remove(licenseFilePath())
}
