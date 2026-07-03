package model

import (
	"STfreApi/common"
	"time"

	"gorm.io/gorm"
)

// PaymentOrder ～记录每一笔充值订单的小本本，从下单到入账全程跟踪哦～
type PaymentOrder struct {
	Id          int     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId      int     `json:"user_id" gorm:"index"`
	Amount      float64 `json:"amount"`                                   // 实际支付金额（货币单位）
	QuotaAdded  int64   `json:"quota_added" gorm:"default:0"`             // 入账的 quota 数，入账后回填
	Status      int     `json:"status" gorm:"default:0;index"`
	Provider    string  `json:"provider" gorm:"type:varchar(30)"`
	PayUrl      string  `json:"pay_url"`
	TradeNo     string  `json:"trade_no" gorm:"uniqueIndex;type:varchar(255)"` // Stripe SessionId / Creem CheckoutId
	NotifyData  string  `json:"notify_data" gorm:"type:text"`
	CreatedAt   int64   `json:"created_at" gorm:"index"`
	CompletedAt int64   `json:"completed_at" gorm:"default:0"` // 入账完成时间
	OrderType   string  `json:"order_type" gorm:"type:varchar(20);default:'topup'"`
	RefId       int     `json:"ref_id" gorm:"default:0"`
}

const (
	PaymentStatusPending = 0
	PaymentStatusPaid    = 1
	PaymentStatusFailed  = 2
	PaymentStatusExpired = 3 // Stripe checkout.session.expired
)

// GetOrderByTradeNo ～按交易号找到对应的订单小票～
func GetOrderByTradeNo(tradeNo string) (*PaymentOrder, error) {
	var order PaymentOrder
	err := common.DB.Where("trade_no = ?", tradeNo).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetUserPaymentOrders ～翻出某个用户最近 90 天的充值记录，分页显示～
func GetUserPaymentOrders(userId, page, pageSize int) ([]PaymentOrder, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	cutoff := time.Now().AddDate(0, 0, -90).Unix()
	var orders []PaymentOrder
	var total int64
	db := common.DB.Model(&PaymentOrder{}).
		Where("user_id = ? AND created_at >= ?", userId, cutoff)
	db.Count(&total)
	err := db.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&orders).Error
	return orders, total, err
}

// GetAllPaymentOrders ～管理员视角，全量订单分页+关键词搜索～
func GetAllPaymentOrders(page, pageSize int, keyword string) ([]PaymentOrder, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	var orders []PaymentOrder
	var total int64
	db := common.DB.Model(&PaymentOrder{})
	if keyword != "" {
		db = db.Where("trade_no LIKE ? OR id = ?", "%"+keyword+"%", keyword)
	}
	db.Count(&total)
	err := db.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&orders).Error
	return orders, total, err
}

// GetPaymentOrderStats ～给管理员汇总一下各状态的订单数量和总金额～
func GetPaymentOrderStats() map[string]interface{} {
	type statusCount struct {
		Status int     `json:"status"`
		Count  int64   `json:"count"`
		Total  float64 `json:"total"`
	}
	var rows []statusCount
	common.DB.Model(&PaymentOrder{}).
		Select("status, COUNT(*) as count, SUM(amount) as total").
		Group("status").
		Scan(&rows)

	result := map[string]interface{}{
		"pending": map[string]interface{}{"count": int64(0), "total": 0.0},
		"paid":    map[string]interface{}{"count": int64(0), "total": 0.0},
		"failed":  map[string]interface{}{"count": int64(0), "total": 0.0},
		"expired": map[string]interface{}{"count": int64(0), "total": 0.0},
	}
	labels := map[int]string{0: "pending", 1: "paid", 2: "failed", 3: "expired"}
	for _, r := range rows {
		if label, ok := labels[r.Status]; ok {
			result[label] = map[string]interface{}{"count": r.Count, "total": r.Total}
		}
	}
	return result
}

// ManualCompleteOrder ～管理员补单：对 pending 状态的订单手动入账，幂等安全～
func ManualCompleteOrder(orderId int) (*PaymentOrder, error) {
	var order PaymentOrder
	if err := common.DB.First(&order, orderId).Error; err != nil {
		return nil, err
	}
	if order.Status == PaymentStatusPaid {
		return &order, nil // 已成功，幂等返回
	}
	if order.Status != PaymentStatusPending {
		return nil, gorm.ErrInvalidData // 只允许从 pending 补单
	}
	return &order, nil // 返回 order，由 service.RechargeOrder 完成入账
}
