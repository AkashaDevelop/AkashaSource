package moderation

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

// hmacSha256 ～用密钥给消息盖一个只有腾讯云认得的火漆印～
func hmacSha256(key, msg []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(msg)
	return mac.Sum(nil)
}

// sha256Hex ～把内容捏成一串看不出原样的哈希小饼干～
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// signTC3 ～按照腾讯云 TC3-HMAC-SHA256 的配方，一步步炖出 Authorization 头～
func signTC3(secretId, secretKey, service, action, payload string, timestamp int64) string {
	const algorithm = "TC3-HMAC-SHA256"
	host := service + ".tencentcloudapi.com"
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")

	canonicalHeaders := "content-type:application/json\nhost:" + host + "\nx-tc-action:" + strings.ToLower(action) + "\n"
	signedHeaders := "content-type;host;x-tc-action"
	canonicalRequest := "POST\n/\n\n" + canonicalHeaders + "\n" + signedHeaders + "\n" + sha256Hex(payload)

	credentialScope := date + "/" + service + "/tc3_request"
	stringToSign := algorithm + "\n" + strconv.FormatInt(timestamp, 10) + "\n" + credentialScope + "\n" + sha256Hex(canonicalRequest)

	secretDate := hmacSha256([]byte("TC3"+secretKey), []byte(date))
	secretService := hmacSha256(secretDate, []byte(service))
	secretSigning := hmacSha256(secretService, []byte("tc3_request"))
	signature := hex.EncodeToString(hmacSha256(secretSigning, []byte(stringToSign)))

	return algorithm + " Credential=" + secretId + "/" + credentialScope +
		", SignedHeaders=" + signedHeaders + ", Signature=" + signature
}
