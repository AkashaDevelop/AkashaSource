package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"fmt"
	"strings"
	"time"

	epay "github.com/liuscraft/epay-sdk-go"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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

	// Create pending subscription
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

	// Create payment order
	provider := common.OptionMap[model.OptionKeyPaymentProvider]
	order := model.PaymentOrder{
		UserId:    userId,
		Amount:    plan.Price,
		Status:    model.PaymentStatusPending,
		Provider:  provider,
		OrderType: "subscription",
		RefId:     sub.Id,
		CreatedAt: time.Now().Unix(),
	}
	if err := common.DB.Create(&order).Error; err != nil {
		common.Fail(c, common.CodeServerError, "创建订单失败")
		return
	}

	if provider == "epay" {
		payUrl, err := buildSubscriptionEpayUrl(order, plan, req.PayType)
		if err != nil {
			common.Fail(c, common.CodeParamError, err.Error())
			return
		}
		order.PayUrl = payUrl
		common.DB.Model(&order).Update("pay_url", payUrl)
	}

	common.OK(c, gin.H{"order": order, "subscription": sub})
}

func buildSubscriptionEpayUrl(order model.PaymentOrder, plan model.SubscriptionPlan, payType string) (string, error) {
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
		"out_trade_no": fmt.Sprintf("%d", order.Id),
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

func GetMySubscriptions(c *gin.Context) {
	userId := c.GetInt("id")
	// Expire overdue subscriptions first
	model.ExpireSubscriptions()

	var subs []model.UserSubscription
	common.DB.Preload("Plan").Where("user_id = ?", userId).Order("id desc").Find(&subs)
	common.OK(c, subs)
}

// ─── Internal: Activate subscription ────────────────────────────────────────

func ActivateSubscription(tx *gorm.DB, subId int) error {
	var sub model.UserSubscription
	if err := tx.Preload("Plan").First(&sub, subId).Error; err != nil {
		return err
	}
	if sub.Status == model.SubStatusActive {
		return nil
	}

	plan := sub.Plan
	if plan == nil {
		return fmt.Errorf("套餐不存在")
	}

	now := time.Now().Unix()
	var expiredAt int64
	if plan.DurationDays > 0 {
		expiredAt = now + int64(plan.DurationDays)*86400
	}

	updates := map[string]interface{}{
		"status":     model.SubStatusActive,
		"started_at": now,
		"expired_at": expiredAt,
	}

	// Apply group change
	if plan.Type == model.PlanTypeGroup || plan.Type == model.PlanTypeCombo {
		if plan.GroupName != "" {
			var user model.User
			tx.Select("group").First(&user, sub.UserId)
			updates["original_group"] = user.Group
			if err := tx.Model(&model.User{}).Where("id = ?", sub.UserId).
				Update("group", plan.GroupName).Error; err != nil {
				return err
			}
		}
	}

	// Apply quota
	if plan.Type == model.PlanTypeQuota || plan.Type == model.PlanTypeCombo {
		if plan.Quota > 0 {
			if err := tx.Model(&model.User{}).Where("id = ?", sub.UserId).
				Update("quota", gorm.Expr("quota + ?", plan.Quota)).Error; err != nil {
				return err
			}
		}
	}

	return tx.Model(&model.UserSubscription{}).Where("id = ?", subId).Updates(updates).Error
}
