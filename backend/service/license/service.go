// ⚠️ REMOVABLE MODULE — 系统授权门禁
// 对外的门面函数：绑定/解绑/查状态/周期复核，controller 层只调这几个
package license

import (
	"fmt"
	"log"
	"time"

	"STfreApi/common"
	"STfreApi/model"
)

// 本地状态用的 Option key，特意不放进 model/option.go 的常量表里——
// 保持这个模块自成一体，以后要整体移除的话，model/option.go 完全不用动喵
const (
	optKeyAuthorized  = "system_license_authorized"
	optKeyGithubLogin = "system_license_github_login"
	optKeyFingerprint = "system_license_fingerprint"
	optKeyBoundAt     = "system_license_bound_at"
	optKeyLastCheck   = "system_license_last_check"
	optKeySignature   = "system_license_signature" // HMAC 签名，防止直接改库伪造授权状态
)

func getOption(key string) string {
	common.OptionLock.RLock()
	defer common.OptionLock.RUnlock()
	return common.OptionMap[key]
}

func setOption(key, value string) {
	var opt model.Option
	common.DB.Where(model.Option{Key: key}).Assign(model.Option{Value: value}).FirstOrCreate(&opt)
	common.UpdateOptionMap(key, value)
}

// GetLocalFingerprint 拿本实例的设备指纹，第一次用到时生成并持久化，之后终身复用
func GetLocalFingerprint() string {
	fp := getOption(optKeyFingerprint)
	if fp != "" {
		return fp
	}
	fp = common.GetUUID()
	setOption(optKeyFingerprint, fp)
	return fp
}

// IsAuthorized 本地是否已标记为授权通过——光有 authorized=="true" 还不够，
// 必须连带的 HMAC 签名也对得上，否则就是有人直接改库/裸调 PUT /api/option
// 伪造出来的假状态，一律当作未授权喵（签名密钥只在编译进二进制的常量里，不落库）
func IsAuthorized() bool {
	if getOption(optKeyAuthorized) != "true" {
		return false
	}
	login := getOption(optKeyGithubLogin)
	fp := getOption(optKeyFingerprint)
	boundAt, _ := parseInt64(getOption(optKeyBoundAt))
	sig := getOption(optKeySignature)
	return verifySignature(login, fp, boundAt, sig)
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
	boundAt, _ := parseInt64(getOption(optKeyBoundAt))
	lastCheck, _ := parseInt64(getOption(optKeyLastCheck))
	return Status{
		FeatureEnabled: true,
		Authorized:     IsAuthorized(),
		GithubLogin:    getOption(optKeyGithubLogin),
		BoundAt:        boundAt,
		LastCheck:      lastCheck,
		Org:            targetOrg,
	}
}

func parseInt64(s string) (int64, error) {
	var n int64
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// DeviceCodeInfo 返回给前端的设备码信息
type DeviceCodeInfo struct {
	DeviceCode      string `json:"device_code"`       // 设备码（轮询时回传，一次性、15分钟过期）
	UserCode        string `json:"user_code"`         // 用户需要输入的短码
	VerificationURI string `json:"verification_uri"`   // 用户打开的页面
	ExpiresIn       int    `json:"expires_in"`          // 设备码有效期（秒）
	Interval        int    `json:"interval"`            // 轮询间隔（秒）
}

// RequestDeviceFlow 发起 Device Flow，返回设备码信息供前端展示
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

// PollDeviceFlow 轮询 GitHub 换取 token，成功后自动完成授权绑定
// 返回 (completed, error)：completed=true 表示授权成功，error 非空表示失败
func PollDeviceFlow(deviceCode string) (bool, error) {
	if !FeatureEnabled() {
		return false, fmt.Errorf("本部署未启用系统授权功能")
	}

	accessToken, pending, err := pollForToken(deviceCode)
	if err != nil {
		return false, err
	}
	if pending {
		// 用户还没完成授权，前端应继续轮询
		return false, nil
	}

	// 拿到 token，完成授权绑定
	if err := authorizeWithToken(accessToken); err != nil {
		return false, err
	}
	return true, nil
}

// authorizeWithToken 用 access token 完成完整授权流程：
// 查用户名 → 查组织成员 → 读/写 Gist → 落本地状态
func authorizeWithToken(accessToken string) error {
	login, err := fetchGitHubLogin(accessToken)
	if err != nil {
		return err
	}

	isMember, err := checkOwnOrgMembership(accessToken, targetOrg)
	if err != nil {
		return err
	}
	if !isMember {
		return fmt.Errorf("您（%s）不是 %s 组织成员，无权限使用本系统", login, targetOrg)
	}

	// 本实例已经绑定了别的账号，不允许静默换绑，得先解绑
	if existingLogin := getOption(optKeyGithubLogin); existingLogin != "" && existingLogin != login && IsAuthorized() {
		return fmt.Errorf("本实例已绑定 GitHub 账号 %s，请先解绑后再用其他账号授权", existingLogin)
	}

	fp := GetLocalFingerprint()
	bindings, err := readBindings()
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	if existing, ok := bindings[login]; ok && existing.Fingerprint != fp {
		return fmt.Errorf("该 GitHub 账号已在其他部署实例授权，请先在原实例解绑后再试")
	}

	bindings[login] = gistBinding{Fingerprint: fp, BoundAt: now}
	if err := writeBindings(bindings); err != nil {
		return err
	}

	setOption(optKeyAuthorized, "true")
	setOption(optKeyGithubLogin, login)
	setOption(optKeyBoundAt, fmt.Sprintf("%d", now))
	setOption(optKeyLastCheck, fmt.Sprintf("%d", now))
	setOption(optKeySignature, computeSignature(login, fp, now))
	log.Printf("[license] 授权成功: github=%s", login)
	return nil
}

// Unbind 解绑：只删本机指纹对应的那条 Gist 记录，避免误删别的实例的绑定
func Unbind() error {
	login := getOption(optKeyGithubLogin)
	if login == "" {
		return fmt.Errorf("当前未绑定任何账号")
	}
	fp := getOption(optKeyFingerprint)

	bindings, err := readBindings()
	if err != nil {
		return err
	}
	if existing, ok := bindings[login]; ok && existing.Fingerprint == fp {
		delete(bindings, login)
		if err := writeBindings(bindings); err != nil {
			return err
		}
	}

	setOption(optKeyAuthorized, "false")
	setOption(optKeyGithubLogin, "")
	setOption(optKeyBoundAt, "0")
	setOption(optKeySignature, "")
	log.Printf("[license] 已解绑: github=%s", login)
	return nil
}

// Revalidate 供 7 天周期任务调用：重新确认组织成员身份 + Gist 绑定仍然有效，
// 任一条件不满足就把本地授权状态打回 false（覆盖"作者去 Gist 里删掉记录来吊销授权"
// 和"账号被踢出组织"两种场景）
func Revalidate() error {
	if !FeatureEnabled() || !IsAuthorized() {
		return nil
	}
	login := getOption(optKeyGithubLogin)
	fp := getOption(optKeyFingerprint)
	if login == "" {
		// 正常走完授权的状态不可能出现 login 为空，这是明显的异常/篡改状态
		log.Printf("[license] 复核发现绑定账号为空（异常状态），取消本机授权")
		setOption(optKeyAuthorized, "false")
		return nil
	}

	isMember, err := checkOrgMembership(login)
	if err != nil {
		return err
	}
	if !isMember {
		log.Printf("[license] 复核发现 %s 已不在组织内，取消本机授权", login)
		setOption(optKeyAuthorized, "false")
		return nil
	}

	bindings, err := readBindings()
	if err != nil {
		return err
	}
	binding, ok := bindings[login]
	if !ok || binding.Fingerprint != fp {
		log.Printf("[license] 复核发现 Gist 绑定记录已变化，取消本机授权")
		setOption(optKeyAuthorized, "false")
		return nil
	}

	setOption(optKeyLastCheck, fmt.Sprintf("%d", time.Now().Unix()))
	return nil
}
