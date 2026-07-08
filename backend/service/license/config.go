// 系统授权门禁 — 功能启用检查
package license

// FeatureEnabled 返回 true 时门禁中间件生效。
// 解密失败(二进制被篡改)时返回 true(fail-closed)：
// 门禁激活但 IsAuthorized() 返回 false，拒绝所有非豁免请求。
func FeatureEnabled() bool {
	ensureInit()
	if decryptedSecrets.decryptFailed {
		return true // fail-closed: 门禁激活，阻止所有访问
	}
	return decryptedSecrets.targetOrg != "" && decryptedSecrets.gistId != "" &&
		decryptedSecrets.gistToken != "" && len(decryptedSecrets.hmacSecret) > 0 &&
		decryptedSecrets.licenseGithubClientId != "" && decryptedSecrets.licenseGithubClientSecret != ""
}
