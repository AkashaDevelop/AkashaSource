package common

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
)

// JwtSecret ～签发和校验 Token 用的秘密咒语，启动时才会真正被赋值哦～
var JwtSecret []byte

const (
	jwtSecretEnvKey  = "AKASHA_JWT_SECRET"
	jwtSecretKeyFile = "jwt_secret.key"
)

// InitJwtSecret ～给系统准备一把独一无二的钥匙：环境变量优先，其次是上次生成的存档，都没有就现场变一把新的～
func InitJwtSecret() {
	if secret := os.Getenv(jwtSecretEnvKey); secret != "" {
		JwtSecret = []byte(secret)
		log.Printf("[JWT] 已从环境变量 %s 加载密钥", jwtSecretEnvKey)
		return
	}

	if content, err := os.ReadFile(jwtSecretKeyFile); err == nil && len(content) > 0 {
		secret := string(content)
		JwtSecret = []byte(secret)
		os.Setenv(jwtSecretEnvKey, secret)
		log.Printf("[JWT] 已从本地密钥文件 %s 加载密钥", jwtSecretKeyFile)
		return
	}

	secret, err := generateRandomSecret()
	if err != nil {
		log.Fatalf("[JWT] 随机密钥生成失败: %v", err)
	}
	if err := os.WriteFile(jwtSecretKeyFile, []byte(secret), 0600); err != nil {
		log.Fatalf("[JWT] 密钥持久化失败: %v", err)
	}
	os.Setenv(jwtSecretEnvKey, secret)
	JwtSecret = []byte(secret)
	log.Printf("[JWT] 未检测到已有密钥，已随机生成新密钥并写入环境变量 %s 与本地文件 %s；多实例部署请显式设置该环境变量以保持一致", jwtSecretEnvKey, jwtSecretKeyFile)
}

// generateRandomSecret ～用密码学安全的随机数捏一把 32 字节长的新钥匙～
func generateRandomSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
