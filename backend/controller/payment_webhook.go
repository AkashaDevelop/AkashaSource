package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strings"

	"STfreApi/common"
	"STfreApi/model"
	paymentservice "STfreApi/service/payment"

	stripe "github.com/stripe/stripe-go/v86"
	stripewebhook "github.com/stripe/stripe-go/v86/webhook"
	"github.com/gin-gonic/gin"
)

// StripeWebhook ～Stripe 发来的回调，验签后分事件类型处理，四种支付状态全覆盖～
func StripeWebhook(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "fail")
		return
	}

	webhookSecret := common.OptionMap[model.OptionKeyStripeWebhookSecret]
	if webhookSecret == "" {
		log.Printf("[StripeWebhook] 未配置 stripe_webhook_secret，拒绝请求")
		c.String(http.StatusBadRequest, "fail")
		return
	}

	sigHeader := c.GetHeader("Stripe-Signature")
	event, err := stripewebhook.ConstructEventWithOptions(rawBody, sigHeader, webhookSecret,
		stripewebhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true})
	if err != nil {
		log.Printf("[StripeWebhook] 签名校验失败: %v", err)
		c.String(http.StatusBadRequest, "fail")
		return
	}

	notifyData := fmt.Sprintf(`{"event_type":%q,"event_id":%q}`, string(event.Type), event.ID)

	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted:
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			log.Printf("[StripeWebhook] 解析 session 失败: %v", err)
			c.String(http.StatusOK, "ok")
			return
		}
		if sess.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid {
			// ～client_reference_id 就是咱们下单时算好的 trade_no，和其他事件分支保持同一套查找方式喵～
			tradeNo := sess.ClientReferenceID
			if tradeNo == "" {
				tradeNo = sess.ID
			}
			if err := paymentservice.RechargeOrderByTradeNo(tradeNo, "stripe", notifyData); err != nil {
				log.Printf("[StripeWebhook] checkout.completed 入账失败: %v", err)
			}
		}

	case stripe.EventTypeCheckoutSessionAsyncPaymentSucceeded:
		if refId := event.GetObjectValue("client_reference_id"); refId != "" {
			if err := paymentservice.RechargeOrderByTradeNo(refId, "stripe", notifyData); err != nil {
				log.Printf("[StripeWebhook] async_payment_succeeded 入账失败 tradeNo=%s: %v", refId, err)
			}
			break
		}
		if sessionId := event.GetObjectValue("id"); sessionId != "" {
			if err := paymentservice.RechargeOrderByTradeNo(sessionId, "stripe", notifyData); err != nil {
				log.Printf("[StripeWebhook] async_payment_succeeded 入账失败 sessionId=%s: %v", sessionId, err)
			}
		}

	case stripe.EventTypeCheckoutSessionAsyncPaymentFailed:
		if refId := event.GetObjectValue("client_reference_id"); refId != "" {
			log.Printf("[StripeWebhook] 异步支付失败 tradeNo=%s", refId)
			order, _ := model.GetOrderByTradeNo(refId)
			if order != nil {
				paymentservice.MarkOrderFailed(order.Id, notifyData)
			}
			break
		}
		if sessionId := event.GetObjectValue("id"); sessionId != "" {
			log.Printf("[StripeWebhook] 异步支付失败 sessionId=%s", sessionId)
			order, _ := model.GetOrderByTradeNo(sessionId)
			if order != nil {
				paymentservice.MarkOrderFailed(order.Id, notifyData)
			}
		}

	case stripe.EventTypeCheckoutSessionExpired:
		if refId := event.GetObjectValue("client_reference_id"); refId != "" {
			paymentservice.MarkOrderExpiredByTradeNo(refId)
			log.Printf("[StripeWebhook] checkout 过期 tradeNo=%s", refId)
			break
		}
		if sessionId := event.GetObjectValue("id"); sessionId != "" {
			paymentservice.MarkOrderExpiredByTradeNo(sessionId)
			log.Printf("[StripeWebhook] checkout 过期 sessionId=%s", sessionId)
		}

	default:
		log.Printf("[StripeWebhook] 忽略事件类型: %s", event.Type)
	}

	c.String(http.StatusOK, "ok")
}

func CreemWebhook(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "fail")
		return
	}

	webhookSecret := common.OptionMap[model.OptionKeyCreemWebhookSecret]
	sig := c.GetHeader("creem-signature")
	// ～未配置 secret 就直接拒绝，不能放任何人白嫖～
	if webhookSecret == "" {
		log.Printf("[CreemWebhook] 未配置 creem_webhook_secret，拒绝请求")
		c.String(http.StatusBadRequest, "fail")
		return
	}
	if !paymentservice.VerifyCreemWebhook(rawBody, sig, webhookSecret) {
		log.Printf("[CreemWebhook] 签名校验失败")
		c.String(http.StatusBadRequest, "fail")
		return
	}

	var payload struct {
		EventType string          `json:"eventType"`
		Object    json.RawMessage `json:"object"`
	}
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		c.String(http.StatusBadRequest, "fail")
		return
	}
	if payload.EventType != "checkout.completed" {
		c.String(http.StatusOK, "ok")
		return
	}

	var obj struct {
		RequestId string `json:"request_id"`
		Id        string `json:"id"`
	}
	json.Unmarshal(payload.Object, &obj)

	// ～request_id 就是咱们下单时算好的 trade_no，一次查询直接命中，不用再靠 "order_" 前缀猜订单 ID 啦～
	tradeNo := obj.RequestId
	if tradeNo == "" {
		tradeNo = obj.Id
	}
	if tradeNo == "" {
		log.Printf("[CreemWebhook] 事件缺少 request_id/id，无法定位订单")
		c.String(http.StatusOK, "ok")
		return
	}

	if err := paymentservice.RechargeOrderByTradeNo(tradeNo, "creem", string(rawBody)); err != nil {
		log.Printf("[CreemWebhook] 入账失败 tradeNo=%s: %v", tradeNo, err)
	}
	c.String(http.StatusOK, "ok")
}

// 订阅 Epay 回调与返回：复用统一支付通知。
func SubscriptionEpayNotify(c *gin.Context) {
	PaymentNotify(c)
}

func SubscriptionEpayReturn(c *gin.Context) {
	PaymentNotify(c)
}

// GetTopUpInfo ～告诉前端当前哪些支付方式可用，顺手返回产品列表和待支付订单提示～
func GetTopUpInfo(c *gin.Context) {
	common.OptionLock.RLock()
	enableTopup := common.OptionMap[model.OptionKeyEnableTopup] == "true"
	provider := strings.TrimSpace(common.OptionMap[model.OptionKeyPaymentProvider])
	minTopup := strings.TrimSpace(common.OptionMap[model.OptionKeyMinTopup])
	price := strings.TrimSpace(common.OptionMap["price"])
	// Stripe 可用性：有 secret key + webhook secret 才算配置完整
	stripeEnabled := enableTopup && provider == "stripe" &&
		strings.TrimSpace(common.OptionMap[model.OptionKeyStripeSecretKey]) != "" &&
		strings.TrimSpace(common.OptionMap[model.OptionKeyStripeWebhookSecret]) != ""
	// Creem 可用性：有 api key 且有产品配置
	creemApiKey := strings.TrimSpace(common.OptionMap[model.OptionKeyCreemApiKey])
	creemProducts := strings.TrimSpace(common.OptionMap[model.OptionKeyCreemProducts])
	creemProductId := strings.TrimSpace(common.OptionMap[model.OptionKeyCreemProductId])
	creemEnabled := enableTopup && provider == "creem" && creemApiKey != "" &&
		(creemProducts != "" || creemProductId != "")
	// Epay 可用性
	epayEnabled := enableTopup && provider == "epay" &&
		strings.TrimSpace(common.OptionMap[model.OptionKeyEpayApiUrl]) != "" &&
		strings.TrimSpace(common.OptionMap[model.OptionKeyEpayPid]) != "" &&
		strings.TrimSpace(common.OptionMap[model.OptionKeyEpayKey]) != ""
	common.OptionLock.RUnlock()

	// 有未处理 pending 订单时给用户提示
	userId := c.GetInt("id")
	var pendingCount int64
	common.DB.Model(&model.PaymentOrder{}).
		Where("user_id = ? AND status = ?", userId, model.PaymentStatusPending).
		Count(&pendingCount)

	common.OK(c, gin.H{
		"enable_topup":    enableTopup,
		"provider":        provider,
		"min_topup":       minTopup,
		"display_price":   price,
		"stripe_enabled":  stripeEnabled,
		"creem_enabled":   creemEnabled,
		"epay_enabled":    epayEnabled,
		"creem_products":  creemProducts,
		"pending_orders":  pendingCount,
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
	// ～强制走 stripe 渠道，让 CreatePayment 的分发逻辑正确选中～
	c.Set("force_provider", "stripe")
	CreatePayment(c)
}

func RequestCreemPay(c *gin.Context) {
	c.Set("force_provider", "creem")
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
	common.OK(c, gin.H{"amount": req.Amount, "quota": int64(math.Round(req.Amount * common.QuotaPerUnit))})
}

func RequestStripeAmount(c *gin.Context) {
	RequestAmount(c)
}
