// ⚠️ REMOVABLE MODULE — 系统授权门禁
// 给"已授权"状态加一层签名，堵住"直接改数据库/裸调 PUT /api/option 把 authorized 字段
// 改成 true 就能绕过整个授权流程"这个洞——签名密钥只在编译进二进制的常量里，不落库、
// 不经过任何接口暴露，篡改者改得动 Option 表里的字符串，但算不出对得上的签名。
package license

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// computeSignature 对"这次授权到底是谁、绑的哪个设备指纹、什么时候绑的"做一次签名
func computeSignature(login, fingerprint string, boundAt int64) string {
	payload := fmt.Sprintf("%s|%s|%d", login, fingerprint, boundAt)
	mac := hmac.New(sha256.New, hmacSecret)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// verifySignature 用常量时间比较，避免签名校验本身被时序攻击试出来
func verifySignature(login, fingerprint string, boundAt int64, signature string) bool {
	if signature == "" {
		return false
	}
	expected := computeSignature(login, fingerprint, boundAt)
	return hmac.Equal([]byte(expected), []byte(signature))
}
