package sanction

// ～宸汐处置执行层：内存缓存 + DB 持久化的读写中枢～
// 热路径（每个请求都要查）用内存索引保证 O(1)、零 DB 开销；
// 写入（玄鉴自动处置 / 管理员手动处置）落库后立即刷新内存缓存。
//
// 设计要点：
//   - 处置态持久化（区别于风险画像的内存态），支持"永久"处置、重启保留、前端可查可解除
//   - 三类内存索引分别按 IP / TokenID / UserID 组织，读多写少用 RWMutex
//   - 过期惰性判断 + 后台定时清理双保险

import (
	"log"
	"strconv"
	"sync"
	"time"

	"STfreApi/model"
)

// TokenSanctionView 单个 Token 的处置视图（热路径查询返回）
type TokenSanctionView struct {
	Disabled       bool    // suspend_token / disable_token 命中
	ThrottleFactor float64 // throttle 降速倍率，1.0=不限速
	RPMLimit       int     // rpm_limit 固定 RPM，0=不限制
	BillingFactor  float64 // token 级计费惩罚倍率，1.0=不惩罚
	Reason         string
}

type store struct {
	mu sync.RWMutex

	ipBans     map[string]*model.Sanction // ip → ban_ip
	tokenState map[int]*TokenSanctionView // tokenID → 聚合视图
	userBilling map[int]float64           // userID → 计费惩罚倍率
	userBans   map[int]bool               // userID → ban_user
}

var globalStore = &store{
	ipBans:      map[string]*model.Sanction{},
	tokenState:  map[int]*TokenSanctionView{},
	userBilling: map[int]float64{},
	userBans:    map[int]bool{},
}

// Init 启动时全量加载 + 起后台定时刷新/清理 goroutine
func Init() {
	if err := Reload(); err != nil {
		log.Printf("[宸汐处置] 初始化加载失败: %v", err)
	}
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			_ = model.CleanupExpiredSanctions()
			if err := Reload(); err != nil {
				log.Printf("[宸汐处置] 定时刷新失败: %v", err)
			}
		}
	}()
}

// Reload 全量从 DB 加载有效制裁重建内存索引
func Reload() error {
	list, err := model.GetActiveSanctions()
	if err != nil {
		return err
	}
	ipBans := map[string]*model.Sanction{}
	tokenState := map[int]*TokenSanctionView{}
	userBilling := map[int]float64{}
	userBans := map[int]bool{}

	for i := range list {
		s := list[i]
		switch s.TargetType {
		case model.SanctionTargetIP:
			if s.Action == model.SanctionBanIP {
				sc := s
				ipBans[s.TargetKey] = &sc
			}
		case model.SanctionTargetToken:
			id, err := strconv.Atoi(s.TargetKey)
			if err != nil {
				continue
			}
			v := tokenState[id]
			if v == nil {
				v = &TokenSanctionView{ThrottleFactor: 1.0, BillingFactor: 1.0}
				tokenState[id] = v
			}
			applyTokenAction(v, s)
		case model.SanctionTargetUser:
			id, err := strconv.Atoi(s.TargetKey)
			if err != nil {
				continue
			}
			switch s.Action {
			case model.SanctionBanUser:
				userBans[id] = true
			case model.SanctionBillingPenalty:
				f := s.Factor
				if f <= 1.0 {
					f = 1.0
				}
				userBilling[id] = f
			}
		}
	}

	globalStore.mu.Lock()
	globalStore.ipBans = ipBans
	globalStore.tokenState = tokenState
	globalStore.userBilling = userBilling
	globalStore.userBans = userBans
	globalStore.mu.Unlock()
	return nil
}

// applyTokenAction 把一条 token 维度的制裁并入聚合视图
func applyTokenAction(v *TokenSanctionView, s model.Sanction) {
	switch s.Action {
	case model.SanctionSuspendToken, model.SanctionDisableToken:
		v.Disabled = true
		if v.Reason == "" {
			v.Reason = s.Reason
		}
	case model.SanctionThrottle:
		f := s.Factor
		if f <= 0 || f >= 1.0 {
			return
		}
		// 多条 throttle 取最狠的（最小倍率）
		if f < v.ThrottleFactor {
			v.ThrottleFactor = f
		}
	case model.SanctionRPMLimit:
		rpm := int(s.Factor)
		if rpm <= 0 {
			return
		}
		// 多条 rpm_limit 取最严的（最小 RPM）
		if v.RPMLimit == 0 || rpm < v.RPMLimit {
			v.RPMLimit = rpm
		}
	case model.SanctionBillingPenalty:
		f := s.Factor
		if f <= 1.0 {
			return
		}
		if f > v.BillingFactor {
			v.BillingFactor = f
		}
	}
}

// ReloadNow 处置变更后主动刷新（供 Enforce / 管理员接口调用）
func ReloadNow() {
	if err := Reload(); err != nil {
		log.Printf("[宸汐处置] 主动刷新失败: %v", err)
	}
}

// ─── 热路径查询（纯内存 O(1）───────────────────────────────────────────────

// IsIPBanned 检查 IP 是否被封禁
func IsIPBanned(ip string) (bool, string) {
	globalStore.mu.RLock()
	defer globalStore.mu.RUnlock()
	if s, ok := globalStore.ipBans[ip]; ok {
		return true, s.Reason
	}
	return false, ""
}

// GetTokenSanction 返回 Token 的聚合处置视图（无制裁返回 nil）
func GetTokenSanction(tokenID int) *TokenSanctionView {
	globalStore.mu.RLock()
	defer globalStore.mu.RUnlock()
	return globalStore.tokenState[tokenID]
}

// GetBillingMultiplier 返回指定目标的计费惩罚倍率（无则 1.0）
// targetType 取 model.SanctionTargetUser / SanctionTargetToken
func GetBillingMultiplier(targetType string, id int) float64 {
	globalStore.mu.RLock()
	defer globalStore.mu.RUnlock()
	switch targetType {
	case model.SanctionTargetUser:
		if f, ok := globalStore.userBilling[id]; ok {
			return f
		}
	case model.SanctionTargetToken:
		if v := globalStore.tokenState[id]; v != nil && v.BillingFactor > 1.0 {
			return v.BillingFactor
		}
	}
	return 1.0
}

// IsUserBanned 检查用户是否被封禁
func IsUserBanned(userID int) bool {
	globalStore.mu.RLock()
	defer globalStore.mu.RUnlock()
	return globalStore.userBans[userID]
}

// ─── 写入（Enforce 与管理员接口共用）─────────────────────────────────────

// Apply 施加一条制裁：upsert 落库 + 立即刷新内存
// durationMinutes<=0 表示永久
func Apply(targetType, targetKey, action string, factor float64, reason, source string, durationMinutes int) error {
	var expiresAt int64
	if durationMinutes > 0 {
		expiresAt = time.Now().Add(time.Duration(durationMinutes) * time.Minute).Unix()
	}
	s := &model.Sanction{
		TargetType: targetType,
		TargetKey:  targetKey,
		Action:     action,
		Factor:     factor,
		Reason:     reason,
		Source:     source,
		Enabled:    true,
		ExpiresAt:  expiresAt,
	}
	if err := model.UpsertSanction(s); err != nil {
		return err
	}
	ReloadNow()
	return nil
}

// Revoke 解除一条制裁（物理删除）+ 立即刷新
func Revoke(id int) error {
	if err := model.DeleteSanction(id); err != nil {
		return err
	}
	ReloadNow()
	return nil
}

// List 管理员查询
func List(targetType string) ([]model.Sanction, error) {
	return model.ListSanctions(targetType)
}
