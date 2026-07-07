package oauth

import (
	"STfreApi/common"
	"STfreApi/model"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type WechatAccessTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenId       string `json:"openid"`
	Scope        string `json:"scope"`
	UnionId      string `json:"unionid"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

type WechatUserInfo struct {
	OpenId     string `json:"openid"`
	Nickname   string `json:"nickname"`
	Sex        int    `json:"sex"`
	Province   string `json:"province"`
	City       string `json:"city"`
	Country    string `json:"country"`
	HeadImgUrl string `json:"headimgurl"`
	UnionId    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

func WechatLogin(c *gin.Context) {
	if common.WechatAppId == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "微信 OAuth 未配置"})
		return
	}
	state := generateState()
	redirectURI := getWechatRedirectURI(c)
	u := fmt.Sprintf(
		"https://open.weixin.qq.com/connect/qrconnect?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_login&state=%s#wechat_redirect",
		common.WechatAppId, url.QueryEscape(redirectURI), state,
	)
	c.Redirect(http.StatusFound, u)
}

func WechatCallback(c *gin.Context) {
	if !verifyState(c.Query("state")) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Invalid state parameter"})
		return
	}
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Invalid code"})
		return
	}

	// Exchange code for access token
	tokenURL := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		common.WechatAppId, common.WechatAppSecret, code,
	)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(tokenURL)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to get access token: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("Token endpoint returned %d: %s", resp.StatusCode, string(body))})
		return
	}

	var tokenResp WechatAccessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if tokenResp.ErrCode != 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": tokenResp.ErrMsg})
		return
	}
	if tokenResp.OpenId == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to get OpenID"})
		return
	}

	// Get user info
	userInfoURL := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN",
		tokenResp.AccessToken, tokenResp.OpenId,
	)
	userResp, err := client.Get(userInfoURL)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to get user info: " + err.Error()})
		return
	}
	defer userResp.Body.Close()
	if userResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(userResp.Body)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("User info endpoint returned %d: %s", userResp.StatusCode, string(body))})
		return
	}

	var wechatUser WechatUserInfo
	if err := json.NewDecoder(userResp.Body).Decode(&wechatUser); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if wechatUser.ErrCode != 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": wechatUser.ErrMsg})
		return
	}

	// Use UnionId if available (cross-app identity), otherwise OpenId
	wechatId := wechatUser.UnionId
	if wechatId == "" {
		wechatId = wechatUser.OpenId
	}

	var maxID int64
	common.DB.Model(&model.User{}).Select("COALESCE(MAX(id), 0)").Scan(&maxID)
	user, pendingSessionID, err := createOAuthUser("wechat_id", wechatId, func() model.User {
		displayName := wechatUser.Nickname
		if displayName == "" {
			displayName = "微信用户"
		}
		return model.User{
			Username:    fmt.Sprintf("wx_%d", maxID+1),
			DisplayName: displayName,
			WechatId:    wechatId,
			Role:        model.RoleUser,
			Status:      model.UserStatusActive,
		}
	}, "wechat")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if pendingSessionID != "" {
		c.Redirect(http.StatusFound, fmt.Sprintf("/oauth/pending?oauth_pending=%s", pendingSessionID))
		return
	}
	oauthRedirect(c, user)
}

func getWechatRedirectURI(c *gin.Context) string {
	systemURL := common.OptionMap["system_url"]
	if systemURL != "" {
		return strings.TrimSuffix(systemURL, "/") + "/oauth/wechat/callback"
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/oauth/wechat/callback", scheme, c.Request.Host)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
