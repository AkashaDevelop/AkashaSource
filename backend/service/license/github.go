// ⚠️ REMOVABLE MODULE — 系统授权门禁
// GitHub OAuth 换 token / 查用户名 / 查组织成员身份，写法照抄 controller/oauth/github.go 的风格
package license

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type githubOAuthResponse struct {
	AccessToken string `json:"access_token"`
}

type githubUser struct {
	Login string `json:"login"`
}

// exchangeCode 用授权码换用户自己的 access token —— 这次要带 read:org 权限，
// 因为组织成员校验（首次授权时）要用这个 token 查"我自己"是不是组织成员，
// 用的是这个模块自己独立注册的 GitHub OAuth App，不是站点登录用的那个～
func exchangeCode(code string) (string, error) {
	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", nil)
	if err != nil {
		return "", err
	}
	q := req.URL.Query()
	q.Add("client_id", licenseGithubClientId)
	q.Add("client_secret", licenseGithubClientSecret)
	q.Add("code", code)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("换取 access token 失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token 接口返回 %d: %s", resp.StatusCode, string(body))
	}

	var oauthResp githubOAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&oauthResp); err != nil {
		return "", err
	}
	if oauthResp.AccessToken == "" {
		return "", fmt.Errorf("未能获取 access token")
	}
	return oauthResp.AccessToken, nil
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
		// ～组织开了 "OAuth App access restrictions"，得去 GitHub 组织设置里手动批准这个 App～
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

// checkOrgMembership 用内置的 gistToken（不是部署方自己的 token）去查目标账号是否为组织成员，
// 只给 7 天周期复核用（复核时手头没有那个人当初登录的 token 了，只能退而求其次）。
// ⚠️ 这个检查依赖 gistToken 所属账号本身也是该组织成员，否则 GitHub 会把请求重定向到
// "公开成员列表"接口，查不到"隐藏成员身份"的人（哪怕对方是所有者）。如果发现复核老是
// 误判成员已离开组织，去 GitHub 把 gistToken 对应的账号也拉进组织里就好了喵。
// 204 = 是成员，404 = 不是成员/对该 token 不可见
func checkOrgMembership(username string) (bool, error) {
	url := fmt.Sprintf("https://api.github.com/orgs/%s/members/%s", targetOrg, username)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+gistToken)
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
