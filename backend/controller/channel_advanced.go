package controller

import (
	ollamaAdapter "STfreApi/adapter/ollama"
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

var multiKeyLockStore = struct {
	sync.Mutex
	m map[int]*sync.Mutex
}{m: map[int]*sync.Mutex{}}

func getMultiKeyLock(channelID int) *sync.Mutex {
	multiKeyLockStore.Lock()
	defer multiKeyLockStore.Unlock()
	if l, ok := multiKeyLockStore.m[channelID]; ok {
		return l
	}
	l := &sync.Mutex{}
	multiKeyLockStore.m[channelID] = l
	return l
}

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

func splitLinesNonEmpty(s string) []string {
	items := make([]string, 0)
	for _, k := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		k = strings.TrimSpace(k)
		if k != "" {
			items = append(items, k)
		}
	}
	return items
}

func parseIntMap(s string) map[int]int {
	if strings.TrimSpace(s) == "" {
		return map[int]int{}
	}
	raw := map[string]int{}
	if json.Unmarshal([]byte(s), &raw) != nil {
		return map[int]int{}
	}
	ret := map[int]int{}
	for k, v := range raw {
		if idx, err := strconv.Atoi(k); err == nil {
			ret[idx] = v
		}
	}
	return ret
}

func parseInt64Map(s string) map[int]int64 {
	if strings.TrimSpace(s) == "" {
		return map[int]int64{}
	}
	raw := map[string]int64{}
	if json.Unmarshal([]byte(s), &raw) != nil {
		return map[int]int64{}
	}
	ret := map[int]int64{}
	for k, v := range raw {
		if idx, err := strconv.Atoi(k); err == nil {
			ret[idx] = v
		}
	}
	return ret
}

func parseStringMap(s string) map[int]string {
	if strings.TrimSpace(s) == "" {
		return map[int]string{}
	}
	raw := map[string]string{}
	if json.Unmarshal([]byte(s), &raw) != nil {
		return map[int]string{}
	}
	ret := map[int]string{}
	for k, v := range raw {
		if idx, err := strconv.Atoi(k); err == nil {
			ret[idx] = v
		}
	}
	return ret
}

func marshalIntMap(m map[int]int) string {
	raw := map[string]int{}
	for k, v := range m {
		raw[strconv.Itoa(k)] = v
	}
	b, _ := json.Marshal(raw)
	return string(b)
}

func marshalInt64Map(m map[int]int64) string {
	raw := map[string]int64{}
	for k, v := range m {
		raw[strconv.Itoa(k)] = v
	}
	b, _ := json.Marshal(raw)
	return string(b)
}

func marshalStringMap(m map[int]string) string {
	raw := map[string]string{}
	for k, v := range m {
		raw[strconv.Itoa(k)] = v
	}
	b, _ := json.Marshal(raw)
	return string(b)
}

func BatchSetChannelTag(c *gin.Context) {
	var req struct {
		Ids []int  `json:"ids"`
		Tag string `json:"tag"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if len(req.Ids) == 0 || strings.TrimSpace(req.Tag) == "" {
		common.Fail(c, common.CodeParamError, "ids 和 tag 不能为空")
		return
	}
	if err := common.DB.Model(&model.Channel{}).Where("id IN ?", req.Ids).Update("tags", req.Tag).Error; err != nil {
		common.Fail(c, common.CodeServerError, "批量设置标签失败")
		return
	}
	common.OK(c, gin.H{"updated": len(req.Ids)})
}

func GetTagModels(c *gin.Context) {
	tag := strings.TrimSpace(c.Query("tag"))
	if tag == "" {
		common.Fail(c, common.CodeParamError, "tag 不能为空")
		return
	}
	var channels []model.Channel
	if err := common.DB.Where("tags LIKE ?", "%"+tag+"%").Find(&channels).Error; err != nil {
		common.Fail(c, common.CodeServerError, "查询失败")
		return
	}
	longestModels := ""
	maxLen := 0
	for _, ch := range channels {
		if strings.TrimSpace(ch.Models) == "" {
			continue
		}
		current := strings.Split(ch.Models, ",")
		if len(current) > maxLen {
			maxLen = len(current)
			longestModels = ch.Models
		}
	}
	common.OK(c, gin.H{"success": true, "message": "", "data": longestModels})
}

func CopyChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "渠道 ID 非法")
		return
	}
	var src model.Channel
	if err = common.DB.First(&src, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}
	clone := src
	clone.Id = 0
	suffix := c.DefaultQuery("suffix", "_复制")
	resetBalance := true
	if rbStr := c.DefaultQuery("reset_balance", "true"); rbStr != "" {
		if v, e := strconv.ParseBool(rbStr); e == nil {
			resetBalance = v
		}
	}
	clone.Name = src.Name + suffix
	clone.TestTime = 0
	clone.ResponseTime = 0
	if resetBalance {
		clone.Balance = 0
		clone.UsedQuota = 0
	}
	if err = common.DB.Create(&clone).Error; err != nil {
		common.Fail(c, common.CodeServerError, "复制渠道失败")
		return
	}
	common.OK(c, gin.H{"success": true, "message": "", "data": gin.H{"id": clone.Id}})
}

type MultiKeyManageRequest struct {
	ChannelID int      `json:"channel_id"`
	Action    string   `json:"action"`
	KeyIndex  *int     `json:"key_index,omitempty"`
	Page      int      `json:"page,omitempty"`
	PageSize  int      `json:"page_size,omitempty"`
	Status    *int     `json:"status,omitempty"`
	Keys      []string `json:"keys,omitempty"`
}

type KeyStatus struct {
	Index        int    `json:"index"`
	Status       int    `json:"status"`
	DisabledTime int64  `json:"disabled_time,omitempty"`
	Reason       string `json:"reason,omitempty"`
	KeyPreview   string `json:"key_preview"`
}

func ManageMultiKeys(c *gin.Context) {
	var req MultiKeyManageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	req.Action = strings.TrimSpace(req.Action)
	if req.ChannelID == 0 {
		common.Fail(c, common.CodeParamError, "channel_id 不能为空")
		return
	}
	if req.Action == "" {
		common.Fail(c, common.CodeParamError, "action 不能为空")
		return
	}

	var ch model.Channel
	if err := common.DB.First(&ch, req.ChannelID).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}

	lock := getMultiKeyLock(ch.Id)
	lock.Lock()
	defer lock.Unlock()

	keys := splitLinesNonEmpty(ch.Key)
	statusMap := parseIntMap(ch.MultiKeyStatus)
	disabledTimeMap := parseInt64Map(ch.MultiKeyDisabledTime)
	disabledReasonMap := parseStringMap(ch.MultiKeyDisabledReason)

	saveState := func() error {
		updates := map[string]interface{}{
			"key":                       strings.Join(keys, "\n"),
			"multi_key_status":          marshalIntMap(statusMap),
			"multi_key_disabled_time":   marshalInt64Map(disabledTimeMap),
			"multi_key_disabled_reason": marshalStringMap(disabledReasonMap),
		}
		return common.DB.Model(&model.Channel{}).Where("id = ?", ch.Id).Updates(updates).Error
	}

	switch req.Action {
	case "set":
		keys = make([]string, 0, len(req.Keys))
		for _, k := range req.Keys {
			k = strings.TrimSpace(k)
			if k != "" {
				keys = append(keys, k)
			}
		}
		if len(keys) == 0 {
			common.Fail(c, common.CodeParamError, "keys 不能为空")
			return
		}
		statusMap = map[int]int{}
		disabledTimeMap = map[int]int64{}
		disabledReasonMap = map[int]string{}
		if err := saveState(); err != nil {
			common.Fail(c, common.CodeServerError, "更新多 Key 失败")
			return
		}
		common.OK(c, gin.H{"success": true, "message": "", "data": gin.H{"count": len(keys)}})
		return

	case "get_key_status":
		page := req.Page
		pageSize := req.PageSize
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 50
		}
		if pageSize > 200 {
			pageSize = 200
		}

		all := make([]KeyStatus, 0, len(keys))
		enabledCount, manualDisabledCount, autoDisabledCount := 0, 0, 0
		for i, k := range keys {
			st := 1
			if v, ok := statusMap[i]; ok {
				st = v
			}
			switch st {
			case 1:
				enabledCount++
			case 2:
				manualDisabledCount++
			case 3:
				autoDisabledCount++
			}
			preview := k
			if len(preview) > 10 {
				preview = preview[:10] + "..."
			}
			item := KeyStatus{Index: i, Status: st, KeyPreview: preview}
			if st != 1 {
				if t, ok := disabledTimeMap[i]; ok {
					item.DisabledTime = t
				}
				if r, ok := disabledReasonMap[i]; ok {
					item.Reason = r
				}
			}
			all = append(all, item)
		}

		filtered := all
		if req.Status != nil {
			filtered = make([]KeyStatus, 0)
			for _, item := range all {
				if item.Status == *req.Status {
					filtered = append(filtered, item)
				}
			}
		}

		total := len(filtered)
		totalPages := (total + pageSize - 1) / pageSize
		if totalPages == 0 {
			totalPages = 1
		}
		if page > totalPages {
			page = totalPages
		}
		start := (page - 1) * pageSize
		end := start + pageSize
		if end > total {
			end = total
		}
		pageData := make([]KeyStatus, 0)
		if start < total {
			pageData = filtered[start:end]
		}
		common.OK(c, gin.H{"success": true, "message": "", "data": gin.H{
			"keys":  pageData,
			"total": total, "page": page, "page_size": pageSize, "total_pages": totalPages,
			"enabled_count": enabledCount, "manual_disabled_count": manualDisabledCount, "auto_disabled_count": autoDisabledCount,
		}})
		return

	case "disable_key", "enable_key", "delete_key":
		if req.KeyIndex == nil {
			common.Fail(c, common.CodeParamError, "未指定要操作的密钥索引")
			return
		}
		idx := *req.KeyIndex
		if idx < 0 || idx >= len(keys) {
			common.Fail(c, common.CodeParamError, "密钥索引超出范围")
			return
		}

		if req.Action == "disable_key" {
			statusMap[idx] = 2
			disabledTimeMap[idx] = time.Now().Unix()
			disabledReasonMap[idx] = "manual"
		} else if req.Action == "enable_key" {
			delete(statusMap, idx)
			delete(disabledTimeMap, idx)
			delete(disabledReasonMap, idx)
		} else {
			if len(keys) <= 1 {
				common.Fail(c, common.CodeParamError, "不能删除最后一个密钥")
				return
			}
			nextKeys := make([]string, 0, len(keys)-1)
			nextStatus := map[int]int{}
			nextTime := map[int]int64{}
			nextReason := map[int]string{}
			ni := 0
			for i, key := range keys {
				if i == idx {
					continue
				}
				nextKeys = append(nextKeys, key)
				if st, ok := statusMap[i]; ok && st != 1 {
					nextStatus[ni] = st
				}
				if t, ok := disabledTimeMap[i]; ok {
					nextTime[ni] = t
				}
				if r, ok := disabledReasonMap[i]; ok {
					nextReason[ni] = r
				}
				ni++
			}
			keys = nextKeys
			statusMap = nextStatus
			disabledTimeMap = nextTime
			disabledReasonMap = nextReason
		}
		if err := saveState(); err != nil {
			common.Fail(c, common.CodeServerError, "更新多 Key 失败")
			return
		}
		common.OK(c, gin.H{"success": true, "message": ""})
		return

	case "enable_all_keys":
		statusMap = map[int]int{}
		disabledTimeMap = map[int]int64{}
		disabledReasonMap = map[int]string{}
		if err := saveState(); err != nil {
			common.Fail(c, common.CodeServerError, "更新多 Key 失败")
			return
		}
		common.OK(c, gin.H{"success": true, "message": ""})
		return

	case "disable_all_keys":
		now := time.Now().Unix()
		disabledCount := 0
		for i := range keys {
			st := 1
			if v, ok := statusMap[i]; ok {
				st = v
			}
			if st == 1 {
				statusMap[i] = 2
				disabledTimeMap[i] = now
				disabledReasonMap[i] = "manual"
				disabledCount++
			}
		}
		if disabledCount == 0 {
			common.Fail(c, common.CodeParamError, "没有可禁用的密钥")
			return
		}
		if err := saveState(); err != nil {
			common.Fail(c, common.CodeServerError, "更新多 Key 失败")
			return
		}
		common.OK(c, gin.H{"success": true, "message": ""})
		return

	case "delete_disabled_keys":
		nextKeys := make([]string, 0, len(keys))
		nextStatus := map[int]int{}
		nextTime := map[int]int64{}
		nextReason := map[int]string{}
		deleted := 0
		ni := 0
		for i, k := range keys {
			st := 1
			if v, ok := statusMap[i]; ok {
				st = v
			}
			if st == 3 {
				deleted++
				continue
			}
			nextKeys = append(nextKeys, k)
			if st != 1 {
				nextStatus[ni] = st
				if t, ok := disabledTimeMap[i]; ok {
					nextTime[ni] = t
				}
				if r, ok := disabledReasonMap[i]; ok {
					nextReason[ni] = r
				}
			}
			ni++
		}
		if deleted == 0 {
			common.Fail(c, common.CodeParamError, "没有需要删除的自动禁用密钥")
			return
		}
		if len(nextKeys) == 0 {
			common.Fail(c, common.CodeParamError, "不能删除最后一个密钥")
			return
		}
		keys = nextKeys
		statusMap = nextStatus
		disabledTimeMap = nextTime
		disabledReasonMap = nextReason
		if err := saveState(); err != nil {
			common.Fail(c, common.CodeServerError, "更新多 Key 失败")
			return
		}
		common.OK(c, gin.H{"success": true, "message": "", "data": deleted})
		return

	default:
		common.Fail(c, common.CodeParamError, "不支持的操作")
		return
	}
}

func resolveOllamaChannel(channelID int) (*model.Channel, string, string, error) {
	var ch model.Channel
	if err := common.DB.First(&ch, channelID).Error; err != nil {
		return nil, "", "", fmt.Errorf("渠道不存在")
	}
	if ch.Type != model.ChannelTypeOllama {
		return nil, "", "", fmt.Errorf("该操作仅支持 Ollama 渠道")
	}

	baseURL := strings.TrimSpace(ch.BaseURL)
	if baseURL == "" {
		baseURL = ollamaAdapter.BaseURL
	}

	key := ""
	keys := splitLinesNonEmpty(ch.Key)
	if len(keys) > 0 {
		key = keys[0]
	}
	return &ch, baseURL, key, nil
}

func OllamaPullModel(c *gin.Context) {
	var req struct {
		ChannelID int    `json:"channel_id"`
		ModelName string `json:"model_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "请求参数错误")
		return
	}
	if req.ChannelID == 0 || strings.TrimSpace(req.ModelName) == "" {
		common.Fail(c, common.CodeParamError, "channel_id 和 model_name 不能为空")
		return
	}

	_, baseURL, key, err := resolveOllamaChannel(req.ChannelID)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	if err := ollamaAdapter.PullOllamaModel(baseURL, key, strings.TrimSpace(req.ModelName)); err != nil {
		common.Fail(c, common.CodeServerError, fmt.Sprintf("拉取模型失败: %s", err.Error()))
		return
	}

	common.OKMsg(c, fmt.Sprintf("模型 %s 拉取成功", req.ModelName), nil)
}

func OllamaPullModelStream(c *gin.Context) {
	var req struct {
		ChannelID int    `json:"channel_id"`
		ModelName string `json:"model_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "请求参数错误")
		return
	}
	if req.ChannelID == 0 || strings.TrimSpace(req.ModelName) == "" {
		common.Fail(c, common.CodeParamError, "channel_id 和 model_name 不能为空")
		return
	}

	_, baseURL, key, err := resolveOllamaChannel(req.ChannelID)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	progressCallback := func(progress ollamaAdapter.OllamaPullResponse) {
		data, _ := json.Marshal(progress)
		_, _ = c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
		c.Writer.Flush()
	}

	err = ollamaAdapter.PullOllamaModelStream(baseURL, key, strings.TrimSpace(req.ModelName), progressCallback)
	if err != nil {
		errorData, _ := json.Marshal(gin.H{"error": err.Error()})
		_, _ = c.Writer.Write([]byte("data: " + string(errorData) + "\n\n"))
	} else {
		successData, _ := json.Marshal(gin.H{"message": fmt.Sprintf("模型 %s 拉取成功", req.ModelName)})
		_, _ = c.Writer.Write([]byte("data: " + string(successData) + "\n\n"))
	}
	_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
	c.Writer.Flush()
}

func OllamaDeleteModel(c *gin.Context) {
	var req struct {
		ChannelID int    `json:"channel_id"`
		ModelName string `json:"model_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "请求参数错误")
		return
	}
	if req.ChannelID == 0 || strings.TrimSpace(req.ModelName) == "" {
		common.Fail(c, common.CodeParamError, "channel_id 和 model_name 不能为空")
		return
	}

	_, baseURL, key, err := resolveOllamaChannel(req.ChannelID)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	if err := ollamaAdapter.DeleteOllamaModel(baseURL, key, strings.TrimSpace(req.ModelName)); err != nil {
		common.Fail(c, common.CodeServerError, fmt.Sprintf("删除模型失败: %s", err.Error()))
		return
	}

	common.OKMsg(c, fmt.Sprintf("模型 %s 删除成功", req.ModelName), nil)
}

func OllamaVersion(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "渠道 ID 非法")
		return
	}

	_, baseURL, key, err := resolveOllamaChannel(id)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	version, err := ollamaAdapter.FetchOllamaVersion(baseURL, key)
	if err != nil {
		common.Fail(c, common.CodeServerError, fmt.Sprintf("获取 Ollama 版本失败: %s", err.Error()))
		return
	}

	common.OK(c, gin.H{"success": true, "data": gin.H{"version": version}})
}

func MarshalJSONOrEmpty(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
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
