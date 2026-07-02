package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"STfreApi/common"
	oauthctl "STfreApi/controller/oauth"
	"STfreApi/model"
	"STfreApi/service"
	passkeysvc "STfreApi/service/passkey"

	"github.com/gin-gonic/gin"
)

// EmailBind 对齐 new-api: GET /api/oauth/email/bind?email=...&code=...
func EmailBind(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}

	emailAddr := strings.TrimSpace(c.Query("email"))
	code := strings.TrimSpace(c.Query("code"))
	if emailAddr == "" || code == "" {
		common.Fail(c, common.CodeParamError, "邮箱或验证码不能为空")
		return
	}
	if !CheckEmailVerifyCode(emailAddr, code) {
		common.Fail(c, common.CodeParamError, "验证码无效或已过期")
		return
	}

	// 与 new-api 保持“验证码通过即绑定”语义
	if err := common.DB.Model(&model.User{}).Where("id = ?", userID).Update("email", emailAddr).Error; err != nil {
		common.Fail(c, common.CodeServerError, "邮箱绑定失败")
		return
	}
	common.OKMsg(c, "邮箱绑定成功", nil)
}

// TelegramLogin 复用现有 Telegram OAuth 回调逻辑。
func TelegramLogin(c *gin.Context) {
	oauthctl.TelegramCallback(c)
}

// TelegramBind 对齐 new-api: 已登录用户绑定 Telegram。
func TelegramBind(c *gin.Context) {
	if common.TelegramBotToken == "" {
		common.Fail(c, common.CodeForbidden, "管理员未开启 Telegram 登录")
		return
	}

	userID := c.GetInt("id")
	if userID <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}

	if !checkTelegramAuthorization(c, common.TelegramBotToken) {
		common.Fail(c, common.CodeParamError, "无效的请求")
		return
	}

	telegramID := strings.TrimSpace(c.Query("id"))
	if telegramID == "" {
		common.Fail(c, common.CodeParamError, "缺少 Telegram ID")
		return
	}

	var count int64
	common.DB.Model(&model.User{}).
		Where("telegram_id = ? AND id <> ?", telegramID, userID).
		Count(&count)
	if count > 0 {
		common.Fail(c, common.CodeConflict, "该 Telegram 账户已被绑定")
		return
	}

	if err := common.DB.Model(&model.User{}).Where("id = ?", userID).Update("telegram_id", telegramID).Error; err != nil {
		common.Fail(c, common.CodeServerError, "绑定 Telegram 失败")
		return
	}

	common.OKMsg(c, "绑定成功", nil)
}

func checkTelegramAuthorization(c *gin.Context, token string) bool {
	hash := c.Query("hash")
	if hash == "" {
		return false
	}

	params := []string{}
	for key, values := range c.Request.URL.Query() {
		if key == "hash" || len(values) == 0 {
			continue
		}
		params = append(params, fmt.Sprintf("%s=%s", key, values[0]))
	}
	sort.Strings(params)
	dataCheckString := strings.Join(params, "\n")

	secretKey := sha256.Sum256([]byte(token))
	mac := hmac.New(sha256.New, secretKey[:])
	_, _ = mac.Write([]byte(dataCheckString))
	return hex.EncodeToString(mac.Sum(nil)) == hash
}

// StripeWebhook / CreemWebhook: 当前项目统一复用 PaymentNotify 处理支付回调。
func StripeWebhook(c *gin.Context) {
	PaymentNotify(c)
}

func CreemWebhook(c *gin.Context) {
	PaymentNotify(c)
}

// 订阅 Epay 回调与返回：复用统一支付通知。
func SubscriptionEpayNotify(c *gin.Context) {
	PaymentNotify(c)
}

func SubscriptionEpayReturn(c *gin.Context) {
	PaymentNotify(c)
}

// GetUserGroups 对齐 new-api 的 /api/user/groups 与 /api/user/self/groups。
func GetUserGroups(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}

	var user model.User
	if err := common.DB.Select("id", "group").First(&user, userID).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}

	var groups []model.Group
	if err := common.DB.Find(&groups).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取分组失败")
		return
	}

	usable := make(map[string]map[string]interface{})
	for _, g := range groups {
		name := strings.TrimSpace(g.Name)
		if name == "" {
			continue
		}
		usable[name] = map[string]interface{}{
			"ratio": 1,
			"desc":  g.Description,
		}
	}
	if user.Group != "" {
		if _, ok := usable[user.Group]; !ok {
			usable[user.Group] = map[string]interface{}{
				"ratio": 1,
				"desc":  "",
			}
		}
	}

	common.OK(c, usable)
}

// GetChannelAffinityUsageCacheStats 读取真实渠道亲和缓存明细统计。
func GetChannelAffinityUsageCacheStats(c *gin.Context) {
	ruleName := strings.TrimSpace(c.Query("rule_name"))
	keyFP := strings.TrimSpace(c.Query("key_fp"))
	usingGroup := strings.TrimSpace(c.Query("using_group"))

	stats, err := getChannelAffinityUsageCacheStats(ruleName, usingGroup, keyFP)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	common.OK(c, stats)
}

// Logout 对齐 new-api 的 /api/user/logout：将当前 JWT 拉黑，使其在到期前立即失效。
func Logout(c *gin.Context) {
	tokenString, _ := c.Get("jwt_token")
	expiresAt, _ := c.Get("jwt_expires_at")
	if ts, ok := tokenString.(string); ok && ts != "" {
		if exp, ok := expiresAt.(time.Time); ok {
			common.BlacklistToken(ts, exp)
		}
	}
	common.OKMsg(c, "登出成功", nil)
}

// DeleteSelf 对齐 new-api 的 /api/user/self DELETE。
func DeleteSelf(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}
	if err := common.DB.Delete(&model.User{}, userID).Error; err != nil {
		common.Fail(c, common.CodeServerError, "注销失败")
		return
	}
	common.OKMsg(c, "账号已注销", nil)
}

// GenerateAccessToken 对齐 new-api 的 /api/user/token。
func GenerateAccessToken(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}
	var user model.User
	if err := common.DB.First(&user, userID).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}
	token, err := common.GenerateToken(user.Id, user.Username, user.Role)
	if err != nil {
		common.Fail(c, common.CodeServerError, "生成令牌失败")
		return
	}
	_ = common.DB.Model(&model.User{}).Where("id = ?", userID).Update("access_token", token).Error
	common.OK(c, gin.H{"access_token": token})
}

// GetUserModels 对齐 new-api 的 /api/user/models，复用现有模型聚合逻辑。
func GetUserModels(c *gin.Context) {
	ListModels(c)
}

// GetAffCode 对齐 new-api 的 /api/user/aff。
func GetAffCode(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}
	var user model.User
	if err := common.DB.Select("id", "aff_code").First(&user, userID).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}
	if strings.TrimSpace(user.AffCode) == "" {
		user.AffCode = fmt.Sprintf("aff_%d_%d", userID, time.Now().Unix()%100000)
		_ = common.DB.Model(&model.User{}).Where("id = ?", userID).Update("aff_code", user.AffCode).Error
	}
	common.OK(c, gin.H{"aff_code": user.AffCode})
}

// GetTopUpInfo 对齐 new-api 的 /api/user/topup/info。
func GetTopUpInfo(c *gin.Context) {
	common.OptionLock.RLock()
	enableTopup := common.OptionMap[model.OptionKeyEnableTopup] == "true"
	provider := strings.TrimSpace(common.OptionMap[model.OptionKeyPaymentProvider])
	minTopup := strings.TrimSpace(common.OptionMap[model.OptionKeyMinTopup])
	price := strings.TrimSpace(common.OptionMap["price"])
	common.OptionLock.RUnlock()
	common.OK(c, gin.H{
		"enable_topup":  enableTopup,
		"provider":      provider,
		"min_topup":     minTopup,
		"display_price": price,
	})
}

// GetUserTopUps 对齐 new-api 的 /api/user/topup/self。
func GetUserTopUps(c *gin.Context) {
	ListPayments(c)
}

// TopUp / RequestEpay / Stripe / Creem 兼容到统一下单能力。
func TopUp(c *gin.Context) {
	CreatePayment(c)
}

func RequestEpay(c *gin.Context) {
	CreatePayment(c)
}

func RequestStripePay(c *gin.Context) {
	CreatePayment(c)
}

func RequestCreemPay(c *gin.Context) {
	CreatePayment(c)
}

func RequestAmount(c *gin.Context) {
	var req struct {
		Amount float64 `json:"amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	common.OK(c, gin.H{"amount": req.Amount, "quota": int64(req.Amount * 500000)})
}

func RequestStripeAmount(c *gin.Context) {
	RequestAmount(c)
}

// TransferAffQuota 将当前用户的邀请返利余额（aff_quota）转入可用额度（quota）。
func TransferAffQuota(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}

	var req struct {
		Quota int64 `json:"quota" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Quota <= 0 {
		common.Fail(c, common.CodeParamError, "转移额度必须大于 0")
		return
	}

	if err := model.TransferAffQuotaToQuota(userID, req.Quota); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	var user model.User
	if err := common.DB.Select("id", "username").First(&user, userID).Error; err == nil {
		_ = common.DB.Create(&model.Log{
			UserId: userID, Username: user.Username, CreatedAt: time.Now().Unix(),
			Type: model.LogTypeSystem, Content: "邀请返利转入可用额度", Quota: req.Quota, ModelName: "system",
		}).Error
	}

	common.OKMsg(c, "转账成功", nil)
}

// UpdateUserSetting 对齐 new-api 的 /api/user/setting，复用 UpdateSelf。
func UpdateUserSetting(c *gin.Context) {
	UpdateSelf(c)
}

// Get2FAStatus / Setup2FA / Enable2FA / Disable2FA / RegenerateBackupCodes 兼容层。
func Get2FAStatus(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}
	var user model.User
	if err := common.DB.Select("id", "totp_enabled", "backup_codes").First(&user, userID).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}
	var codes []string
	_ = json.Unmarshal([]byte(user.BackupCodes), &codes)
	common.OK(c, gin.H{"enabled": user.TOTPEnabled, "backup_codes_count": len(codes)})
}

func Setup2FA(c *gin.Context) {
	TOTPSetup(c)
}

func Enable2FA(c *gin.Context) {
	TOTPEnable(c)
}

func Disable2FA(c *gin.Context) {
	TOTPDisable(c)
}

func RegenerateBackupCodes(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}
	var user model.User
	if err := common.DB.First(&user, userID).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}
	if !user.TOTPEnabled {
		common.Fail(c, common.CodeParamError, "2FA 未启用")
		return
	}
	codes, err := generateBackupCodes(8)
	if err != nil {
		common.Fail(c, common.CodeServerError, "生成备份码失败")
		return
	}
	data, _ := json.Marshal(codes)
	if err = common.DB.Model(&model.User{}).Where("id = ?", userID).Update("backup_codes", string(data)).Error; err != nil {
		common.Fail(c, common.CodeServerError, "保存备份码失败")
		return
	}
	common.OK(c, gin.H{"backup_codes": codes})
}

// PasskeyVerifyBegin 已登录用户用 Passkey 做二次验证（assertion flow），
// 与登录的 discoverable login 不同，这里针对特定用户的凭证发起 BeginLogin。
func PasskeyVerifyBegin(c *gin.Context) {
	userID := c.GetInt("id")
	if userID == 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}

	credential, err := model.GetPasskeyByUserID(userID)
	if err != nil {
		common.Fail(c, common.CodeNotFound, "该用户尚未绑定 Passkey")
		return
	}

	var user model.User
	if err := common.DB.First(&user, userID).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}

	wa, err := passkeysvc.BuildWebAuthn(c.Request)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	waUser := passkeysvc.NewWebAuthnUser(&user, credential)
	assertion, sessionData, err := wa.BeginLogin(waUser)
	if err != nil {
		common.Fail(c, common.CodeServerError, fmt.Sprintf("生成 Passkey 验证参数失败: %v", err))
		return
	}

	sessionID, err := passkeysvc.SaveVerifySession(sessionData)
	if err != nil {
		common.Fail(c, common.CodeServerError, "保存 Passkey 会话失败")
		return
	}

	common.OK(c, gin.H{"session_id": sessionID, "options": assertion})
}

// PasskeyVerifyFinish 完成 Passkey 二次验证，校验通过后仅返回成功，不重新发 token。
func PasskeyVerifyFinish(c *gin.Context) {
	userID := c.GetInt("id")
	if userID == 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}

	var req struct {
		SessionID string `json:"session_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "缺少 session_id")
		return
	}

	sessionData, err := passkeysvc.PopVerifySession(req.SessionID)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	credential, err := model.GetPasskeyByUserID(userID)
	if err != nil {
		common.Fail(c, common.CodeNotFound, "该用户尚未绑定 Passkey")
		return
	}

	var user model.User
	if err := common.DB.First(&user, userID).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}

	wa, err := passkeysvc.BuildWebAuthn(c.Request)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	waUser := passkeysvc.NewWebAuthnUser(&user, credential)
	updatedCred, err := wa.FinishLogin(waUser, *sessionData, c.Request)
	if err != nil {
		common.Fail(c, common.CodeUnauthorized, fmt.Sprintf("Passkey 验证失败: %v", err))
		return
	}

	// 更新凭证最后使用时间
	updated := model.NewPasskeyCredentialFromWebAuthn(userID, updatedCred)
	if updated != nil {
		now := time.Now()
		updated.LastUsedAt = &now
		_ = model.UpsertPasskeyCredential(updated)
	}

	common.OKMsg(c, "Passkey 验证成功", nil)
}

// Admin2FAStats 对齐 new-api 的管理员 2FA 统计接口。
func Admin2FAStats(c *gin.Context) {
	var total int64
	var enabled int64

	if err := common.DB.Model(&model.User{}).Count(&total).Error; err != nil {
		common.Fail(c, common.CodeServerError, "统计失败")
		return
	}
	if err := common.DB.Model(&model.User{}).Where("totp_enabled = ?", true).Count(&enabled).Error; err != nil {
		common.Fail(c, common.CodeServerError, "统计失败")
		return
	}

	disabled := total - enabled
	if disabled < 0 {
		disabled = 0
	}

	common.OK(c, gin.H{
		"total_users":    total,
		"enabled_users":  enabled,
		"disabled_users": disabled,
	})
}

// AdminDisable2FA 对齐 new-api 的管理员禁用用户 2FA。
func AdminDisable2FA(c *gin.Context) {
	userID := strings.TrimSpace(c.Param("id"))
	if userID == "" {
		common.Fail(c, common.CodeParamError, "用户 ID 不能为空")
		return
	}

	updates := map[string]interface{}{
		"totp_enabled":    false,
		"totp_secret":     "",
		"backup_codes":    "",
		"totp_fail_count": 0,
		"totp_locked_at":  0,
	}
	result := common.DB.Model(&model.User{}).Where("id = ?", userID).Updates(updates)
	if result.Error != nil {
		common.Fail(c, common.CodeServerError, "禁用 2FA 失败")
		return
	}
	if result.RowsAffected == 0 {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}

	common.OKMsg(c, "已禁用该用户 2FA", nil)
}

func fetchUpstreamModelList(channel model.Channel) ([]string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(channel.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	targetURL := baseURL + "/v1/models"

	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	key := service.GetNextKey(channel.Key)
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := common.NewHTTPClient(channel.Proxy).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("upstream status %d", resp.StatusCode)
	}

	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	models := make([]string, 0, len(payload.Data))
	seen := map[string]bool{}
	for _, item := range payload.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		models = append(models, id)
	}
	sort.Strings(models)
	return models, nil
}

func parseChannelIDs(c *gin.Context) ([]int, error) {
	var req struct {
		ChannelIDs []int `json:"channel_ids"`
		ChannelID  int   `json:"channel_id"`
		ID         int   `json:"id"`
	}
	_ = c.ShouldBindJSON(&req)
	ids := make([]int, 0)
	for _, id := range req.ChannelIDs {
		if id > 0 {
			ids = append(ids, id)
		}
	}
	if req.ChannelID > 0 {
		ids = append(ids, req.ChannelID)
	}
	if req.ID > 0 {
		ids = append(ids, req.ID)
	}
	uniq := map[int]bool{}
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if !uniq[id] {
			uniq[id] = true
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("缺少 channel_ids")
	}
	return out, nil
}

// FetchUpstreamModels 对齐 new-api: GET /api/channel/fetch_models/:id
func FetchUpstreamModels(c *gin.Context) {
	FetchChannelModels(c)
}

// FetchModels 对齐 new-api: POST /api/channel/fetch_models
// 兼容两种请求：
// 1) 传 channel_ids/channel_id/id（按已存在渠道 ID 拉取）
// 2) 直接传新增表单字段 type/key/base_url/proxy（未保存渠道预拉取）
func FetchModels(c *gin.Context) {
	var req struct {
		ChannelIDs []int  `json:"channel_ids"`
		ChannelID  int    `json:"channel_id"`
		ID         int    `json:"id"`
		Type       int    `json:"type"`
		Key        string `json:"key"`
		BaseURL    string `json:"base_url"`
		Proxy      string `json:"proxy"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "请求参数格式错误")
		return
	}

	// 优先按 payload 直拉（新增渠道场景）
	if strings.TrimSpace(req.Key) != "" || strings.TrimSpace(req.BaseURL) != "" || strings.TrimSpace(req.Proxy) != "" {
		ch := model.Channel{
			Type:    req.Type,
			Key:     req.Key,
			BaseURL: req.BaseURL,
			Proxy:   req.Proxy,
		}
		models, fetchErr := fetchUpstreamModelList(ch)
		if fetchErr != nil {
			common.Fail(c, common.CodeServerError, fetchErr.Error())
			return
		}
		common.OK(c, gin.H{"models": models})
		return
	}

	ids := make([]int, 0)
	for _, id := range req.ChannelIDs {
		if id > 0 {
			ids = append(ids, id)
		}
	}
	if req.ChannelID > 0 {
		ids = append(ids, req.ChannelID)
	}
	if req.ID > 0 {
		ids = append(ids, req.ID)
	}
	uniq := map[int]bool{}
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if !uniq[id] {
			uniq[id] = true
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		common.Fail(c, common.CodeParamError, "缺少 channel_ids 或可用于拉取模型的渠道参数")
		return
	}

	var channels []model.Channel
	if err := common.DB.Where("id IN ?", out).Find(&channels).Error; err != nil {
		common.Fail(c, common.CodeServerError, "查询渠道失败")
		return
	}

	results := make(map[string]interface{})
	for _, ch := range channels {
		models, fetchErr := fetchUpstreamModelList(ch)
		if fetchErr != nil {
			results[strconv.Itoa(ch.Id)] = gin.H{"error": fetchErr.Error()}
			continue
		}
		results[strconv.Itoa(ch.Id)] = gin.H{"models": models}
	}
	common.OK(c, results)
}

func detectUpstreamUpdatesByIDs(ids []int) gin.H {
	var channels []model.Channel
	_ = common.DB.Where("id IN ?", ids).Find(&channels).Error

	items := make([]gin.H, 0, len(channels))
	for _, ch := range channels {
		upstreamModels, err := fetchUpstreamModelList(ch)
		if err != nil {
			items = append(items, gin.H{"channel_id": ch.Id, "channel_name": ch.Name, "error": err.Error()})
			continue
		}
		localSet := map[string]bool{}
		for _, m := range strings.Split(ch.Models, ",") {
			m = strings.TrimSpace(m)
			if m != "" {
				localSet[m] = true
			}
		}
		upSet := map[string]bool{}
		for _, m := range upstreamModels {
			upSet[m] = true
		}

		added := make([]string, 0)
		removed := make([]string, 0)
		for m := range upSet {
			if !localSet[m] {
				added = append(added, m)
			}
		}
		for m := range localSet {
			if !upSet[m] {
				removed = append(removed, m)
			}
		}
		sort.Strings(added)
		sort.Strings(removed)

		items = append(items, gin.H{
			"channel_id":   ch.Id,
			"channel_name": ch.Name,
			"has_updates":  len(added) > 0 || len(removed) > 0,
			"added":        added,
			"removed":      removed,
		})
	}
	return gin.H{"items": items, "count": len(items)}
}

// DetectChannelUpstreamModelUpdates 对齐 new-api: POST /api/channel/upstream_updates/detect
func DetectChannelUpstreamModelUpdates(c *gin.Context) {
	ids, err := parseChannelIDs(c)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	common.OK(c, detectUpstreamUpdatesByIDs(ids))
}

// DetectAllChannelUpstreamModelUpdates 对齐 new-api: POST /api/channel/upstream_updates/detect_all
func DetectAllChannelUpstreamModelUpdates(c *gin.Context) {
	var ids []int
	_ = common.DB.Model(&model.Channel{}).Pluck("id", &ids).Error
	common.OK(c, detectUpstreamUpdatesByIDs(ids))
}

func applyUpstreamUpdatesByIDs(ids []int) gin.H {
	var channels []model.Channel
	_ = common.DB.Where("id IN ?", ids).Find(&channels).Error

	updated := 0
	items := make([]gin.H, 0, len(channels))
	for _, ch := range channels {
		upstreamModels, err := fetchUpstreamModelList(ch)
		if err != nil {
			items = append(items, gin.H{"channel_id": ch.Id, "channel_name": ch.Name, "error": err.Error()})
			continue
		}
		joined := strings.Join(upstreamModels, ",")
		if strings.TrimSpace(joined) == strings.TrimSpace(ch.Models) {
			items = append(items, gin.H{"channel_id": ch.Id, "channel_name": ch.Name, "updated": false})
			continue
		}
		if err = common.DB.Model(&model.Channel{}).Where("id = ?", ch.Id).Update("models", joined).Error; err != nil {
			items = append(items, gin.H{"channel_id": ch.Id, "channel_name": ch.Name, "error": err.Error()})
			continue
		}
		updated++
		items = append(items, gin.H{"channel_id": ch.Id, "channel_name": ch.Name, "updated": true, "model_count": len(upstreamModels)})
	}
	return gin.H{"updated": updated, "items": items}
}

// ApplyChannelUpstreamModelUpdates 对齐 new-api: POST /api/channel/upstream_updates/apply
func ApplyChannelUpstreamModelUpdates(c *gin.Context) {
	ids, err := parseChannelIDs(c)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	common.OK(c, applyUpstreamUpdatesByIDs(ids))
}

// ApplyAllChannelUpstreamModelUpdates 对齐 new-api: POST /api/channel/upstream_updates/apply_all
func ApplyAllChannelUpstreamModelUpdates(c *gin.Context) {
	var ids []int
	_ = common.DB.Model(&model.Channel{}).Pluck("id", &ids).Error
	common.OK(c, applyUpstreamUpdatesByIDs(ids))
}

func collectChannelModelNames(channels []model.Channel) []string {
	seen := make(map[string]bool)
	models := make([]string, 0)
	for _, channel := range channels {
		for _, name := range strings.Split(channel.Models, ",") {
			modelName := strings.TrimSpace(name)
			if modelName == "" || seen[modelName] {
				continue
			}
			seen[modelName] = true
			models = append(models, modelName)
		}
	}
	sort.Strings(models)
	return models
}

func queryExistingModelMetaNames(modelNames []string) map[string]bool {
	out := make(map[string]bool)
	if len(modelNames) == 0 {
		return out
	}
	var existing []string
	if err := common.DB.Model(&model.ModelMeta{}).Where("model_name IN ?", modelNames).Pluck("model_name", &existing).Error; err != nil {
		return out
	}
	for _, name := range existing {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" {
			out[trimmed] = true
		}
	}
	return out
}

func getActiveChannelsByIDs(ids []int) ([]model.Channel, error) {
	var channels []model.Channel
	db := common.DB.Model(&model.Channel{}).Where("status = ?", model.ChannelStatusActive)
	if len(ids) > 0 {
		db = db.Where("id IN ?", ids)
	}
	err := db.Find(&channels).Error
	return channels, err
}

func parseOptionalChannelIDs(c *gin.Context) []int {
	idsMap := make(map[int]bool)

	appendID := func(id int) {
		if id > 0 {
			idsMap[id] = true
		}
	}

	if raw := strings.TrimSpace(c.Query("channel_id")); raw != "" {
		if id, err := strconv.Atoi(raw); err == nil {
			appendID(id)
		}
	}
	if raw := strings.TrimSpace(c.Query("id")); raw != "" {
		if id, err := strconv.Atoi(raw); err == nil {
			appendID(id)
		}
	}
	if raw := strings.TrimSpace(c.Query("channel_ids")); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			if id, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
				appendID(id)
			}
		}
	}

	var req struct {
		ChannelIDs []int `json:"channel_ids"`
		ChannelID  int   `json:"channel_id"`
		ID         int   `json:"id"`
	}
	if c.Request != nil && c.Request.Body != nil {
		_ = c.ShouldBindJSON(&req)
		for _, id := range req.ChannelIDs {
			appendID(id)
		}
		appendID(req.ChannelID)
		appendID(req.ID)
	}

	ids := make([]int, 0, len(idsMap))
	for id := range idsMap {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// GetMissingModels 对齐 new-api: GET /api/models/missing
// 语义：返回“在启用渠道中出现，但尚未配置到 models meta 表”的模型名。
func GetMissingModels(c *gin.Context) {
	ids := parseOptionalChannelIDs(c)
	channels, err := getActiveChannelsByIDs(ids)
	if err != nil {
		common.Fail(c, common.CodeServerError, "读取渠道失败")
		return
	}

	referenced := collectChannelModelNames(channels)
	existingSet := queryExistingModelMetaNames(referenced)

	missing := make([]string, 0)
	for _, name := range referenced {
		if !existingSet[name] {
			missing = append(missing, name)
		}
	}

	common.OK(c, gin.H{
		"missing":       missing,
		"missing_count": len(missing),
		"channel_count": len(channels),
	})
}

// SyncUpstreamPreview 对齐 new-api: GET /api/models/sync_upstream/preview
// 语义：从渠道实时拉取 /v1/models，预览缺失模型（仅预览不写库）。
func SyncUpstreamPreview(c *gin.Context) {
	ids := parseOptionalChannelIDs(c)
	channels, err := getActiveChannelsByIDs(ids)
	if err != nil {
		common.Fail(c, common.CodeServerError, "读取渠道失败")
		return
	}
	if len(channels) == 0 {
		common.OK(c, gin.H{"missing": []string{}, "missing_count": 0, "channel_count": 0, "upstream_count": 0, "failed_channels": []gin.H{}})
		return
	}

	upstreamSet := make(map[string]bool)
	failedChannels := make([]gin.H, 0)
	for _, channel := range channels {
		models, fetchErr := fetchUpstreamModelList(channel)
		if fetchErr != nil {
			failedChannels = append(failedChannels, gin.H{"channel_id": channel.Id, "channel_name": channel.Name, "error": fetchErr.Error()})
			continue
		}
		for _, name := range models {
			trimmed := strings.TrimSpace(name)
			if trimmed != "" {
				upstreamSet[trimmed] = true
			}
		}
	}

	upstreamModels := make([]string, 0, len(upstreamSet))
	for name := range upstreamSet {
		upstreamModels = append(upstreamModels, name)
	}
	sort.Strings(upstreamModels)

	existingSet := queryExistingModelMetaNames(upstreamModels)
	missing := make([]string, 0)
	for _, name := range upstreamModels {
		if !existingSet[name] {
			missing = append(missing, name)
		}
	}

	common.OK(c, gin.H{
		"missing":         missing,
		"missing_count":   len(missing),
		"upstream_count":  len(upstreamModels),
		"channel_count":   len(channels),
		"failed_channels": failedChannels,
	})
}

// SyncUpstreamModels 对齐 new-api: POST /api/models/sync_upstream
// 语义：基于实时 upstream 模型列表，补齐缺失的 models meta 记录。
func SyncUpstreamModels(c *gin.Context) {
	var req struct {
		ChannelIDs []int `json:"channel_ids"`
		ChannelID  int   `json:"channel_id"`
		ID         int   `json:"id"`
		Enabled    *bool `json:"enabled"`
	}
	_ = c.ShouldBindJSON(&req)

	idsMap := make(map[int]bool)
	for _, id := range req.ChannelIDs {
		if id > 0 {
			idsMap[id] = true
		}
	}
	if req.ChannelID > 0 {
		idsMap[req.ChannelID] = true
	}
	if req.ID > 0 {
		idsMap[req.ID] = true
	}
	ids := make([]int, 0, len(idsMap))
	for id := range idsMap {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	channels, err := getActiveChannelsByIDs(ids)
	if err != nil {
		common.Fail(c, common.CodeServerError, "读取渠道失败")
		return
	}
	if len(channels) == 0 {
		common.OK(c, gin.H{"created": 0, "skipped": 0, "created_models": []string{}, "skipped_models": []string{}})
		return
	}

	upstreamSet := make(map[string]bool)
	failedChannels := make([]gin.H, 0)
	for _, channel := range channels {
		models, fetchErr := fetchUpstreamModelList(channel)
		if fetchErr != nil {
			failedChannels = append(failedChannels, gin.H{"channel_id": channel.Id, "channel_name": channel.Name, "error": fetchErr.Error()})
			continue
		}
		for _, name := range models {
			trimmed := strings.TrimSpace(name)
			if trimmed != "" {
				upstreamSet[trimmed] = true
			}
		}
	}

	upstreamModels := make([]string, 0, len(upstreamSet))
	for name := range upstreamSet {
		upstreamModels = append(upstreamModels, name)
	}
	sort.Strings(upstreamModels)

	existingSet := queryExistingModelMetaNames(upstreamModels)
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	createdModels := make([]string, 0)
	skippedModels := make([]string, 0)
	for _, name := range upstreamModels {
		if existingSet[name] {
			skippedModels = append(skippedModels, name)
			continue
		}
		item := &model.ModelMeta{
			VendorId:    0,
			ModelName:   name,
			DisplayName: name,
			ModelType:   "text",
			Enabled:     enabled,
		}
		if insertErr := item.Insert(); insertErr != nil {
			skippedModels = append(skippedModels, name)
			continue
		}
		createdModels = append(createdModels, name)
		existingSet[name] = true
	}

	common.OK(c, gin.H{
		"created":         len(createdModels),
		"skipped":         len(skippedModels),
		"created_models":  createdModels,
		"skipped_models":  skippedModels,
		"upstream_count":  len(upstreamModels),
		"channel_count":   len(channels),
		"failed_channels": failedChannels,
	})
}
