package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	paymentservice "STfreApi/service/payment"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	epay "github.com/liuscraft/epay-sdk-go"
	"gorm.io/gorm"
)

const (
	BillingPreferenceSubscriptionFirst = "subscription_first"
	BillingPreferenceWalletFirst       = "wallet_first"
	BillingPreferenceSubscriptionOnly  = "subscription_only"
	BillingPreferenceWalletOnly        = "wallet_only"
)

func normalizeBillingPreference(pref string) string {
	switch strings.TrimSpace(pref) {
	case BillingPreferenceSubscriptionFirst,
		BillingPreferenceWalletFirst,
		BillingPreferenceSubscriptionOnly,
		BillingPreferenceWalletOnly:
		return strings.TrimSpace(pref)
	default:
		return BillingPreferenceSubscriptionFirst
	}
}

// ─── Admin: Plan CRUD ───────────────────────────────────────────────────────

func GetAllSubscriptionPlans(c *gin.Context) {
	var plans []model.SubscriptionPlan
	common.DB.Order("sort_order asc, id asc").Find(&plans)
	common.OK(c, plans)
}

func AddSubscriptionPlan(c *gin.Context) {
	var plan model.SubscriptionPlan
	if err := c.ShouldBindJSON(&plan); err != nil {
		common.Fail(c, common.CodeParamError, "参数解析失败")
		return
	}
	plan.Id = 0
	plan.CreatedAt = time.Now().Unix()
	if err := common.DB.Create(&plan).Error; err != nil {
		common.Fail(c, common.CodeServerError, "创建失败")
		return
	}
	common.OK(c, plan)
}

func UpdateSubscriptionPlan(c *gin.Context) {
	var plan model.SubscriptionPlan
	if err := c.ShouldBindJSON(&plan); err != nil {
		common.Fail(c, common.CodeParamError, "参数解析失败")
		return
	}
	if err := common.DB.Model(&plan).Updates(&plan).Error; err != nil {
		common.Fail(c, common.CodeServerError, "更新失败")
		return
	}
	common.OK(c, plan)
}

func DeleteSubscriptionPlan(c *gin.Context) {
	id := c.Param("id")
	if err := common.DB.Delete(&model.SubscriptionPlan{}, id).Error; err != nil {
		common.Fail(c, common.CodeServerError, "删除失败")
		return
	}
	common.OKMsg(c, "删除成功", nil)
}

// ─── Public: List enabled plans ─────────────────────────────────────────────

func GetPublicSubscriptionPlans(c *gin.Context) {
	var plans []model.SubscriptionPlan
	common.DB.Where("enabled = ?", true).Order("sort_order asc, id asc").Find(&plans)
	common.OK(c, plans)
}

// ─── User: Subscribe ─────────────────────────────────────────────────────────

func CreateSubscriptionOrder(c *gin.Context) {
	var req struct {
		PlanId  int    `json:"plan_id"`
		PayType string `json:"pay_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId == 0 {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}

	var plan model.SubscriptionPlan
	if err := common.DB.First(&plan, req.PlanId).Error; err != nil || !plan.Enabled {
		common.Fail(c, common.CodeNotFound, "套餐不存在或已下架")
		return
	}

	userId := c.GetInt("id")

	provider := common.OptionMap[model.OptionKeyPaymentProvider]
	// ～provider 没配置，或者填了奇怪的值，拒绝掉，免得创建一堆永远 pending 的僵尸订阅喵～
	if provider != "epay" && provider != "stripe" && provider != "creem" {
		common.Fail(c, common.CodeNotImplemented, "当前未配置有效的支付渠道，请联系管理员")
		return
	}

	sub := model.UserSubscription{
		UserId:    userId,
		PlanId:    plan.Id,
		Status:    model.SubStatusPending,
		CreatedAt: time.Now().Unix(),
	}
	if err := common.DB.Create(&sub).Error; err != nil {
		common.Fail(c, common.CodeServerError, "创建订阅失败")
		return
	}

	tradeNo := paymentservice.NewTradeNo()
	var payUrl string
	var err error

	switch provider {
	case "epay":
		payUrl, err = buildSubscriptionEpayUrl(tradeNo, plan, req.PayType)

	case "stripe":
		if plan.StripePriceId == "" {
			common.DB.Delete(&sub)
			common.Fail(c, common.CodeParamError, "该套餐未配置 Stripe 定价ID，请联系管理员")
			return
		}
		secretKey := common.OptionMap[model.OptionKeyStripeSecretKey]
		successUrl := common.OptionMap[model.OptionKeyStripeSuccessUrl]
		cancelUrl := common.OptionMap[model.OptionKeyStripeCancelUrl]
		systemUrl := common.OptionMap[model.OptionKeySystemUrl]
		if successUrl == "" {
			successUrl = systemUrl + "/billing?stripe_success=1"
		}
		if cancelUrl == "" {
			cancelUrl = systemUrl + "/billing"
		}
		var result *paymentservice.StripeCheckoutResult
		result, err = paymentservice.CreateStripeSubscriptionCheckout(secretKey, successUrl, cancelUrl, plan.StripePriceId, tradeNo)
		if err == nil {
			payUrl = result.PayUrl
		}

	case "creem":
		if plan.CreemProductId == "" {
			common.DB.Delete(&sub)
			common.Fail(c, common.CodeParamError, "该套餐未配置 Creem 产品ID，请联系管理员")
			return
		}
		apiKey := common.OptionMap[model.OptionKeyCreemApiKey]
		successUrl := common.OptionMap[model.OptionKeyCreemSuccessUrl]
		testMode := common.OptionMap[model.OptionKeyCreemTestMode] == "true"
		if successUrl == "" {
			successUrl = common.OptionMap[model.OptionKeySystemUrl] + "/billing?creem_success=1"
		}
		var result *paymentservice.CreemCheckoutResult
		result, err = paymentservice.CreateCreemCheckout(apiKey, plan.CreemProductId, successUrl, tradeNo, testMode)
		if err == nil {
			payUrl = result.PayUrl
		}
	}

	if err != nil {
		common.DB.Delete(&sub)
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	order := model.PaymentOrder{
		UserId:    userId,
		Amount:    plan.Price,
		Status:    model.PaymentStatusPending,
		Provider:  provider,
		OrderType: "subscription",
		RefId:     sub.Id,
		TradeNo:   tradeNo,
		PayUrl:    payUrl,
		CreatedAt: time.Now().Unix(),
	}
	if err := common.DB.Create(&order).Error; err != nil {
		common.DB.Delete(&sub)
		common.Fail(c, common.CodeServerError, "创建订单失败")
		return
	}

	common.OK(c, gin.H{"order": order, "subscription": sub})
}

func buildSubscriptionEpayUrl(tradeNo string, plan model.SubscriptionPlan, payType string) (string, error) {
	api := strings.TrimSpace(common.OptionMap[model.OptionKeyEpayApiUrl])
	pid := strings.TrimSpace(common.OptionMap[model.OptionKeyEpayPid])
	key := strings.TrimSpace(common.OptionMap[model.OptionKeyEpayKey])
	if payType == "" {
		payType = strings.TrimSpace(common.OptionMap[model.OptionKeyEpayType])
	}
	notifyUrl := strings.TrimSpace(common.OptionMap[model.OptionKeyEpayNotifyUrl])
	returnUrl := strings.TrimSpace(common.OptionMap[model.OptionKeyEpayReturnUrl])

	if api == "" || pid == "" || key == "" {
		return "", fmt.Errorf("易支付配置不完整")
	}
	if notifyUrl == "" {
		notifyUrl = common.OptionMap[model.OptionKeySystemUrl] + "/api/payment/notify"
	}
	if returnUrl == "" {
		returnUrl = common.OptionMap[model.OptionKeySystemUrl]
	}

	params := map[string]string{
		"pid":          pid,
		"out_trade_no": tradeNo,
		"notify_url":   notifyUrl,
		"return_url":   returnUrl,
		"name":         "订阅：" + plan.Name,
		"money":        fmt.Sprintf("%.2f", plan.Price),
	}
	if payType != "" {
		params["type"] = payType
	}

	signed := epay.NewSigner(key).SignWithParams(params)
	query := epay.BuildURLQuery(signed)
	return strings.TrimRight(api, "/") + "/pay/submit.php?" + query, nil
}

// ─── User: My subscriptions ──────────────────────────────────────────────────

type billingPreferenceRequest struct {
	BillingPreference string `json:"billing_preference"`
}

func GetMySubscriptions(c *gin.Context) {
	userId := c.GetInt("id")
	model.ExpireSubscriptions()

	var subs []model.UserSubscription
	common.DB.Preload("Plan").Where("user_id = ?", userId).Order("id desc").Find(&subs)
	common.OK(c, subs)
}

// GetSubscriptionSelf 对齐 new-api 的 /api/subscription/self。
func GetSubscriptionSelf(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}

	model.ExpireSubscriptions()

	var user model.User
	if err := common.DB.Select("id", "billing_preference").First(&user, userId).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}
	pref := normalizeBillingPreference(user.BillingPreference)

	now := time.Now().Unix()
	var activeSubs []model.UserSubscription
	if err := common.DB.Preload("Plan").
		Where("user_id = ? AND status = ? AND (expired_at = 0 OR expired_at > ?)", userId, model.SubStatusActive, now).
		Order("id desc").
		Find(&activeSubs).Error; err != nil {
		common.Fail(c, common.CodeServerError, "查询失败")
		return
	}

	var allSubs []model.UserSubscription
	if err := common.DB.Preload("Plan").
		Where("user_id = ?", userId).
		Order("id desc").
		Find(&allSubs).Error; err != nil {
		common.Fail(c, common.CodeServerError, "查询失败")
		return
	}

	common.OK(c, gin.H{
		"billing_preference": pref,
		"subscriptions":      activeSubs,
		"all_subscriptions":  allSubs,
	})
}

// UpdateSubscriptionPreference 对齐 new-api 的 /api/subscription/self/preference。
func UpdateSubscriptionPreference(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}

	var req billingPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	pref := normalizeBillingPreference(req.BillingPreference)

	if err := common.DB.Model(&model.User{}).Where("id = ?", userId).Update("billing_preference", pref).Error; err != nil {
		common.Fail(c, common.CodeServerError, "更新失败")
		return
	}

	common.OK(c, gin.H{"billing_preference": pref})
}

// ─── Internal: Activate subscription ────────────────────────────────────────
// ～真正的激活逻辑搬到 model.ActivateSubscription 去了（CAS 版，防并发重复执行），
// 这里只是给管理员绑定场景保留一个薄薄的调用入口喵～

// ─── Admin: Subscription APIs (new-api compatibility) ────────────────────────

type adminSubscriptionPlanDTO struct {
	Plan model.SubscriptionPlan `json:"plan"`
}

type adminUpsertSubscriptionPlanRequest struct {
	Plan model.SubscriptionPlan `json:"plan"`
}

func AdminListSubscriptionPlans(c *gin.Context) {
	var plans []model.SubscriptionPlan
	if err := common.DB.Order("sort_order asc, id asc").Find(&plans).Error; err != nil {
		common.Fail(c, common.CodeServerError, "查询失败")
		return
	}
	result := make([]adminSubscriptionPlanDTO, 0, len(plans))
	for _, p := range plans {
		result = append(result, adminSubscriptionPlanDTO{Plan: p})
	}
	common.OK(c, result)
}

func AdminCreateSubscriptionPlan(c *gin.Context) {
	var req adminUpsertSubscriptionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数解析失败")
		return
	}
	plan := req.Plan
	plan.Id = 0
	plan.CreatedAt = time.Now().Unix()
	if strings.TrimSpace(plan.Name) == "" {
		common.Fail(c, common.CodeParamError, "套餐名称不能为空")
		return
	}
	if plan.Price < 0 {
		common.Fail(c, common.CodeParamError, "价格不能为负数")
		return
	}
	if err := common.DB.Create(&plan).Error; err != nil {
		common.Fail(c, common.CodeServerError, "创建失败")
		return
	}
	common.OK(c, plan)
}

func AdminUpdateSubscriptionPlan(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.Fail(c, common.CodeParamError, "无效的ID")
		return
	}

	var req adminUpsertSubscriptionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数解析失败")
		return
	}

	plan := req.Plan
	if strings.TrimSpace(plan.Name) == "" {
		common.Fail(c, common.CodeParamError, "套餐名称不能为空")
		return
	}
	if plan.Price < 0 {
		common.Fail(c, common.CodeParamError, "价格不能为负数")
		return
	}

	updates := map[string]interface{}{
		"name":          plan.Name,
		"description":   plan.Description,
		"price":         plan.Price,
		"duration_days": plan.DurationDays,
		"type":          plan.Type,
		"group_name":    plan.GroupName,
		"quota":         plan.Quota,
		"rpm":           plan.RPM,
		"enabled":       plan.Enabled,
		"sort_order":    plan.SortOrder,
	}
	if err := common.DB.Model(&model.SubscriptionPlan{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		common.Fail(c, common.CodeServerError, "更新失败")
		return
	}
	common.OK(c, nil)
}

type adminUpdateSubscriptionPlanStatusRequest struct {
	Enabled *bool `json:"enabled"`
}

func AdminUpdateSubscriptionPlanStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.Fail(c, common.CodeParamError, "无效的ID")
		return
	}
	var req adminUpdateSubscriptionPlanStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		common.Fail(c, common.CodeParamError, "参数解析失败")
		return
	}
	if err := common.DB.Model(&model.SubscriptionPlan{}).Where("id = ?", id).Update("enabled", *req.Enabled).Error; err != nil {
		common.Fail(c, common.CodeServerError, "更新失败")
		return
	}
	common.OK(c, nil)
}

type adminBindSubscriptionRequest struct {
	UserId int `json:"user_id"`
	PlanId int `json:"plan_id"`
}

func adminBindSubscriptionTx(tx *gorm.DB, userId, planId int) error {
	var user model.User
	if err := tx.First(&user, userId).Error; err != nil {
		return fmt.Errorf("用户不存在")
	}
	var plan model.SubscriptionPlan
	if err := tx.First(&plan, planId).Error; err != nil {
		return fmt.Errorf("套餐不存在")
	}
	sub := model.UserSubscription{
		UserId:    userId,
		PlanId:    plan.Id,
		Status:    model.SubStatusPending,
		CreatedAt: time.Now().Unix(),
	}
	if err := tx.Create(&sub).Error; err != nil {
		return err
	}
	return model.ActivateSubscription(tx, sub.Id)
}

func AdminBindSubscription(c *gin.Context) {
	var req adminBindSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.UserId <= 0 || req.PlanId <= 0 {
		common.Fail(c, common.CodeParamError, "参数解析失败")
		return
	}
	if err := common.DB.Transaction(func(tx *gorm.DB) error {
		return adminBindSubscriptionTx(tx, req.UserId, req.PlanId)
	}); err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}
	common.OK(c, nil)
}

func AdminListUserSubscriptions(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Param("id"))
	if userId <= 0 {
		common.Fail(c, common.CodeParamError, "无效的用户ID")
		return
	}
	model.ExpireSubscriptions()
	var subs []model.UserSubscription
	if err := common.DB.Preload("Plan").Where("user_id = ?", userId).Order("id desc").Find(&subs).Error; err != nil {
		common.Fail(c, common.CodeServerError, "查询失败")
		return
	}
	common.OK(c, subs)
}

type adminCreateUserSubscriptionRequest struct {
	PlanId int `json:"plan_id"`
}

func AdminCreateUserSubscription(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Param("id"))
	if userId <= 0 {
		common.Fail(c, common.CodeParamError, "无效的用户ID")
		return
	}
	var req adminCreateUserSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.Fail(c, common.CodeParamError, "参数解析失败")
		return
	}
	if err := common.DB.Transaction(func(tx *gorm.DB) error {
		return adminBindSubscriptionTx(tx, userId, req.PlanId)
	}); err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}
	common.OK(c, nil)
}

func AdminInvalidateUserSubscription(c *gin.Context) {
	subId, _ := strconv.Atoi(c.Param("id"))
	if subId <= 0 {
		common.Fail(c, common.CodeParamError, "无效的订阅ID")
		return
	}
	if err := common.DB.Transaction(func(tx *gorm.DB) error {
		var sub model.UserSubscription
		if err := tx.Preload("Plan").First(&sub, subId).Error; err != nil {
			return fmt.Errorf("订阅不存在")
		}
		if sub.Status == model.SubStatusCancelled || sub.Status == model.SubStatusExpired {
			return nil
		}
		if sub.Plan != nil && (sub.Plan.Type == model.PlanTypeGroup || sub.Plan.Type == model.PlanTypeCombo) && sub.OriginalGroup != "" {
			if err := tx.Model(&model.User{}).Where("id = ?", sub.UserId).Update("group", sub.OriginalGroup).Error; err != nil {
				return err
			}
		}
		updates := map[string]interface{}{
			"status":     model.SubStatusCancelled,
			"expired_at": time.Now().Unix(),
		}
		return tx.Model(&model.UserSubscription{}).Where("id = ?", subId).Updates(updates).Error
	}); err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}
	common.OK(c, nil)
}

func AdminDeleteUserSubscription(c *gin.Context) {
	subId, _ := strconv.Atoi(c.Param("id"))
	if subId <= 0 {
		common.Fail(c, common.CodeParamError, "无效的订阅ID")
		return
	}
	if err := common.DB.Transaction(func(tx *gorm.DB) error {
		var sub model.UserSubscription
		if err := tx.Preload("Plan").First(&sub, subId).Error; err != nil {
			return fmt.Errorf("订阅不存在")
		}
		if sub.Status == model.SubStatusActive && sub.Plan != nil && (sub.Plan.Type == model.PlanTypeGroup || sub.Plan.Type == model.PlanTypeCombo) && sub.OriginalGroup != "" {
			if err := tx.Model(&model.User{}).Where("id = ?", sub.UserId).Update("group", sub.OriginalGroup).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&model.UserSubscription{}, subId).Error
	}); err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}
	common.OK(c, nil)
}
