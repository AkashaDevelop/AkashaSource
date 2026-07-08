// 系统授权门禁 — 对外门面函数
// 授权状态存储在签名后的 license.dat 文件中，不写数据库，防止直接改库绕过
package license

import (
	"fmt"
	"log"
	"os"
	"time"

	"STfreApi/common"
)

// GetLocalFingerprint 拿本实例的设备指纹
// 授权文件存在时从文件读取，不存在时生成新的
func GetLocalFingerprint() string {
	if ld := readLicenseFile(); ld != nil && ld.Fingerprint != "" {
		return ld.Fingerprint
	}
	return common.GetUUID()
}

// IsAuthorized 授权文件存在且签名有效才算已授权
func IsAuthorized() bool {
	if FeatureEnabled() && decryptedSecrets.decryptFailed {
		return false // 解密失败(fail-closed)
	}
	ld := readLicenseFile()
	if ld == nil || ld.GithubLogin == "" || ld.BoundAt == 0 {
		return false
	}
	return verifySignature(ld.GithubLogin, ld.Fingerprint, ld.BoundAt, ld.Signature)
}

// Status 给前端展示用的状态快照
type Status struct {
	FeatureEnabled bool   `json:"feature_enabled"`
	Authorized     bool   `json:"authorized"`
	GithubLogin    string `json:"github_login"`
	BoundAt        int64  `json:"bound_at"`
	LastCheck      int64  `json:"last_check"`
	Org            string `json:"org"`
}

func GetStatus() Status {
	if !FeatureEnabled() {
		return Status{FeatureEnabled: false}
	}
	ld := readLicenseFile()
	if ld == nil {
		return Status{FeatureEnabled: true, Authorized: false, Org: getSecretTargetOrg()}
	}
	return Status{
		FeatureEnabled: true,
		Authorized:     IsAuthorized(),
		GithubLogin:    ld.GithubLogin,
		BoundAt:        ld.BoundAt,
		LastCheck:      ld.LastCheck,
		Org:            getSecretTargetOrg(),
	}
}

// DeviceCodeInfo 返回给前端的设备码信息
type DeviceCodeInfo struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

func RequestDeviceFlow() (*DeviceCodeInfo, error) {
	if !FeatureEnabled() {
		return nil, fmt.Errorf("本部署未启用系统授权功能")
	}
	resp, err := requestDeviceCode()
	if err != nil {
		return nil, err
	}
	return &DeviceCodeInfo{
		DeviceCode:      resp.DeviceCode,
		UserCode:        resp.UserCode,
		VerificationURI: resp.VerificationURI,
		ExpiresIn:       resp.ExpiresIn,
		Interval:        resp.Interval,
	}, nil
}

func PollDeviceFlow(deviceCode string) (bool, error) {
	if !FeatureEnabled() {
		return false, fmt.Errorf("本部署未启用系统授权功能")
	}

	accessToken, pending, err := pollForToken(deviceCode)
	if err != nil {
		return false, err
	}
	if pending {
		return false, nil
	}

	if err := authorizeWithToken(accessToken); err != nil {
		return false, err
	}
	return true, nil
}

// authorizeWithToken 用 access token 完成完整授权流程
func authorizeWithToken(accessToken string) error {
	login, err := fetchGitHubLogin(accessToken)
	if err != nil {
		return err
	}

	isMember, err := checkOwnOrgMembership(accessToken, getSecretTargetOrg())
	if err != nil {
		return err
	}
	if !isMember {
		return fmt.Errorf("您（%s）不是 %s 组织成员，无权限使用本系统", login, getSecretTargetOrg())
	}

	// 已绑定了别的账号，不允许静默换绑
	existing := readLicenseFile()
	if existing != nil && existing.GithubLogin != "" && existing.GithubLogin != login && IsAuthorized() {
		return fmt.Errorf("本实例已绑定 GitHub 账号 %s，请先解绑后再用其他账号授权", existing.GithubLogin)
	}

	fp := GetLocalFingerprint()
	bindings, err := readBindings()
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	if e, ok := bindings[login]; ok && e.Fingerprint != fp {
		return fmt.Errorf("该 GitHub 账号已在其他部署实例授权，请先在原实例解绑后再试")
	}

	bindings[login] = gistBinding{Fingerprint: fp, BoundAt: now}
	if err := writeBindings(bindings); err != nil {
		return err
	}

	// 写入授权文件
	if err := writeLicenseFile(&licenseData{
		GithubLogin:         login,
		Fingerprint:         fp,
		BoundAt:             now,
		LastCheck:           now,
		RevalidateFailCount: 0,
	}); err != nil {
		return fmt.Errorf("写入授权文件失败: %w", err)
	}

	log.Printf("[license] 授权成功: github=%s", login)
	return nil
}

// Unbind 解绑：删 Gist 记录 + 删授权文件
func Unbind() error {
	ld := readLicenseFile()
	if ld == nil || ld.GithubLogin == "" {
		return fmt.Errorf("当前未绑定任何账号")
	}

	bindings, err := readBindings()
	if err != nil {
		return err
	}
	if existing, ok := bindings[ld.GithubLogin]; ok && existing.Fingerprint == ld.Fingerprint {
		delete(bindings, ld.GithubLogin)
		if err := writeBindings(bindings); err != nil {
			return err
		}
	}

	if err := deleteLicenseFile(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除授权文件失败: %w", err)
	}

	log.Printf("[license] 已解绑: github=%s", ld.GithubLogin)
	return nil
}

// Revalidate 周期复核：确认组织成员身份 + Gist 绑定仍然有效
func Revalidate() error {
	if !FeatureEnabled() || !IsAuthorized() {
		return nil
	}

	ld := readLicenseFile()
	if ld == nil || ld.GithubLogin == "" {
		log.Printf("[license] 复核发现授权文件异常，吊销授权")
		revokeAuthorization()
		return nil
	}

	isMember, err := checkOrgMembership(ld.GithubLogin)
	if err != nil {
		return incrementFailCount(ld)
	}
	if !isMember {
		log.Printf("[license] 复核发现 %s 已不在组织内，吊销授权", ld.GithubLogin)
		revokeAuthorization()
		return nil
	}

	bindings, err := readBindings()
	if err != nil {
		return incrementFailCount(ld)
	}
	binding, ok := bindings[ld.GithubLogin]
	if !ok || binding.Fingerprint != ld.Fingerprint {
		log.Printf("[license] 复核发现 Gist 绑定记录已变化，吊销授权")
		revokeAuthorization()
		return nil
	}

	// 复核通过，更新时间戳
	ld.LastCheck = time.Now().Unix()
	ld.RevalidateFailCount = 0
	if err := writeLicenseFile(ld); err != nil {
		return fmt.Errorf("更新授权文件失败: %w", err)
	}
	return nil
}

// revokeAuthorization 删除授权文件
func revokeAuthorization() {
	if err := deleteLicenseFile(); err != nil && !os.IsNotExist(err) {
		log.Printf("[license] 删除授权文件失败: %v", err)
	}
}

// incrementFailCount 复核网络失败时递增计数器，连续失败3次后吊销
func incrementFailCount(ld *licenseData) error {
	ld.RevalidateFailCount++
	if err := writeLicenseFile(ld); err != nil {
		return fmt.Errorf("更新失败计数失败: %w", err)
	}
	if ld.RevalidateFailCount >= 3 {
		log.Printf("[license] 复核连续失败 %d 次，强制吊销授权", ld.RevalidateFailCount)
		revokeAuthorization()
		return fmt.Errorf("复核连续失败 %d 次，已吊销授权", ld.RevalidateFailCount)
	}
	return fmt.Errorf("复核失败（第 %d 次），授权暂时保留", ld.RevalidateFailCount)
}
