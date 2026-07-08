package payment

import (
	"fmt"

	stripe "github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/client"
)

// StripeCheckoutResult ～Stripe 发回来的付款跳转小票～
type StripeCheckoutResult struct {
	SessionId string
	PayUrl    string
}

// CreateStripeCheckout ～让 Stripe 开一个一次性付款会话，返回跳转链接～
func CreateStripeCheckout(secretKey, currency, successUrl, cancelUrl string, amountCents int64, tradeNo string, description string) (*StripeCheckoutResult, error) {
	if secretKey == "" {
		return nil, fmt.Errorf("Stripe Secret Key 未配置")
	}

	// ～用 client.API 实例而不是包级全局 stripe.Key，避免并发请求间互相覆盖密钥喵～
	sc := &client.API{}
	sc.Init(secretKey, nil)

	if currency == "" {
		currency = "usd"
	}
	if successUrl == "" {
		return nil, fmt.Errorf("Stripe 成功跳转地址未配置")
	}
	if cancelUrl == "" {
		cancelUrl = successUrl
	}

	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(successUrl),
		CancelURL:  stripe.String(cancelUrl),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Quantity: stripe.Int64(1),
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(description),
					},
					UnitAmount: stripe.Int64(amountCents),
				},
			},
		},
		// ～把咱们自己算好的交易号藏在 ClientReferenceID 里，Webhook 收到后能对上号，
		// 和 trade_no 语义完全统一，不用再靠 SessionId/OrderId 两套键值混着查啦～
		ClientReferenceID: stripe.String(tradeNo),
	}

	sess, err := sc.CheckoutSessions.New(params)
	if err != nil {
		return nil, fmt.Errorf("创建 Stripe Checkout Session 失败: %w", err)
	}

	return &StripeCheckoutResult{
		SessionId: sess.ID,
		PayUrl:    sess.URL,
	}, nil
}

// CreateStripeSubscriptionCheckout ～让 Stripe 开一个订阅制付款会话，引用后台预配置好的 recurring Price ID～
func CreateStripeSubscriptionCheckout(secretKey, successUrl, cancelUrl, priceId, tradeNo string) (*StripeCheckoutResult, error) {
	if secretKey == "" {
		return nil, fmt.Errorf("Stripe Secret Key 未配置")
	}
	if priceId == "" {
		return nil, fmt.Errorf("Stripe Price ID 未配置")
	}

	sc := &client.API{}
	sc.Init(secretKey, nil)

	if successUrl == "" {
		return nil, fmt.Errorf("Stripe 成功跳转地址未配置")
	}
	if cancelUrl == "" {
		cancelUrl = successUrl
	}

	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(successUrl),
		CancelURL:  stripe.String(cancelUrl),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceId),
				Quantity: stripe.Int64(1),
			},
		},
		ClientReferenceID: stripe.String(tradeNo),
	}

	sess, err := sc.CheckoutSessions.New(params)
	if err != nil {
		return nil, fmt.Errorf("创建 Stripe 订阅 Checkout Session 失败: %w", err)
	}

	return &StripeCheckoutResult{
		SessionId: sess.ID,
		PayUrl:    sess.URL,
	}, nil
}
