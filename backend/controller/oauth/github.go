package oauth

import (
	"STfreApi/common"
	"STfreApi/model"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

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

func GitHubLogin(c *gin.Context) {
	if common.GitHubClientId == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "GitHub OAuth not configured"})
		return
	}
	state := generateState()
	url := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&scope=user:email&state=%s",
		common.GitHubClientId, state,
	)
	c.Redirect(http.StatusFound, url)
}

func GitHubCallback(c *gin.Context) {
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
	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", nil)
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

	var oauthResp GitHubOAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&oauthResp); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if oauthResp.AccessToken == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to get access token"})
		return
	}

	// Get user info
	userReq, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
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

	var githubUser GitHubUser
	if err := json.NewDecoder(userResp.Body).Decode(&githubUser); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	idValue := fmt.Sprintf("%d", githubUser.Id)
	user, pendingSessionID, err := createOAuthUser("github_id", idValue, func() model.User {
		displayName := githubUser.Name
		if displayName == "" {
			displayName = githubUser.Login
		}
		return model.User{
			Username:    fmt.Sprintf("github_%d", githubUser.Id),
			DisplayName: displayName,
			Email:       githubUser.Email,
			GithubId:    idValue,
			Role:        model.RoleUser,
			Status:      model.UserStatusActive,
		}
	}, "github")
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
