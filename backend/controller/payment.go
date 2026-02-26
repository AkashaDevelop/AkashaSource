package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PaymentCreateRequest struct {
	Amount float64 `json:"amount"`
}

func CreatePayment(c *gin.Context) {
	var req PaymentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数解析失败"})
		return
	}
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "充值金额必须大于0"})
		return
	}

	minTopupStr := common.OptionMap[model.OptionKeyMinTopup]
	if minTopupStr != "" {
		if minTopup, err := strconv.ParseFloat(minTopupStr, 64); err == nil {
			if req.Amount < minTopup {
				c.JSON(http.StatusBadRequest, gin.H{"error": "充值金额低于最低限制"})
				return
			}
		}
	}

	userId := c.GetInt("id")
	provider := common.OptionMap[model.OptionKeyPaymentProvider]
	payLink := common.OptionMap[model.OptionKeyTopupLink]

	order := model.PaymentOrder{
		UserId:    userId,
		Amount:    req.Amount,
		Status:    model.PaymentStatusPending,
		Provider:  provider,
		CreatedAt: time.Now().Unix(),
	}
	if err := common.DB.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建订单失败"})
		return
	}
	if provider == "epay" {
		payUrl, err := buildEpayUrl(order, req.Amount)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		order.PayUrl = payUrl
		common.DB.Model(&order).Update("pay_url", payUrl)
	} else if payLink != "" {
		payUrl := strings.ReplaceAll(payLink, "{order_id}", strconv.Itoa(order.Id))
		payUrl = strings.ReplaceAll(payUrl, "{amount}", strconv.FormatFloat(req.Amount, 'f', 2, 64))
		order.PayUrl = payUrl
		common.DB.Model(&order).Update("pay_url", payUrl)
	}

	c.JSON(http.StatusOK, gin.H{"data": order})
}

func ListPayments(c *gin.Context) {
	userId := c.GetInt("id")
	var orders []model.PaymentOrder
	if err := common.DB.Where("user_id = ?", userId).Order("id desc").Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取订单失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": orders})
}

func PaymentNotify(c *gin.Context) {
	if err := c.Request.ParseForm(); err == nil {
		if c.Request.FormValue("out_trade_no") != "" {
			handleEpayNotify(c)
			return
		}
	}

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数解析失败"})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "签名校验失败"})
			return
		}
	}

	orderId, _ := strconv.Atoi(toString(payload["order_id"]))
	status := strings.ToLower(toString(payload["status"]))
	amount, _ := strconv.ParseFloat(toString(payload["amount"]), 64)
	if orderId == 0 || amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "订单参数错误"})
		return
	}

	var order model.PaymentOrder
	if err := common.DB.First(&order, orderId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "订单不存在"})
		return
	}
	if order.Status == model.PaymentStatusPaid {
		c.JSON(http.StatusOK, gin.H{"message": "已处理"})
		return
	}

	if status != "paid" {
		order.Status = model.PaymentStatusFailed
		common.DB.Model(&order).Updates(map[string]interface{}{
			"status":      order.Status,
			"notify_data": toJSON(payload),
		})
		c.JSON(http.StatusOK, gin.H{"message": "已记录"})
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
		if err := tx.Model(&model.User{}).Where("id = ?", order.UserId).
			Update("quota", gorm.Expr("quota + ?", quotaAdd)).Error; err != nil {
			return err
		}
		logItem := model.Log{
			UserId:    order.UserId,
			CreatedAt: time.Now().Unix(),
			Type:      model.LogTypeTopup,
			Content:   "充值成功",
			Quota:     quotaAdd,
			ModelName: "system",
		}
		service.EnqueueLog(logItem)
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "订单更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "处理成功"})
}

func buildEpayUrl(order model.PaymentOrder, amount float64) (string, error) {
	api := strings.TrimSpace(common.OptionMap[model.OptionKeyEpayApiUrl])
	pid := strings.TrimSpace(common.OptionMap[model.OptionKeyEpayPid])
	key := strings.TrimSpace(common.OptionMap[model.OptionKeyEpayKey])
	payType := strings.TrimSpace(common.OptionMap[model.OptionKeyEpayType])
	notifyUrl := strings.TrimSpace(common.OptionMap[model.OptionKeyEpayNotifyUrl])
	returnUrl := strings.TrimSpace(common.OptionMap[model.OptionKeyEpayReturnUrl])

	if api == "" || pid == "" || key == "" {
		return "", errors.New("易支付配置不完整")
	}
	if payType == "" {
		payType = "alipay"
	}
	if notifyUrl == "" {
		notifyUrl = common.OptionMap[model.OptionKeySystemUrl] + "/api/payment/notify"
	}
	if returnUrl == "" {
		returnUrl = common.OptionMap[model.OptionKeySystemUrl]
	}

	params := map[string]string{
		"pid":          pid,
		"type":         payType,
		"out_trade_no": strconv.Itoa(order.Id),
		"notify_url":   notifyUrl,
		"return_url":   returnUrl,
		"name":         "Akasha 账户充值",
		"money":        strconv.FormatFloat(amount, 'f', 2, 64),
	}
	sign := epaySign(params, key)
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	values.Set("sign", sign)
	values.Set("sign_type", "MD5")

	api = strings.TrimRight(api, "/")
	return api + "/submit.php?" + values.Encode(), nil
}

func handleEpayNotify(c *gin.Context) {
	params := map[string]string{}
	for k, v := range c.Request.Form {
		if len(v) == 0 {
			continue
		}
		params[k] = v[0]
	}
	key := strings.TrimSpace(common.OptionMap[model.OptionKeyEpayKey])
	sign := params["sign"]
	delete(params, "sign")
	delete(params, "sign_type")
	if sign == "" || key == "" {
		c.String(http.StatusBadRequest, "fail")
		return
	}
	expect := epaySign(params, key)
	if !strings.EqualFold(sign, expect) {
		c.String(http.StatusBadRequest, "fail")
		return
	}

	orderId, _ := strconv.Atoi(params["out_trade_no"])
	amount, _ := strconv.ParseFloat(params["money"], 64)
	status := strings.ToLower(params["trade_status"])
	if status == "" {
		status = strings.ToLower(params["status"])
	}
	if orderId == 0 || amount <= 0 {
		c.String(http.StatusBadRequest, "fail")
		return
	}

	var order model.PaymentOrder
	if err := common.DB.First(&order, orderId).Error; err != nil {
		c.String(http.StatusOK, "success")
		return
	}
	if order.Status == model.PaymentStatusPaid {
		c.String(http.StatusOK, "success")
		return
	}
	if status != "trade_success" && status != "success" && status != "paid" {
		order.Status = model.PaymentStatusFailed
		common.DB.Model(&order).Updates(map[string]interface{}{
			"status":      order.Status,
			"notify_data": toJSON(params),
		})
		c.String(http.StatusOK, "success")
		return
	}

	quotaAdd := int64(amount * 500000)
	err := common.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.PaymentOrder{}).Where("id = ?", order.Id).Updates(map[string]interface{}{
			"status":      model.PaymentStatusPaid,
			"paid_at":     time.Now().Unix(),
			"notify_data": toJSON(params),
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).Where("id = ?", order.UserId).
			Update("quota", gorm.Expr("quota + ?", quotaAdd)).Error; err != nil {
			return err
		}
		logItem := model.Log{
			UserId:    order.UserId,
			CreatedAt: time.Now().Unix(),
			Type:      model.LogTypeTopup,
			Content:   "易支付充值成功",
			Quota:     quotaAdd,
			ModelName: "system",
		}
		service.EnqueueLog(logItem)
		return nil
	})
	if err != nil {
		c.String(http.StatusOK, "success")
		return
	}
	c.String(http.StatusOK, "success")
}

func epaySign(params map[string]string, key string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if params[k] == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	builder := strings.Builder{}
	for i, k := range keys {
		if i > 0 {
			builder.WriteString("&")
		}
		builder.WriteString(k)
		builder.WriteString("=")
		builder.WriteString(params[k])
	}
	builder.WriteString(key)
	hash := md5.Sum([]byte(builder.String()))
	return hex.EncodeToString(hash[:])
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
