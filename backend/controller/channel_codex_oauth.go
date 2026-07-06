package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	codexOAuthClientID     = "app_EMoamEEZ73f0CkXaXp7hrann"
	codexOAuthAuthorizeURL = "https://auth.openai.com/oauth/authorize"
	codexOAuthTokenURL     = "https://auth.openai.com/oauth/token"
	codexOAuthRedirectURI  = "http://localhost:1455/auth/callback"
	codexOAuthScope        = "openid profile email offline_access"
	codexJWTClaimPath      = "https://api.openai.com/auth"
)

type codexOAuthFlow struct {
	State     string
	Verifier  string
	CreatedAt int64
}

var codexFlowStore = struct {
	sync.RWMutex
	m map[int]codexOAuthFlow
}{m: map[int]codexOAuthFlow{}}

func storeCodexFlow(channelID int, state string, verifier string) {
	codexFlowStore.Lock()
	defer codexFlowStore.Unlock()
	codexFlowStore.m[channelID] = codexOAuthFlow{State: state, Verifier: verifier, CreatedAt: time.Now().Unix()}
}

func loadCodexFlow(channelID int) (codexOAuthFlow, bool) {
	codexFlowStore.RLock()
	defer codexFlowStore.RUnlock()
	f, ok := codexFlowStore.m[channelID]
	if !ok {
		return codexOAuthFlow{}, false
	}
	if time.Now().Unix()-f.CreatedAt > 15*60 {
		return codexOAuthFlow{}, false
	}
	return f, true
}

func deleteCodexFlow(channelID int) {
	codexFlowStore.Lock()
	defer codexFlowStore.Unlock()
	delete(codexFlowStore.m, channelID)
}

func createPKCEPair() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	verifier := base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func buildCodexAuthorizeURL(state string, challenge string) (string, error) {
	u, err := url.Parse(codexOAuthAuthorizeURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", codexOAuthClientID)
	q.Set("redirect_uri", codexOAuthRedirectURI)
	q.Set("scope", codexOAuthScope)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", state)
	q.Set("id_token_add_organizations", "true")
	q.Set("codex_cli_simplified_flow", "true")
	q.Set("originator", "codex_cli_rs")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func parseCodexAuthorizationInput(input string) (string, string, error) {
	v := strings.TrimSpace(input)
	if v == "" {
		return "", "", fmt.Errorf("empty input")
	}
	if strings.Contains(v, "#") {
		parts := strings.SplitN(v, "#", 2)
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
	}
	if strings.Contains(v, "code=") {
		u, err := url.Parse(v)
		if err == nil {
			q := u.Query()
			return strings.TrimSpace(q.Get("code")), strings.TrimSpace(q.Get("state")), nil
		}
		q, err := url.ParseQuery(v)
		if err == nil {
			return strings.TrimSpace(q.Get("code")), strings.TrimSpace(q.Get("state")), nil
		}
	}
	return v, "", nil
}

func exchangeCodexAuthorizationCode(code string, verifier string, proxy string) (map[string]interface{}, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", codexOAuthClientID)
	form.Set("code", strings.TrimSpace(code))
	form.Set("code_verifier", strings.TrimSpace(verifier))
	form.Set("redirect_uri", codexOAuthRedirectURI)

	req, err := http.NewRequest(http.MethodPost, codexOAuthTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := common.NewHTTPClient(strings.TrimSpace(proxy))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("oauth exchange failed: %d", resp.StatusCode)
	}
	return body, nil
}

func decodeJWTClaims(token string) (map[string]interface{}, bool) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return nil, false
	}
	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, false
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payloadRaw, &claims); err != nil {
		return nil, false
	}
	return claims, true
}

func extractCodexAccountIDFromJWT(token string) (string, bool) {
	claims, ok := decodeJWTClaims(token)
	if !ok {
		return "", false
	}
	raw, ok := claims[codexJWTClaimPath]
	if !ok {
		return "", false
	}
	obj, ok := raw.(map[string]interface{})
	if !ok {
		return "", false
	}
	v, ok := obj["chatgpt_account_id"]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return s, true
}

func extractEmailFromJWT(token string) string {
	claims, ok := decodeJWTClaims(token)
	if !ok {
		return ""
	}
	v, ok := claims["email"].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(v)
}

func startCodexOAuthWithChannelID(c *gin.Context, channelID int) {
	if channelID > 0 {
		var ch model.Channel
		if err := common.DB.First(&ch, channelID).Error; err != nil {
			common.Fail(c, common.CodeNotFound, "渠道不存在")
			return
		}
		if ch.Type != model.ChannelTypeCodex {
			common.Fail(c, common.CodeParamError, "channel type is not Codex")
			return
		}
	}
	state := common.GetUUID()
	verifier, challenge, err := createPKCEPair()
	if err != nil {
		common.Fail(c, common.CodeServerError, "生成 OAuth 参数失败")
		return
	}
	authorizeURL, err := buildCodexAuthorizeURL(state, challenge)
	if err != nil {
		common.Fail(c, common.CodeServerError, "生成授权地址失败")
		return
	}
	storeCodexFlow(channelID, state, verifier)
	common.OK(c, gin.H{"success": true, "message": "", "data": gin.H{"authorize_url": authorizeURL}})
}

func StartCodexOAuth(c *gin.Context) {
	startCodexOAuthWithChannelID(c, 0)
}

func StartCodexOAuthForChannel(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "渠道 ID 非法")
		return
	}
	startCodexOAuthWithChannelID(c, channelID)
}

func completeCodexOAuthWithChannelID(c *gin.Context, channelID int) {
	var req struct {
		Input string `json:"input"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	code, state, err := parseCodexAuthorizationInput(req.Input)
	if err != nil {
		common.Fail(c, common.CodeParamError, "解析授权信息失败")
		return
	}
	if strings.TrimSpace(code) == "" {
		common.Fail(c, common.CodeParamError, "missing authorization code")
		return
	}
	if strings.TrimSpace(state) == "" {
		common.Fail(c, common.CodeParamError, "missing state in input")
		return
	}

	flow, ok := loadCodexFlow(channelID)
	if !ok {
		common.Fail(c, common.CodeParamError, "oauth flow not started or session expired")
		return
	}
	if state != flow.State {
		common.Fail(c, common.CodeParamError, "state mismatch")
		return
	}

	channelProxy := ""
	if channelID > 0 {
		var ch model.Channel
		if err := common.DB.First(&ch, channelID).Error; err != nil {
			common.Fail(c, common.CodeNotFound, "渠道不存在")
			return
		}
		if ch.Type != model.ChannelTypeCodex {
			common.Fail(c, common.CodeParamError, "channel type is not Codex")
			return
		}
		channelProxy = ch.Proxy
	}

	tokenRes, err := exchangeCodexAuthorizationCode(code, flow.Verifier, channelProxy)
	if err != nil {
		common.Fail(c, common.CodeServerError, "授权码交换失败，请重试")
		return
	}

	accessToken, _ := tokenRes["access_token"].(string)
	refreshToken, _ := tokenRes["refresh_token"].(string)
	if strings.TrimSpace(accessToken) == "" || strings.TrimSpace(refreshToken) == "" {
		common.Fail(c, common.CodeServerError, "codex oauth token response missing fields")
		return
	}

	accountID, ok := extractCodexAccountIDFromJWT(accessToken)
	if !ok {
		common.Fail(c, common.CodeServerError, "failed to extract account_id from access_token")
		return
	}
	email := extractEmailFromJWT(accessToken)

	expiresAt := ""
	if expiresIn, ok := tokenRes["expires_in"].(float64); ok && expiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(int64(expiresIn)) * time.Second).Format(time.RFC3339)
	}

	keyObj := map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"account_id":    accountID,
		"last_refresh":  time.Now().Format(time.RFC3339),
		"expired":       expiresAt,
		"email":         email,
		"type":          "codex",
	}
	encoded, _ := json.Marshal(keyObj)
	deleteCodexFlow(channelID)

	if channelID > 0 {
		if err := common.DB.Model(&model.Channel{}).Where("id = ?", channelID).Update("key", string(encoded)).Error; err != nil {
			common.Fail(c, common.CodeServerError, "保存 Codex 凭据失败")
			return
		}
		common.OK(c, gin.H{"success": true, "message": "saved", "data": gin.H{"channel_id": channelID, "account_id": accountID, "email": email, "expires_at": expiresAt}})
		return
	}
	common.OK(c, gin.H{"success": true, "message": "generated", "data": gin.H{"key": string(encoded), "account_id": accountID, "email": email, "expires_at": expiresAt}})
}

func CompleteCodexOAuth(c *gin.Context) {
	completeCodexOAuthWithChannelID(c, 0)
}

func CompleteCodexOAuthForChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "渠道 ID 非法")
		return
	}
	completeCodexOAuthWithChannelID(c, id)
}

func GetCodexChannelUsage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "渠道 ID 非法")
		return
	}
	var ch model.Channel
	if err = common.DB.First(&ch, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}
	common.OK(c, gin.H{
		"channel_id":   id,
		"used_quota":   ch.UsedQuota,
		"balance":      ch.Balance,
		"last_test_at": ch.TestTime,
	})
}

func RefreshCodexChannelCredential(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "渠道 ID 非法")
		return
	}
	var ch model.Channel
	if err = common.DB.First(&ch, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}
	if ch.Type != model.ChannelTypeCodex {
		common.Fail(c, common.CodeParamError, "channel type is not Codex")
		return
	}
	var keyMap map[string]interface{}
	if err = json.Unmarshal([]byte(ch.Key), &keyMap); err != nil {
		common.Fail(c, common.CodeParamError, "渠道 key 不是 Codex OAuth 格式")
		return
	}
	refreshToken, _ := keyMap["refresh_token"].(string)
	if strings.TrimSpace(refreshToken) == "" {
		common.Fail(c, common.CodeParamError, "refresh_token 缺失")
		return
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", codexOAuthClientID)
	req, err := http.NewRequest(http.MethodPost, codexOAuthTokenURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		common.Fail(c, common.CodeServerError, "刷新凭据失败")
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := common.NewHTTPClient(ch.Proxy).Do(req)
	if err != nil {
		common.Fail(c, common.CodeServerError, "刷新凭据失败")
		return
	}
	defer resp.Body.Close()
	var tokenResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		common.Fail(c, common.CodeServerError, "刷新凭据失败")
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		common.Fail(c, common.CodeServerError, "刷新凭据失败")
		return
	}
	at, _ := tokenResp["access_token"].(string)
	rt, _ := tokenResp["refresh_token"].(string)
	if strings.TrimSpace(at) == "" {
		common.Fail(c, common.CodeServerError, "刷新凭据失败")
		return
	}
	if strings.TrimSpace(rt) == "" {
		rt = refreshToken
	}
	keyMap["access_token"] = at
	keyMap["refresh_token"] = rt
	keyMap["last_refresh"] = time.Now().Format(time.RFC3339)
	if expiresIn, ok := tokenResp["expires_in"].(float64); ok && expiresIn > 0 {
		keyMap["expired"] = time.Now().Add(time.Duration(int64(expiresIn)) * time.Second).Format(time.RFC3339)
	}
	encoded, _ := json.Marshal(keyMap)
	if err = common.DB.Model(&model.Channel{}).Where("id = ?", id).Update("key", string(encoded)).Error; err != nil {
		common.Fail(c, common.CodeServerError, "刷新凭据失败")
		return
	}
	common.OK(c, gin.H{"success": true, "message": "", "data": gin.H{"channel_id": id}})
}
