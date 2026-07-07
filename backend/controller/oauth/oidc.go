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

type OIDCDiscovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
}

type OIDCTokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
}

type OIDCUserInfo struct {
	Sub              string `json:"sub"`
	Name             string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	Email            string `json:"email"`
}

var cachedDiscovery *OIDCDiscovery

func fetchOIDCDiscovery() (*OIDCDiscovery, error) {
	if cachedDiscovery != nil {
		return cachedDiscovery, nil
	}
	issuer := strings.TrimSuffix(common.OIDCIssuerURL, "/")
	resp, err := http.Get(issuer + "/.well-known/openid-configuration")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var disc OIDCDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&disc); err != nil {
		return nil, err
	}
	cachedDiscovery = &disc
	return cachedDiscovery, nil
}

func OIDCLogin(c *gin.Context) {
	if common.OIDCClientId == "" || common.OIDCIssuerURL == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "OIDC not configured"})
		return
	}
	disc, err := fetchOIDCDiscovery()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to fetch OIDC discovery: " + err.Error()})
		return
	}
	state := generateState()
	redirectURI := getOIDCRedirectURI(c)
	u := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=openid+profile+email&state=%s",
		disc.AuthorizationEndpoint, common.OIDCClientId, url.QueryEscape(redirectURI), state,
	)
	c.Redirect(http.StatusFound, u)
}

func OIDCCallback(c *gin.Context) {
	if !verifyState(c.Query("state")) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Invalid state parameter"})
		return
	}
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Invalid code"})
		return
	}
	disc, err := fetchOIDCDiscovery()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "OIDC discovery failed"})
		return
	}
	redirectURI := getOIDCRedirectURI(c)

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", common.OIDCClientId)
	data.Set("client_secret", common.OIDCClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", disc.TokenEndpoint, strings.NewReader(data.Encode()))
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

	var tokenResp OIDCTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if tokenResp.AccessToken == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to get access token"})
		return
	}

	userReq, _ := http.NewRequest("GET", disc.UserinfoEndpoint, nil)
	userReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
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

	var userInfo OIDCUserInfo
	if err := json.NewDecoder(userResp.Body).Decode(&userInfo); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if userInfo.Sub == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to get user info"})
		return
	}

	user, pendingSessionID, err := createOAuthUser("oidc_id", userInfo.Sub, func() model.User {
		displayName := userInfo.Name
		if displayName == "" {
			displayName = userInfo.PreferredUsername
		}
		return model.User{
			Username:    fmt.Sprintf("oidc_%s", userInfo.Sub),
			DisplayName: displayName,
			Email:       userInfo.Email,
			OIDCId:      userInfo.Sub,
			Role:        model.RoleUser,
			Status:      model.UserStatusActive,
		}
	}, "oidc")
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

func getOIDCRedirectURI(c *gin.Context) string {
	systemURL := common.OptionMap["system_url"]
	if systemURL != "" {
		return strings.TrimSuffix(systemURL, "/") + "/oauth/oidc/callback"
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/oauth/oidc/callback", scheme, c.Request.Host)
}
