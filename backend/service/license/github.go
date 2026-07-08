// ⚠️ REMOVABLE MODULE — 系统授权门禁
// GitHub Device Flow（RFC 8628）—— 无需回调地址，用户在 github.com/login/device 输入设备码完成授权
package license

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type githubUser struct {
	Login string `json:"login"`
}

// DeviceCodeResponse Device Flow 第一步返回的设备码信息
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`         // 用户需要输入的短码，如 "WDJB-MJHT"
	VerificationURI string `json:"verification_uri"`   // 用户打开的页面，如 https://github.com/login/device
	ExpiresIn       int    `json:"expires_in"`          // 设备码有效期（秒）
	Interval        int    `json:"interval"`            // 轮询最小间隔（秒）
}

// devicePollResponse 轮询换 token 时的响应
type devicePollResponse struct {
	AccessToken string `json:"access_token"`
	Error       string `json:"error"` // authorization_pending / slow_down / expired_token / access_denied
}

// requestDeviceCode 向 GitHub 请求设备码，返回给前端展示
func requestDeviceCode() (*DeviceCodeResponse, error) {
	form := url.Values{}
	form.Set("client_id", getSecretOauthClientId())
	form.Set("scope", "read:org")
	req, _ := http.NewRequest("POST", "https://github.com/login/device/code", strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求设备码失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("设备码接口返回 %d: %s", resp.StatusCode, string(raw))
	}

	var result DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.DeviceCode == "" || result.UserCode == "" {
		return nil, fmt.Errorf("GitHub 返回的设备码不完整")
	}
	return &result, nil
}

// pollForToken 轮询 GitHub 换取 access_token
// 返回 (token, pending, error)：
//   - pending=true 表示用户还没完成授权，调用方应等待 interval 秒后重试
//   - token 非空表示授权成功
//   - error 表示不可恢复的错误（过期/拒绝/网络异常等）
func pollForToken(deviceCode string) (token string, pending bool, err error) {
	form := url.Values{}
	form.Set("client_id", getSecretOauthClientId())
	form.Set("device_code", deviceCode)
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	req, _ := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("轮询 token 失败: %w", err)
	}
	defer resp.Body.Close()

	var result devicePollResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", false, err
	}

	switch {
	case result.AccessToken != "":
		return result.AccessToken, false, nil
	case result.Error == "authorization_pending":
		return "", true, nil
	case result.Error == "slow_down":
		// GitHub 要求放慢轮询，调用方自行处理
		return "", true, nil
	case result.Error == "expired_token":
		return "", false, fmt.Errorf("设备码已过期，请重新发起授权")
	case result.Error == "access_denied":
		return "", false, fmt.Errorf("用户拒绝了授权请求")
	default:
		return "", false, fmt.Errorf("未知的授权状态: %s", result.Error)
	}
}

// fetchGitHubLogin 用刚换到的 access token 查一下"这次登录的到底是谁"
func fetchGitHubLogin(accessToken string) (string, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("获取 GitHub 用户信息失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("用户信息接口返回 %d: %s", resp.StatusCode, string(body))
	}

	var u githubUser
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return "", err
	}
	if u.Login == "" {
		return "", fmt.Errorf("未能获取 GitHub 用户名")
	}
	return u.Login, nil
}

// checkOwnOrgMembership 用刚登录那个人自己的 token 查"我自己是不是这个组织的成员"——
// 查自己不受"公开/隐藏成员身份"设置影响，一定准确，首次授权时用这个～
func checkOwnOrgMembership(accessToken, org string) (bool, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/user/memberships/orgs/"+org, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("查询组织成员身份失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode == http.StatusForbidden {
		return false, fmt.Errorf(
			"%s 组织开启了 OAuth App 访问限制，需要先手动批准这个 App：打开 "+
				"https://github.com/organizations/%s/settings/oauth_application_policy，"+
				"在 Pending requests 里批准（或用 Client ID 手动允许），批准后重新走一遍授权即可",
			org, org,
		)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("组织成员接口返回 %d: %s", resp.StatusCode, string(body))
	}

	var membership struct {
		State string `json:"state"` // "active" 或 "pending"（受邀但还没确认加入）
	}
	if err := json.NewDecoder(resp.Body).Decode(&membership); err != nil {
		return false, err
	}
	return membership.State == "active", nil
}

// checkOrgMembership 用内置的 gisttoken（不是部署方自己的 token）去查目标账号是否为组织成员，
// 只给 7 天周期复核用（复核时手头没有那个人当初登录的 token 了，只能退而求其次）。
// ⚠️ 这个检查依赖 gistToken 所属账号本身也是该组织成员，否则 GitHub 会把请求重定向到
// "公开成员列表"接口，查不到"隐藏成员身份"的人（哪怕对方是所有者）。
// 204 = 是成员，404 = 不是成员/对该 token 不可见
func checkOrgMembership(username string) (bool, error) {
	url := fmt.Sprintf("https://api.github.com/orgs/%s/members/%s", getSecretTargetOrg(), username)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+getSecretGistToken())
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("查询组织成员身份失败: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("组织成员接口返回 %d: %s", resp.StatusCode, string(body))
	}
}
