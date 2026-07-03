package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service"
	paymentservice "STfreApi/service/payment"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	epay "github.com/liuscraft/epay-sdk-go"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PaymentCreateRequest struct {
	Amount    float64 `json:"amount"`
	ProductId string  `json:"product_id"` // Creem 多产品时指定
}

func CreatePayment(c *gin.Context) {
	var req PaymentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数解析失败")
		return
	}
	if req.Amount <= 0 {
		common.Fail(c, common.CodeParamError, "充值金额必须大于0")
		return
	}
	// ～防止超大金额溢出或污染统计，上限 10000 USD 够用了～
	if req.Amount > 10000 {
		common.Fail(c, common.CodeParamError, "充值金额不能超过 10000")
		return
	}
	if common.OptionMap[model.OptionKeyEnableTopup] != "true" {
		common.Fail(c, common.CodeForbidden, "充值功能未开启")
		return
	}

	minTopupStr := common.OptionMap[model.OptionKeyMinTopup]
	if minTopupStr != "" {
		if minTopup, err := strconv.ParseFloat(minTopupStr, 64); err == nil {
			if req.Amount < minTopup {
				common.Failf(c, common.CodeParamError, "充值金额不能低于 %.2f", minTopup)
				return
			}
		}
	}

	userId := c.GetInt("id")
	provider := common.OptionMap[model.OptionKeyPaymentProvider]
	// ～/user/stripe/pay 或 /user/creem/pay 路由会通过 force_provider 强制指定渠道～
	if fp, ok := c.Get("force_provider"); ok {
		provider = fp.(string)
	}

	// ～provider 没配置，或者填了奇怪的值，拒绝掉，免得创建一堆永远 pending 的僵尸订单～
	if provider != "epay" && provider != "stripe" && provider != "creem" {
		common.Fail(c, common.CodeNotImplemented, "当前未配置有效的支付渠道，请联系管理员")
		return
	}

	order := model.PaymentOrder{
		UserId:    userId,
		Amount:    req.Amount,
		Status:    model.PaymentStatusPending,
		Provider:  provider,
		CreatedAt: time.Now().Unix(),
	}
	if err := common.DB.Create(&order).Error; err != nil {
		common.Fail(c, common.CodeServerError, "创建订单失败")
		return
	}

	switch provider {
	case "epay":
		payUrl, err := buildEpayUrl(order, req.Amount)
		if err != nil {
			common.DB.Delete(&order)
			common.Fail(c, common.CodeParamError, err.Error())
			return
		}
		order.PayUrl = payUrl
		common.DB.Model(&order).Update("pay_url", payUrl)

	case "stripe":
		// ～美元/分换算：1 USD = 100 cents；OptionMap 里存的 amount 是美元小数～
		secretKey := common.OptionMap[model.OptionKeyStripeSecretKey]
		currency := common.OptionMap[model.OptionKeyStripeCurrency]
		successUrl := common.OptionMap[model.OptionKeyStripeSuccessUrl]
		cancelUrl := common.OptionMap[model.OptionKeyStripeCancelUrl]
		systemUrl := common.OptionMap[model.OptionKeySystemUrl]
		if successUrl == "" {
			successUrl = systemUrl + "/topup?stripe_success=1"
		}
		if cancelUrl == "" {
			cancelUrl = systemUrl + "/topup"
		}
		amountCents := int64(req.Amount * 100)
		result, err := paymentservice.CreateStripeCheckout(
			secretKey, currency, successUrl, cancelUrl,
			amountCents, order.Id, fmt.Sprintf("Akasha 账户充值 %.2f", req.Amount),
		)
		if err != nil {
			common.DB.Delete(&order)
			common.Fail(c, common.CodeParamError, err.Error())
			return
		}
		order.PayUrl = result.PayUrl
		order.TradeNo = result.SessionId
		common.DB.Model(&order).Updates(map[string]interface{}{
			"pay_url":  result.PayUrl,
			"trade_no": result.SessionId,
		})

	case "creem":
		apiKey := common.OptionMap[model.OptionKeyCreemApiKey]
		successUrl := common.OptionMap[model.OptionKeyCreemSuccessUrl]
		testMode := common.OptionMap[model.OptionKeyCreemTestMode] == "true"
		if successUrl == "" {
			successUrl = common.OptionMap[model.OptionKeySystemUrl] + "/topup?creem_success=1"
		}

		// ～先尝试多产品目录，没有配置再降级到旧的单一 product_id～
		productId := req.ProductId
		if productId == "" {
			productId = common.OptionMap[model.OptionKeyCreemProductId]
		}
		if productsJSON := common.OptionMap[model.OptionKeyCreemProducts]; productsJSON != "" && req.ProductId != "" {
			// 从产品列表里找对应的产品，更新订单金额
			var products []struct {
				ProductId string  `json:"product_id"`
				Price     float64 `json:"price"`
				Quota     int64   `json:"quota"`
			}
			if err := json.Unmarshal([]byte(productsJSON), &products); err == nil {
				for _, p := range products {
					if p.ProductId == req.ProductId {
						// 用产品定义的 quota 直接更新订单
						common.DB.Model(&order).Update("quota_added", p.Quota)
						break
					}
				}
			}
			productId = req.ProductId
		}
		if productId == "" {
			common.DB.Delete(&order)
			common.Fail(c, common.CodeParamError, "未指定 Creem 产品 ID，请在设置中配置 creem_products 或 creem_product_id")
			return
		}
		result, err := paymentservice.CreateCreemCheckout(
			apiKey, productId, successUrl,
			fmt.Sprintf("order_%d", order.Id),
			testMode,
		)
		if err != nil {
			common.DB.Delete(&order)
			common.Fail(c, common.CodeParamError, err.Error())
			return
		}
		order.PayUrl = result.PayUrl
		order.TradeNo = result.CheckoutId
		common.DB.Model(&order).Updates(map[string]interface{}{
			"pay_url":  result.PayUrl,
			"trade_no": result.CheckoutId,
		})
	}

	common.OK(c, order)
}

func ListPayments(c *gin.Context) {
	userId := c.GetInt("id")
	var orders []model.PaymentOrder
	if err := common.DB.Where("user_id = ?", userId).Order("id desc").Find(&orders).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取订单失败")
		return
	}
	common.OK(c, orders)
}

func PaymentNotify(c *gin.Context) {
	// Epay/CodePay/VPay callbacks: GET with query params (sign is always present)
	if c.Query("sign") != "" {
		handleEpayNotify(c)
		return
	}
	// POST form-encoded variant
	if err := c.Request.ParseForm(); err == nil {
		if c.Request.PostFormValue("sign") != "" {
			handleEpayNotify(c)
			return
		}
	}

	// 通用 JSON 回调（非 epay 渠道）
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		common.Fail(c, common.CodeParamError, "参数解析失败")
		return
	}

	// ～secret 必须配置，否则任何人都能伪造回调，直接拒绝～
	secret := common.OptionMap[model.OptionKeyPaymentNotifySecret]
	if secret == "" {
		log.Printf("[PaymentNotify] 未配置 payment_notify_secret，拒绝 JSON 回调")
		common.Fail(c, common.CodeForbidden, "支付回调未启用")
		return
	}
	orderId := toString(payload["order_id"])
	amount := toString(payload["amount"])
	sign := toString(payload["sign"])
	raw := orderId + amount + secret
	hash := md5.Sum([]byte(raw))
	expect := hex.EncodeToString(hash[:])
	if sign != expect {
		common.Fail(c, common.CodeParamError, "签名校验失败")
		return
	}

	orderIdInt, _ := strconv.Atoi(orderId)
	status := strings.ToLower(toString(payload["status"]))
	if orderIdInt == 0 {
		common.Fail(c, common.CodeParamError, "订单参数错误")
		return
	}

	var order model.PaymentOrder
	if err := common.DB.First(&order, orderIdInt).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "订单不存在")
		return
	}

	if status != "paid" {
		common.DB.Model(&order).Updates(map[string]interface{}{
			"status":      model.PaymentStatusFailed,
			"notify_data": toJSON(payload),
		})
		common.OKMsg(c, "已记录", nil)
		return
	}

	if order.OrderType == "subscription" {
		// 订阅订单走原有事务逻辑
		err := common.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&model.PaymentOrder{}).Where("id = ?", order.Id).Updates(map[string]interface{}{
				"status":      model.PaymentStatusPaid,
				"paid_at":     time.Now().Unix(),
				"notify_data": toJSON(payload),
			}).Error; err != nil {
				return err
			}
			return ActivateSubscription(tx, order.RefId)
		})
		if err != nil {
			common.Fail(c, common.CodeServerError, "订阅激活失败")
			return
		}
		common.OKMsg(c, "处理成功", nil)
		return
	}

	// ～改用统一入账函数，金额从数据库读取，带幂等锁保护～
	if err := paymentservice.RechargeOrder(order.Id, toJSON(payload)); err != nil {
		common.Fail(c, common.CodeServerError, "订单更新失败")
		return
	}
	common.OKMsg(c, "处理成功", nil)
}

func buildEpayUrl(order model.PaymentOrder, amount float64) (string, error) {
	api := strings.TrimSpace(common.OptionMap[model.OptionKeyEpayApiUrl])
	pid := strings.TrimSpace(common.OptionMap[model.OptionKeyEpayPid])
	key := strings.TrimSpace(common.OptionMap[model.OptionKeyEpayKey])
	payType := strings.TrimSpace(common.OptionMap[model.OptionKeyEpayType])
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
		"out_trade_no": strconv.Itoa(order.Id),
		"notify_url":   notifyUrl,
		"return_url":   returnUrl,
		"name":         "Akasha 账户充值",
		"money":        fmt.Sprintf("%.2f", amount),
	}
	if payType != "" {
		params["type"] = payType
	}

	signed := epay.NewSigner(key).SignWithParams(params)
	query := epay.BuildURLQuery(signed)
	return strings.TrimRight(api, "/") + "/pay/submit.php?" + query, nil
}

func handleEpayNotify(c *gin.Context) {
	params := map[string]string{}
	// Collect from URL query string (GET callbacks)
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}
	// Merge POST form body (POST callbacks)
	if err := c.Request.ParseForm(); err == nil {
		for k, v := range c.Request.PostForm {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	}
	key := strings.TrimSpace(common.OptionMap[model.OptionKeyEpayKey])
	sign := params["sign"]
	if sign == "" || key == "" {
		c.String(400, "fail")
		return
	}
	if !epay.NewSigner(key).Verify(params, sign) {
		c.String(400, "fail")
		return
	}
	delete(params, "sign")
	delete(params, "sign_type")

	orderId, _ := strconv.Atoi(params["out_trade_no"])
	amount, _ := strconv.ParseFloat(params["money"], 64)
	status := strings.ToLower(params["trade_status"])
	if status == "" {
		status = strings.ToLower(params["status"])
	}
	if orderId == 0 || amount <= 0 {
		c.String(400, "fail")
		return
	}

	var order model.PaymentOrder
	if err := common.DB.First(&order, orderId).Error; err != nil {
		c.String(200, "success")
		return
	}
	if order.Status == model.PaymentStatusPaid {
		c.String(200, "success")
		return
	}
	if status != "trade_success" && status != "success" && status != "paid" {
		common.DB.Model(&order).Updates(map[string]interface{}{
			"status":      model.PaymentStatusFailed,
			"notify_data": toJSON(params),
		})
		c.String(200, "success")
		return
	}

	if order.OrderType == "subscription" {
		err := common.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&model.PaymentOrder{}).Where("id = ?", order.Id).Updates(map[string]interface{}{
				"status":      model.PaymentStatusPaid,
				"paid_at":     time.Now().Unix(),
				"notify_data": toJSON(params),
			}).Error; err != nil {
				return err
			}
			return ActivateSubscription(tx, order.RefId)
		})
		if err != nil {
			log.Printf("[handleEpayNotify] 订阅激活失败 orderId=%d: %v", order.Id, err)
		}
		c.String(200, "success")
		return
	}

	// ～改用统一入账函数，金额从数据库读取，自带幂等锁保护～
	if err := paymentservice.RechargeOrder(order.Id, toJSON(params)); err != nil {
		log.Printf("[handleEpayNotify] 入账失败 orderId=%d: %v", order.Id, err)
	}
	c.String(200, "success")
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', 2, 64)
	case int:
		return strconv.Itoa(t)
	default:
		return ""
	}
}

func toJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

// GetUserOrderHistory ～用户自己查最近 90 天的充值记录，分页翻页～
func GetUserOrderHistory(c *gin.Context) {
	userId := c.GetInt("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	orders, total, err := model.GetUserPaymentOrders(userId, page, pageSize)
	if err != nil {
		common.Fail(c, common.CodeServerError, "查询失败")
		return
	}
	common.OK(c, gin.H{"data": orders, "total": total, "page": page, "size": pageSize})
}

// AdminListOrders ～管理员翻全量订单，支持按交易号/ID关键词搜索～
func AdminListOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	keyword := c.Query("keyword")
	orders, total, err := model.GetAllPaymentOrders(page, pageSize, keyword)
	if err != nil {
		common.Fail(c, common.CodeServerError, "查询失败")
		return
	}
	common.OK(c, gin.H{"data": orders, "total": total, "page": page, "size": pageSize})
}

// AdminGetOrderStats ～管理员统计各状态订单数和总金额～
func AdminGetOrderStats(c *gin.Context) {
	common.OK(c, model.GetPaymentOrderStats())
}

// AdminManualCompleteOrder ～管理员手动补单：对 pending 订单强制入账～
func AdminManualCompleteOrder(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.Fail(c, common.CodeParamError, "无效的订单 ID")
		return
	}
	order, err := model.ManualCompleteOrder(id)
	if err != nil {
		if err == gorm.ErrInvalidData {
			common.Fail(c, common.CodeParamError, "只能对待支付的订单进行补单操作")
			return
		}
		common.Fail(c, common.CodeNotFound, "订单不存在")
		return
	}
	if err := paymentservice.RechargeOrder(order.Id, `{"source":"admin_manual"}`); err != nil {
		common.Fail(c, common.CodeServerError, "补单失败: "+err.Error())
		return
	}
	service.EnqueueLog(model.Log{
		UserId:    order.UserId,
		CreatedAt: time.Now().Unix(),
		Type:      model.LogTypeSystem,
		Content:   fmt.Sprintf("管理员手动补单订单 #%d", order.Id),
		ModelName: "system",
	})
	common.OKMsg(c, "补单成功", nil)
}
