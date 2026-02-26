package oauth

import (
	"STfreApi/common"
	"STfreApi/model"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type GitHubOAuthResponse struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

type GitHubUser struct {
	Login     string `json:"login"`
	Id        int    `json:"id"`
	AvatarUrl string `json:"avatar_url"`
	Name      string `json:"name"`
	Email     string `json:"email"`
}

// GitHubLogin redirect user to GitHub login page
func GitHubLogin(c *gin.Context) {
	if common.GitHubClientId == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "GitHub OAuth not configured",
		})
		return
	}
	// TODO: Add state parameter for security
	// NewAPI uses "https://github.com/login/oauth/authorize?client_id=%s&scope=user:email&state=%s"
	url := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&scope=user:email", common.GitHubClientId)
	c.Redirect(http.StatusFound, url)
}

// GitHubCallback handle GitHub callback
func GitHubCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Invalid code",
		})
		return
	}

	// Exchange code for access token
	tokenUrl := "https://github.com/login/oauth/access_token"

	req, err := http.NewRequest("POST", tokenUrl, nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	q := req.URL.Query()
	q.Add("client_id", common.GitHubClientId)
	q.Add("client_secret", common.GitHubClientSecret)
	q.Add("code", code)
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

	var oauthResp GitHubOAuthResponse
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
	userReq, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	userReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", oauthResp.AccessToken))
	var userResp *http.Response
	userResp, err = client.Do(userReq)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	defer userResp.Body.Close()

	var githubUser GitHubUser
	err = json.NewDecoder(userResp.Body).Decode(&githubUser)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	// Check if user exists
	var user model.User
	if err := common.DB.Where("github_id = ?", fmt.Sprintf("%d", githubUser.Id)).First(&user).Error; err != nil {
		// User not found, check if register is enabled
		// For now assume register is enabled

		// Create new user
		user = model.User{
			Username:    fmt.Sprintf("github_%d", githubUser.Id),
			DisplayName: githubUser.Name,
			Email:       githubUser.Email,
			GithubId:    fmt.Sprintf("%d", githubUser.Id),
			Role:        model.RoleUser,
			Status:      model.UserStatusActive,
			Quota:       0, // Default quota
		}
		if user.DisplayName == "" {
			user.DisplayName = githubUser.Login
		}

		// Check if username/email conflicts?
		// Username is unique, so github_ID should be unique.

		if err := common.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to create user: " + err.Error()})
			return
		}
	} else {
		// User exists, check if banned
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

	// Redirect to frontend with token
	// Assuming frontend is running on same domain or we know the URL
	// For dev: localhost:3000?token=xxx
	// For prod: /?token=xxx

	// We can set a cookie or redirect with query param
	c.Redirect(http.StatusFound, fmt.Sprintf("/?token=%s", token))
}
