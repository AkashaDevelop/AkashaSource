package oauth

import (
	"STfreApi/common"
	"STfreApi/model"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

type DiscordOAuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type DiscordUser struct {
	Id            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	GlobalName    string `json:"global_name"`
	Avatar        string `json:"avatar"`
	Email         string `json:"email"`
}

func DiscordLogin(c *gin.Context) {
	if common.DiscordClientId == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Discord OAuth not configured"})
		return
	}
	redirectURI := getDiscordRedirectURI(c)
	u := fmt.Sprintf(
		"https://discord.com/api/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=identify%%20email",
		common.DiscordClientId, url.QueryEscape(redirectURI),
	)
	c.Redirect(http.StatusFound, u)
}

func DiscordCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Invalid code"})
		return
	}

	redirectURI := getDiscordRedirectURI(c)

	// Exchange code for token
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

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	defer resp.Body.Close()

	var oauthResp DiscordOAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&oauthResp); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if oauthResp.AccessToken == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to get access token"})
		return
	}

	// Get user info
	userReq, _ := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
	userReq.Header.Set("Authorization", "Bearer "+oauthResp.AccessToken)
	userResp, err := client.Do(userReq)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	defer userResp.Body.Close()

	var discordUser DiscordUser
	if err := json.NewDecoder(userResp.Body).Decode(&discordUser); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	// Find or create user
	var user model.User
	if err := common.DB.Where("discord_id = ?", discordUser.Id).First(&user).Error; err != nil {
		displayName := discordUser.GlobalName
		if displayName == "" {
			displayName = discordUser.Username
		}
		user = model.User{
			Username:    fmt.Sprintf("discord_%s", discordUser.Id),
			DisplayName: displayName,
			Email:       discordUser.Email,
			DiscordId:   discordUser.Id,
			Role:        model.RoleUser,
			Status:      model.UserStatusActive,
			Quota:       0,
		}
		if err := common.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to create user: " + err.Error()})
			return
		}
	} else {
		if user.Status == model.UserStatusBanned {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "User is banned"})
			return
		}
	}

	token, err := common.GenerateToken(user.Id, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to generate token"})
		return
	}
	c.Redirect(http.StatusFound, fmt.Sprintf("/?token=%s", token))
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
