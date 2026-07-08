package payment

import (
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
	"STfreApi/service"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// NewTradeNo ～下单前就把交易号算好，不依赖任何网关的返回值，
// 一次性 Create 订单，彻底不需要"占位符+事后 Update"这种两阶段写入啦～
func NewTradeNo() string {
	return "AK" + common.GetUUID()
}

// RechargeOrder ～统一入账函数，带订单级锁保证幂等安全，任何渠道的支付成功都走这里～
// expectedProvider 不为空时会校验订单归属渠道，防止伪造某渠道回调却指向另一渠道订单的攻击；传空字符串跳过校验。
func RechargeOrder(orderId int, expectedProvider string, notifyData string) error {
	// 先从数据库读最新状态（不能信 controller 层传来的缓存对象）
	var order model.PaymentOrder
	if err := common.DB.First(&order, orderId).Error; err != nil {
		return fmt.Errorf("订单 #%d 不存在: %w", orderId, err)
	}

	tradeNo := order.TradeNo
	if tradeNo == "" {
		tradeNo = fmt.Sprintf("order-%d", orderId)
	}

	LockOrder(tradeNo)
	defer UnlockOrder(tradeNo)

	// 重新读一次，确保拿到锁后的最新状态
	if err := common.DB.First(&order, orderId).Error; err != nil {
		return fmt.Errorf("订单 #%d 重读失败: %w", orderId, err)
	}

	// 幂等：已入账直接返回，不重复操作
	if order.Status == model.PaymentStatusPaid {
		return nil
	}

	// ～渠道来源校验：伪造的 Stripe 签名不该能让 Creem 订单入账，反之亦然喵～
	if expectedProvider != "" && order.Provider != expectedProvider {
		return fmt.Errorf("订单 #%d 归属渠道 %q 与回调渠道 %q 不符，拒绝入账", orderId, order.Provider, expectedProvider)
	}

	quotaAdd := int64(math.Round(order.Amount * common.QuotaPerUnit))

	err := common.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"status":       model.PaymentStatusPaid,
			"completed_at": time.Now().Unix(),
			"quota_added":  quotaAdd,
			"notify_data":  notifyData,
		}

		// ～WHERE 里带上 status=pending 做数据库层的防重复兜底，和上面的进程内锁形成双保险喵～
		result := tx.Model(&model.PaymentOrder{}).
			Where("id = ? AND status = ?", order.Id, model.PaymentStatusPending).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			// 已经被其他并发请求处理过了，当作幂等成功，不重复加账
			return nil
		}

		if order.OrderType == "subscription" {
			// ～订阅订单：给对应的 UserSubscription 做同样的 CAS 激活，不直接充值到钱包～
			return model.ActivateSubscription(tx, order.RefId)
		}

		if err := tx.Model(&model.User{}).Where("id = ?", order.UserId).
			Update("quota", gorm.Expr("quota + ?", quotaAdd)).Error; err != nil {
			return err
		}

		service.EnqueueLog(model.Log{
			UserId:    order.UserId,
			CreatedAt: time.Now().Unix(),
			Type:      model.LogTypeTopup,
			Content:   fmt.Sprintf("%s 充值成功 %.2f，到账 quota: %d", order.Provider, order.Amount, quotaAdd),
			Quota:     quotaAdd,
			ModelName: "system",
		})
		return nil
	})

	if err != nil {
		log.Printf("[RechargeOrder] 入账失败 orderId=%d tradeNo=%s: %v", orderId, tradeNo, err)
		return err
	}

	log.Printf("[RechargeOrder] 入账成功 orderId=%d tradeNo=%s quotaAdd=%d", orderId, tradeNo, quotaAdd)
	// ～充值订单入账成功后悄悄检查一下余额，不够的话给用户发个小提醒；订阅订单不涉及钱包余额，跳过～
	if order.OrderType != "subscription" {
		go checkAndNotifyLowBalance(order.UserId)
	}
	return nil
}

// RechargeOrderByTradeNo ～按交易号入账，Webhook 收到 tradeNo 时直接调用～
func RechargeOrderByTradeNo(tradeNo, expectedProvider, notifyData string) error {
	order, err := model.GetOrderByTradeNo(tradeNo)
	if err != nil {
		return fmt.Errorf("交易号 %s 对应的订单不存在: %w", tradeNo, err)
	}
	return RechargeOrder(order.Id, expectedProvider, notifyData)
}

// MarkOrderFailed ～把订单标记为失败，并记录原因～
func MarkOrderFailed(orderId int, reason string) {
	common.DB.Model(&model.PaymentOrder{}).Where("id = ? AND status = ?", orderId, model.PaymentStatusPending).
		Updates(map[string]interface{}{
			"status":      model.PaymentStatusFailed,
			"notify_data": reason,
		})
}

// MarkOrderExpiredByTradeNo ～按交易号把 pending 的订单标记为已过期（Stripe session expired）～
func MarkOrderExpiredByTradeNo(tradeNo string) {
	common.DB.Model(&model.PaymentOrder{}).
		Where("trade_no = ? AND status = ?", tradeNo, model.PaymentStatusPending).
		Updates(map[string]interface{}{
			"status":       model.PaymentStatusExpired,
			"completed_at": time.Now().Unix(),
		})
}

// checkAndNotifyLowBalance ～入账后检查余额，余额低于预警线就给用户发个消息提醒充值哦～
func checkAndNotifyLowBalance(userId int) {
	common.OptionLock.RLock()
	thresholdStr := common.OptionMap[model.OptionKeyLowBalanceThreshold]
	common.OptionLock.RUnlock()

	threshold := int64(common.QuotaPerUnit) // 默认 $1
	if thresholdStr != "" {
		if n, err := strconv.ParseInt(thresholdStr, 10, 64); err == nil && n > 0 {
			threshold = n
		}
	}

	var user model.User
	if err := common.DB.Select("id,email,quota,setting").First(&user, userId).Error; err != nil {
		return
	}
	if user.Quota >= threshold {
		return
	}

	setting := user.GetSetting()
	usd := float64(user.Quota) / common.QuotaPerUnit
	notify := dto.NewNotify(
		dto.NotifyTypeQuota,
		"余额不足提醒",
		fmt.Sprintf("您的账户余额仅剩 $%.4f（quota: %d），请及时充值以免影响使用。", usd, user.Quota),
		nil,
	)
	if err := service.NotifyUser(user.Id, user.Email, setting, notify); err != nil {
		log.Printf("[余额通知] 发送失败 userId=%d: %v", userId, err)
	}
}
