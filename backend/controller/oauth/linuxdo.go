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

// LinuxDOLogin redirect user to LinuxDO login page
func LinuxDOLogin(c *gin.Context) {
	if common.LinuxDOClientId == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "LinuxDO OAuth not configured",
		})
		return
	}
	url := fmt.Sprintf("https://connect.linux.do/oauth2/authorize?client_id=%s&response_type=code", common.LinuxDOClientId)
	c.Redirect(http.StatusFound, url)
}

// LinuxDOCallback handle LinuxDO callback
func LinuxDOCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Invalid code",
		})
		return
	}

	// Exchange code for access token
	tokenUrl := "https://connect.linux.do/oauth2/token"

	req, err := http.NewRequest("POST", tokenUrl, nil)
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
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	defer resp.Body.Close()

	var oauthResp LinuxDOOAuthResponse
	err = json.NewDecoder(resp.Body).Decode(&oauthResp)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	if oauthResp.AccessToken == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to get access token"})
		return
	}

	// Get User Info
	userReq, _ := http.NewRequest("GET", "https://connect.linux.do/api/user", nil)
	userReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", oauthResp.AccessToken))
	var userResp *http.Response
	userResp, err = client.Do(userReq)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	defer userResp.Body.Close()

	var linuxDOUser LinuxDOUser
	err = json.NewDecoder(userResp.Body).Decode(&linuxDOUser)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	// Check if user exists
	var user model.User
	if err := common.DB.Where("linuxdo_id = ?", fmt.Sprintf("%d", linuxDOUser.Id)).First(&user).Error; err != nil {
		// Create new user
		user = model.User{
			Username:     fmt.Sprintf("linuxdo_%d", linuxDOUser.Id),
			DisplayName:  linuxDOUser.Name,
			LinuxDOId:    fmt.Sprintf("%d", linuxDOUser.Id),
			LinuxDOLevel: linuxDOUser.TrustLevel,
			Role:         model.RoleUser,
			Status:       model.UserStatusActive,
			Quota:        0, // Default quota
		}
		if user.DisplayName == "" {
			user.DisplayName = linuxDOUser.Username
		}

		// Initial Quota / Points logic
		if quota, ok := common.LinuxDOLevelQuota[linuxDOUser.TrustLevel]; ok {
			user.Quota = quota
		}

		if err := common.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to create user: " + err.Error()})
			return
		}
	} else {
		// Update LinuxDO Level
		if user.LinuxDOLevel != linuxDOUser.TrustLevel {
			user.LinuxDOLevel = linuxDOUser.TrustLevel
			common.DB.Model(&user).Update("linuxdo_level", linuxDOUser.TrustLevel)

			// Optional: Update Quota if upgraded?
			// For now, let's keep it simple: Quota is only given on registration
		}

		if user.Status == model.UserStatusBanned {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "User is banned"})
			return
		}
	}

	// Generate JWT
	token, err := common.GenerateToken(user.Id, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to generate token"})
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/?token=%s", token))
}
