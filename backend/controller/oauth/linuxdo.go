package oauth

import (
	"STfreApi/common"
	"STfreApi/model"
	"encoding/json"
	"fmt"
	"net/http"

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
	url := fmt.Sprintf(
		"https://connect.linux.do/oauth2/authorize?client_id=%s&response_type=code&state=%s",
		common.LinuxDOClientId, state,
	)
	c.Redirect(http.StatusFound, url)
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

	req, err := http.NewRequest("POST", "https://connect.linux.do/oauth2/token", nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	q := req.URL.Query()
	q.Add("client_id", common.LinuxDOClientId)
	q.Add("client_secret", common.LinuxDOClientSecret)
	q.Add("code", code)
	q.Add("grant_type", "authorization_code")
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	defer resp.Body.Close()

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
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	defer userResp.Body.Close()

	var linuxDOUser LinuxDOUser
	if err := json.NewDecoder(userResp.Body).Decode(&linuxDOUser); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	idValue := fmt.Sprintf("%d", linuxDOUser.Id)
	user, err := createOAuthUser("linuxdo_id", idValue, func() model.User {
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
	}, c.Query("aff"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	// Sync trust level on subsequent logins
	if user.LinuxDOLevel != linuxDOUser.TrustLevel {
		common.DB.Model(user).Update("linuxdo_level", linuxDOUser.TrustLevel)
	}

	oauthRedirect(c, user)
}
