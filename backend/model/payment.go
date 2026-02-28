package model

type PaymentOrder struct {
	Id         int     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId     int     `json:"user_id" gorm:"index"`
	Amount     float64 `json:"amount"`
	Status     int     `json:"status" gorm:"default:0"`
	Provider   string  `json:"provider"`
	PayUrl     string  `json:"pay_url"`
	NotifyData string  `json:"notify_data" gorm:"type:text"`
	CreatedAt  int64   `json:"created_at" gorm:"index"`
	PaidAt     int64   `json:"paid_at"`
	OrderType  string  `json:"order_type" gorm:"type:varchar(20);default:'topup'"` // topup, subscription
	RefId      int     `json:"ref_id" gorm:"default:0"` // UserSubscription.Id for subscription orders
}

const (
	PaymentStatusPending = 0
	PaymentStatusPaid    = 1
	PaymentStatusFailed  = 2
)
