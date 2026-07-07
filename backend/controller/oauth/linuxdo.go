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

type LinuxDOOAuthResponse struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

type LinuxDOUser struct {
	Id         int    `json:"id"`
	Username   string `json:"username"`
	Name       string `json:"name"`
	AvatarUrl  string `json:"avatar_url"`
	TrustLevel int    `json:"trust_level"`
	Active     bool   `json:"active"`
}

func LinuxDOLogin(c *gin.Context) {
	if common.LinuxDOClientId == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "LinuxDO OAuth not configured"})
		return
	}
	state := generateState()
	redirectURI := getLinuxDORedirectURI(c)
	authURL := fmt.Sprintf(
		"https://connect.linux.do/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&state=%s",
		common.LinuxDOClientId, url.QueryEscape(redirectURI), state,
	)
	c.Redirect(http.StatusFound, authURL)
}

func LinuxDOCallback(c *gin.Context) {
	if !verifyState(c.Query("state")) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Invalid state parameter"})
		return
	}
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Invalid code"})
		return
	}

	redirectURI := getLinuxDORedirectURI(c)
	data := url.Values{}
	data.Set("client_id", common.LinuxDOClientId)
	data.Set("client_secret", common.LinuxDOClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", "https://connect.linux.do/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
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

	var oauthResp LinuxDOOAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&oauthResp); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if oauthResp.AccessToken == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to get access token"})
		return
	}

	userReq, _ := http.NewRequest("GET", "https://connect.linux.do/api/user", nil)
	userReq.Header.Set("Authorization", "Bearer "+oauthResp.AccessToken)
	userResp, err := client.Do(userReq)
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

	var linuxDOUser LinuxDOUser
	if err := json.NewDecoder(userResp.Body).Decode(&linuxDOUser); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if common.LinuxDOMinTrustLevel > 0 && linuxDOUser.TrustLevel < common.LinuxDOMinTrustLevel {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": fmt.Sprintf("LinuxDO 信任等级不足，需要至少 %d 级", common.LinuxDOMinTrustLevel)})
		return
	}

	idValue := fmt.Sprintf("%d", linuxDOUser.Id)
	user, pendingSessionID, err := createOAuthUser("linuxdo_id", idValue, func() model.User {
		displayName := linuxDOUser.Name
		if displayName == "" {
			displayName = linuxDOUser.Username
		}
		// LinuxDO level quota takes priority over generic new_user_reward
		var quota int64
		if q, ok := common.LinuxDOLevelQuota[linuxDOUser.TrustLevel]; ok && q > 0 {
			quota = q
		}
		return model.User{
			Username:     fmt.Sprintf("linuxdo_%d", linuxDOUser.Id),
			DisplayName:  displayName,
			LinuxDOId:    idValue,
			LinuxDOLevel: linuxDOUser.TrustLevel,
			Role:         model.RoleUser,
			Status:       model.UserStatusActive,
			Quota:        quota,
		}
	}, "linuxdo")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if pendingSessionID != "" {
		c.Redirect(http.StatusFound, fmt.Sprintf("/oauth/pending?oauth_pending=%s", pendingSessionID))
		return
	}

	// Sync trust level on subsequent logins
	if user.LinuxDOLevel != linuxDOUser.TrustLevel {
		common.DB.Model(user).Update("linuxdo_level", linuxDOUser.TrustLevel)
	}

	oauthRedirect(c, user)
}

func getLinuxDORedirectURI(c *gin.Context) string {
	systemURL := common.OptionMap["system_url"]
	if systemURL != "" {
		return strings.TrimSuffix(systemURL, "/") + "/oauth/linuxdo/callback"
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/oauth/linuxdo/callback", scheme, c.Request.Host)
}
