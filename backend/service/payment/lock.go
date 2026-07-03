package payment

import "sync"

// ～每张订单都有自己的小锁，防止 Webhook 并发重入导致重复入账哦～
var orderLocks sync.Map

// LockOrder 获取（或创建）指定 tradeNo 的锁并 Lock
func LockOrder(tradeNo string) {
	v, _ := orderLocks.LoadOrStore(tradeNo, &sync.Mutex{})
	v.(*sync.Mutex).Lock()
}

// UnlockOrder 解锁指定 tradeNo 的锁
func UnlockOrder(tradeNo string) {
	if v, ok := orderLocks.Load(tradeNo); ok {
		v.(*sync.Mutex).Unlock()
	}
}
