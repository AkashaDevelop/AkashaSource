package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
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
			if err := paymentservice.RechargeOrderByTradeNo(sess.ID, notifyData); err != nil {
				log.Printf("[StripeWebhook] checkout.completed 入账失败: %v", err)
			}
		}

	case stripe.EventTypeCheckoutSessionAsyncPaymentSucceeded:
		// ～client_reference_id 存的是下单时的订单自增 Id（见 service/payment/stripe.go），
		// 不是 trade_no（trade_no 存的是 Stripe SessionId），两者不能混用查询喵～
		if refId := event.GetObjectValue("client_reference_id"); refId != "" {
			orderId, err := strconv.Atoi(refId)
			if err != nil {
				log.Printf("[StripeWebhook] async_payment_succeeded client_reference_id 非法: %s", refId)
				break
			}
			if err := paymentservice.RechargeOrder(orderId, notifyData); err != nil {
				log.Printf("[StripeWebhook] async_payment_succeeded 入账失败 orderId=%d: %v", orderId, err)
			}
			break
		}
		if sessionId := event.GetObjectValue("id"); sessionId != "" {
			if err := paymentservice.RechargeOrderByTradeNo(sessionId, notifyData); err != nil {
				log.Printf("[StripeWebhook] async_payment_succeeded 入账失败 sessionId=%s: %v", sessionId, err)
			}
		}

	case stripe.EventTypeCheckoutSessionAsyncPaymentFailed:
		if refId := event.GetObjectValue("client_reference_id"); refId != "" {
			orderId, err := strconv.Atoi(refId)
			if err != nil {
				log.Printf("[StripeWebhook] async_payment_failed client_reference_id 非法: %s", refId)
				break
			}
			log.Printf("[StripeWebhook] 异步支付失败 orderId=%d", orderId)
			paymentservice.MarkOrderFailed(orderId, notifyData)
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
			orderId, err := strconv.Atoi(refId)
			if err != nil {
				log.Printf("[StripeWebhook] session expired client_reference_id 非法: %s", refId)
				break
			}
			paymentservice.MarkOrderExpired(orderId)
			log.Printf("[StripeWebhook] checkout 过期 orderId=%d", orderId)
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

	var order model.PaymentOrder
	// ～我们把 order ID 藏在 request_id 里，格式是 "order_123"～
	orderId := strings.TrimPrefix(obj.RequestId, "order_")
	if err := common.DB.Where("id = ? AND provider = 'creem'", orderId).First(&order).Error; err != nil {
		if obj.Id != "" {
			common.DB.Where("trade_no = ?", obj.Id).First(&order)
		}
		if order.Id == 0 {
			log.Printf("[CreemWebhook] 订单未找到: request_id=%s checkout_id=%s", obj.RequestId, obj.Id)
			c.String(http.StatusOK, "ok")
			return
		}
	}

	handleOrderPaid(&order, string(rawBody))
	c.String(http.StatusOK, "ok")
}

// 订阅 Epay 回调与返回：复用统一支付通知。
func SubscriptionEpayNotify(c *gin.Context) {
	PaymentNotify(c)
}

func SubscriptionEpayReturn(c *gin.Context) {
	PaymentNotify(c)
}

// handleOrderPaid ～旧版入账，已迁移到 service/payment/recharge.go，这里仅保留用于 epay 订阅回调兼容，待彻底清理～
func handleOrderPaid(order *model.PaymentOrder, notifyData string) {
	if err := paymentservice.RechargeOrder(order.Id, notifyData); err != nil {
		log.Printf("[handleOrderPaid] 入账失败 orderId=%d: %v", order.Id, err)
	}
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
