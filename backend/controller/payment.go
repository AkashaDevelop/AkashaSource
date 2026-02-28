package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	epay "github.com/liuscraft/epay-sdk-go"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PaymentCreateRequest struct {
	Amount float64 `json:"amount"`
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
	if provider == "epay" {
		payUrl, err := buildEpayUrl(order, req.Amount)
		if err != nil {
			common.Fail(c, common.CodeParamError, err.Error())
			return
		}
		order.PayUrl = payUrl
		common.DB.Model(&order).Update("pay_url", payUrl)
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
	// DEBUG: log raw request info
	bodyBytes, _ := io.ReadAll(c.Request.Body)
	log.Printf("[PaymentNotify] method=%s url=%s query=%s body=%s headers=%v",
		c.Request.Method,
		c.Request.URL.String(),
		c.Request.URL.RawQuery,
		string(bodyBytes),
		c.Request.Header,
	)

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

	log.Printf("[PaymentNotify] no epay sign found, falling through to JSON parse. query keys: %v", c.Request.URL.Query())

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Printf("[PaymentNotify] JSON parse failed: %v", err)
		common.Fail(c, common.CodeParamError, "参数解析失败")
		return
	}

	secret := common.OptionMap[model.OptionKeyPaymentNotifySecret]
	if secret != "" {
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
	}

	orderId, _ := strconv.Atoi(toString(payload["order_id"]))
	status := strings.ToLower(toString(payload["status"]))
	amount, _ := strconv.ParseFloat(toString(payload["amount"]), 64)
	if orderId == 0 || amount <= 0 {
		common.Fail(c, common.CodeParamError, "订单参数错误")
		return
	}

	var order model.PaymentOrder
	if err := common.DB.First(&order, orderId).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "订单不存在")
		return
	}
	if order.Status == model.PaymentStatusPaid {
		common.OKMsg(c, "已处理", nil)
		return
	}

	if status != "paid" {
		order.Status = model.PaymentStatusFailed
		common.DB.Model(&order).Updates(map[string]interface{}{
			"status":      order.Status,
			"notify_data": toJSON(payload),
		})
		common.OKMsg(c, "已记录", nil)
		return
	}

	quotaAdd := int64(amount * 500000)
	err := common.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.PaymentOrder{}).Where("id = ?", order.Id).Updates(map[string]interface{}{
			"status":      model.PaymentStatusPaid,
			"paid_at":     time.Now().Unix(),
			"notify_data": toJSON(payload),
		}).Error; err != nil {
			return err
		}
		if order.OrderType == "subscription" {
			return ActivateSubscription(tx, order.RefId)
		}
		if err := tx.Model(&model.User{}).Where("id = ?", order.UserId).
			Update("quota", gorm.Expr("quota + ?", quotaAdd)).Error; err != nil {
			return err
		}
		service.EnqueueLog(model.Log{
			UserId:    order.UserId,
			CreatedAt: time.Now().Unix(),
			Type:      model.LogTypeTopup,
			Content:   "充值成功",
			Quota:     quotaAdd,
			ModelName: "system",
		})
		return nil
	})
	if err != nil {
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
		order.Status = model.PaymentStatusFailed
		common.DB.Model(&order).Updates(map[string]interface{}{
			"status":      order.Status,
			"notify_data": toJSON(params),
		})
		c.String(200, "success")
		return
	}

	err := common.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.PaymentOrder{}).Where("id = ?", order.Id).Updates(map[string]interface{}{
			"status":      model.PaymentStatusPaid,
			"paid_at":     time.Now().Unix(),
			"notify_data": toJSON(params),
		}).Error; err != nil {
			return err
		}
		if order.OrderType == "subscription" {
			return ActivateSubscription(tx, order.RefId)
		}
		quotaAdd := int64(amount * 500000)
		if err := tx.Model(&model.User{}).Where("id = ?", order.UserId).
			Update("quota", gorm.Expr("quota + ?", quotaAdd)).Error; err != nil {
			return err
		}
		service.EnqueueLog(model.Log{
			UserId:    order.UserId,
			CreatedAt: time.Now().Unix(),
			Type:      model.LogTypeTopup,
			Content:   "易支付充值成功",
			Quota:     quotaAdd,
			ModelName: "system",
		})
		return nil
	})
	if err != nil {
		c.String(200, "success")
		return
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
