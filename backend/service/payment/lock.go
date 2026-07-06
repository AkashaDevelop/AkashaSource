package payment

import "sync"

// ～每张订单都有自己的小锁，防止 Webhook 并发重入导致重复入账哦～
// 用带引用计数的锁而不是让 sync.Map 里的 *sync.Mutex 永久堆积——
// 订单量一大，进程常驻内存就会跟着无限涨，这里在锁归零时把 entry 一并摘掉～
type refCountedMutex struct {
	mu       sync.Mutex
	refCount int
}

var (
	orderLocks   = map[string]*refCountedMutex{}
	orderLocksMu sync.Mutex
)

// LockOrder 获取（或创建）指定 tradeNo 的锁并 Lock
func LockOrder(tradeNo string) {
	orderLocksMu.Lock()
	rcm, ok := orderLocks[tradeNo]
	if !ok {
		rcm = &refCountedMutex{}
		orderLocks[tradeNo] = rcm
	}
	rcm.refCount++
	orderLocksMu.Unlock()

	rcm.mu.Lock()
}

// UnlockOrder 解锁指定 tradeNo 的锁；引用计数归零时把 entry 从 map 里摘掉，不留内存尾巴
func UnlockOrder(tradeNo string) {
	orderLocksMu.Lock()
	rcm, ok := orderLocks[tradeNo]
	if !ok {
		orderLocksMu.Unlock()
		return
	}
	rcm.refCount--
	if rcm.refCount <= 0 {
		delete(orderLocks, tradeNo)
	}
	orderLocksMu.Unlock()

	rcm.mu.Unlock()
}
