package payment

import (
	"fmt"

	stripe "github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/checkout/session"
)

// StripeCheckoutResult ～Stripe 发回来的付款跳转小票～
type StripeCheckoutResult struct {
	SessionId  string
	PayUrl     string
	OrderRefId string
}

// CreateStripeCheckout ～让 Stripe 开一个付款会话，返回跳转链接～
func CreateStripeCheckout(secretKey, currency, successUrl, cancelUrl string, amountCents int64, orderId int, description string) (*StripeCheckoutResult, error) {
	if secretKey == "" {
		return nil, fmt.Errorf("Stripe Secret Key 未配置")
	}

	stripe.Key = secretKey

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
		// ～把订单 ID 藏在 ClientReferenceID 里，Webhook 收到后能对上号～
		ClientReferenceID: stripe.String(fmt.Sprintf("%d", orderId)),
	}

	sess, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("创建 Stripe Checkout Session 失败: %w", err)
	}

	return &StripeCheckoutResult{
		SessionId:  sess.ID,
		PayUrl:     sess.URL,
		OrderRefId: fmt.Sprintf("%d", orderId),
	}, nil
}
