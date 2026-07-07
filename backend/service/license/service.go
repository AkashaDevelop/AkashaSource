// ⚠️ REMOVABLE MODULE — 系统授权门禁
// 对外的门面函数：绑定/解绑/查状态/周期复核，controller 层只调这几个
package license

import (
	"fmt"
	"log"
	"net/url"
	"strings"
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

// BuildAuthorizeURL 生成一次性 state 并拼好 GitHub 授权跳转链接（用这个模块自己独立的 OAuth App）
func BuildAuthorizeURL() (string, error) {
	if !FeatureEnabled() {
		return "", fmt.Errorf("本部署未启用系统授权功能")
	}
	state := generateState()
	// ～必须显式带上 redirect_uri，且要和 GitHub OAuth App 里注册的回调地址一致，
	// 否则 GitHub 会直接拒绝或跳错地方喵～
	systemUrl := strings.TrimRight(common.OptionMap[model.OptionKeySystemUrl], "/")
	redirectUri := systemUrl + "/api/system-license/github/callback"
	return "https://github.com/login/oauth/authorize?client_id=" + licenseGithubClientId +
		"&scope=read:org&state=" + state + "&redirect_uri=" + url.QueryEscape(redirectUri), nil
}

// HandleCallback 校验 state 后完成授权绑定流程
func HandleCallback(state, code string) error {
	if !verifyState(state) {
		return fmt.Errorf("授权请求已过期或无效，请重新发起")
	}
	if code == "" {
		return fmt.Errorf("缺少授权码")
	}
	return Authorize(code)
}

// Authorize 完整的授权流程：换 token → 查用户名 → 查组织成员 → 读/写 Gist → 落本地状态
func Authorize(code string) error {
	if !FeatureEnabled() {
		return fmt.Errorf("本部署未启用系统授权功能")
	}

	accessToken, err := exchangeCode(code)
	if err != nil {
		return err
	}
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
		// 正常走完 Authorize() 的状态不可能出现 login 为空，这是明显的异常/篡改状态，
		// 之前这里直接 return nil 什么都不做是个 bug——会让篡改后的状态永远得不到纠正
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
