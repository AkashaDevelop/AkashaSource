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

type DiscordOAuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type DiscordUser struct {
	Id         string `json:"id"`
	Username   string `json:"username"`
	GlobalName string `json:"global_name"`
	Email      string `json:"email"`
}

func DiscordLogin(c *gin.Context) {
	if common.DiscordClientId == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Discord OAuth not configured"})
		return
	}
	state := generateState()
	redirectURI := getDiscordRedirectURI(c)
	u := fmt.Sprintf(
		"https://discord.com/api/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=identify%%20email&state=%s",
		common.DiscordClientId, url.QueryEscape(redirectURI), state,
	)
	c.Redirect(http.StatusFound, u)
}

func DiscordCallback(c *gin.Context) {
	if !verifyState(c.Query("state")) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Invalid state parameter"})
		return
	}
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Invalid code"})
		return
	}

	redirectURI := getDiscordRedirectURI(c)
	data := url.Values{}
	data.Set("client_id", common.DiscordClientId)
	data.Set("client_secret", common.DiscordClientSecret)
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", "https://discord.com/api/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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

	var oauthResp DiscordOAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&oauthResp); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if oauthResp.AccessToken == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to get access token"})
		return
	}

	userReq, _ := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
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

	var discordUser DiscordUser
	if err := json.NewDecoder(userResp.Body).Decode(&discordUser); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	user, pendingSessionID, err := createOAuthUser("discord_id", discordUser.Id, func() model.User {
		displayName := discordUser.GlobalName
		if displayName == "" {
			displayName = discordUser.Username
		}
		return model.User{
			Username:    fmt.Sprintf("discord_%s", discordUser.Id),
			DisplayName: displayName,
			Email:       discordUser.Email,
			DiscordId:   discordUser.Id,
			Role:        model.RoleUser,
			Status:      model.UserStatusActive,
		}
	}, "discord")
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

func getDiscordRedirectURI(c *gin.Context) string {
	systemURL := common.OptionMap["system_url"]
	if systemURL != "" {
		return strings.TrimSuffix(systemURL, "/") + "/oauth/discord/callback"
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/oauth/discord/callback", scheme, c.Request.Host)
}
