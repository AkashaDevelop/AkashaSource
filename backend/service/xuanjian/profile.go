package xuanjian

// ～宸汐玄鉴·行为画像档案室～ (づ｡◕‿‿◕｡)づ
// 每一个 token 的行为都在这里默默积累，
// 窗口期一过就会自动淡忘，不会永远追究哦～
// 注意：画像只存在内存里，进程重启就清空——这是故意的设计，
// 防止攻击者通过观察冷却时间来估算防御窗口。

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// shortPromptMaxTokens 读取全局配置里的短 prompt token 上限，未配置时兜底默认值 20
func shortPromptMaxTokens() int {
	cfg, _ := GetConfig()
	if cfg.ShortPromptMaxTokens > 0 {
		return cfg.ShortPromptMaxTokens
	}
	return 20
}

// TokenProfile 单个 API Token 的行为画像
type TokenProfile struct {
	mu sync.Mutex // 每个 profile 独立锁，避免全局锁争抢

	TokenID   int
	UserID    int
	FirstSeen time.Time
	LastSeen  time.Time

	// 滑动窗口（每次请求后检查是否需要重置）
	WindowStart time.Time

	// 基础计数
	RequestCount     int
	QuotaBurned      int64
	ErrorCount       int
	ShortPromptCount int

	// 集合计数
	ModelSet  map[string]int // 模型名 → 请求次数
	IPCIDRSet map[string]int // /24 CIDR → 请求次数（修正：按 CIDR 聚合非单 IP）

	// 时序分析：最近 20 次请求的间隔 ms
	IntervalBuffer []int64
	LastRequestAt  time.Time

	// qingyuan 信号序列：最近 20 次的风险分
	QYScoreSequence []int

	// 破限会话跟踪
	SessionMessages   int // 当前连续对话轮数
	JailbreakAttempts int // 累计破限命中次数
	LastJailbreakAt   time.Time

	// MinHash prompt 环形缓冲（最近 50 条，存完整签名才能做真正的近似相似度比较）
	PromptHashes []MinHashSignature

	// LLMjacking 专属：全量 quota（跨窗口累计，不重置）
	TotalLifetimeQuota int64
	TokenCreatedAt     time.Time
}

// UserProfile 用户级画像（跨 token 聚合）
type UserProfile struct {
	mu sync.Mutex

	UserID      int
	WindowStart time.Time
	TokenIDSet  map[int]bool   // 5min 内用过的 token_id
	IPCIDRSet   map[string]int // /24 CIDR → 次数
}

// profileStore 全局画像存储
type profileStore struct {
	mu     sync.RWMutex
	tokens map[int]*TokenProfile // tokenID → profile
	users  map[int]*UserProfile  // userID → profile
}

var globalStore = &profileStore{
	tokens: make(map[int]*TokenProfile),
	users:  make(map[int]*UserProfile),
}

// GetOrCreateToken 获取或新建 TokenProfile
func GetOrCreateToken(tokenID, userID int, tokenCreatedAt time.Time) *TokenProfile {
	globalStore.mu.RLock()
	p := globalStore.tokens[tokenID]
	globalStore.mu.RUnlock()
	if p != nil {
		return p
	}

	globalStore.mu.Lock()
	defer globalStore.mu.Unlock()
	if p = globalStore.tokens[tokenID]; p != nil {
		return p
	}
	now := time.Now()
	p = &TokenProfile{
		TokenID:        tokenID,
		UserID:         userID,
		FirstSeen:      now,
		LastSeen:       now,
		WindowStart:    now,
		ModelSet:       make(map[string]int),
		IPCIDRSet:      make(map[string]int),
		TokenCreatedAt: tokenCreatedAt,
	}
	globalStore.tokens[tokenID] = p
	return p
}

// GetOrCreateUser 获取或新建 UserProfile
func GetOrCreateUser(userID int) *UserProfile {
	globalStore.mu.RLock()
	p := globalStore.users[userID]
	globalStore.mu.RUnlock()
	if p != nil {
		return p
	}

	globalStore.mu.Lock()
	defer globalStore.mu.Unlock()
	if p = globalStore.users[userID]; p != nil {
		return p
	}
	p = &UserProfile{
		UserID:      userID,
		WindowStart: time.Now(),
		TokenIDSet:  make(map[int]bool),
		IPCIDRSet:   make(map[string]int),
	}
	globalStore.users[userID] = p
	return p
}

// UpdateToken 更新 TokenProfile（在 RecordRequest 里异步调用）
func (p *TokenProfile) Update(rec RequestRecord, windowMinutes int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	windowDur := time.Duration(windowMinutes) * time.Minute

	// 窗口重置
	if now.Sub(p.WindowStart) > windowDur {
		p.WindowStart = now
		p.RequestCount = 0
		p.QuotaBurned = 0
		p.ErrorCount = 0
		p.ShortPromptCount = 0
		p.ModelSet = make(map[string]int)
		p.IPCIDRSet = make(map[string]int)
	}

	p.RequestCount++
	p.QuotaBurned += int64(rec.PromptTokens + rec.CompletionTokens)
	p.TotalLifetimeQuota += int64(rec.PromptTokens + rec.CompletionTokens)
	p.LastSeen = now

	if rec.StatusCode >= 400 {
		p.ErrorCount++
	}
	if rec.PromptTokens > 0 && rec.PromptTokens < shortPromptMaxTokens() {
		p.ShortPromptCount++
	}

	if rec.Model != "" {
		p.ModelSet[rec.Model]++
	}

	cidr := extractCIDR(rec.IP)
	if cidr != "" {
		p.IPCIDRSet[cidr]++
	}

	// 时序间隔
	if !p.LastRequestAt.IsZero() {
		interval := now.Sub(p.LastRequestAt).Milliseconds()
		p.IntervalBuffer = append(p.IntervalBuffer, interval)
		if len(p.IntervalBuffer) > 20 {
			p.IntervalBuffer = p.IntervalBuffer[len(p.IntervalBuffer)-20:]
		}
	}
	p.LastRequestAt = now

	// qingyuan 分数序列
	p.QYScoreSequence = append(p.QYScoreSequence, rec.QYRiskScore)
	if len(p.QYScoreSequence) > 20 {
		p.QYScoreSequence = p.QYScoreSequence[len(p.QYScoreSequence)-20:]
	}

	// prompt hash
	if !rec.PromptHash.IsZero() {
		p.PromptHashes = append(p.PromptHashes, rec.PromptHash)
		if len(p.PromptHashes) > 50 {
			p.PromptHashes = p.PromptHashes[len(p.PromptHashes)-50:]
		}
	}

	// 会话计数
	p.SessionMessages++
}

// UpdateUser 更新 UserProfile
func (p *UserProfile) Update(tokenID int, ip string, windowMinutes int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	windowDur := time.Duration(windowMinutes) * time.Minute
	if now.Sub(p.WindowStart) > windowDur {
		p.WindowStart = now
		p.TokenIDSet = make(map[int]bool)
		p.IPCIDRSet = make(map[string]int)
	}

	p.TokenIDSet[tokenID] = true
	cidr := extractCIDR(ip)
	if cidr != "" {
		p.IPCIDRSet[cidr]++
	}
}

// StdDev 计算时序间隔标准差（用于机器人检测）
func (p *TokenProfile) StdDev() float64 {
	if len(p.IntervalBuffer) < 5 {
		return -1 // 数据不足
	}
	var sum float64
	for _, v := range p.IntervalBuffer {
		sum += float64(v)
	}
	mean := sum / float64(len(p.IntervalBuffer))
	var variance float64
	for _, v := range p.IntervalBuffer {
		d := float64(v) - mean
		variance += d * d
	}
	variance /= float64(len(p.IntervalBuffer))
	// 简单整数平方根近似
	return sqrtApprox(variance)
}

// QYScoreRising 判断 qingyuan 风险分是否在单调递增（Crescendo 检测）
func (p *TokenProfile) QYScoreRising(minLen int) bool {
	if len(p.QYScoreSequence) < minLen {
		return false
	}
	seq := p.QYScoreSequence[len(p.QYScoreSequence)-minLen:]
	for i := 1; i < len(seq); i++ {
		if seq[i] < seq[i-1] {
			return false
		}
	}
	return seq[len(seq)-1] >= 40 // 末尾需要达到一定分数才算真正"上升"
}

// extractCIDR 从 IP 字符串提取 /24 CIDR（修正后：聚合同 C 段 IP）
func extractCIDR(ipStr string) string {
	if ipStr == "" {
		return ""
	}
	// 去掉端口
	if strings.Contains(ipStr, ":") && !strings.HasPrefix(ipStr, "[") {
		parts := strings.Split(ipStr, ":")
		if len(parts) == 2 {
			ipStr = parts[0]
		}
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ipStr
	}
	if ip.To4() != nil {
		// IPv4 取前三段
		parts := strings.Split(ip.String(), ".")
		if len(parts) == 4 {
			return fmt.Sprintf("%s.%s.%s.0/24", parts[0], parts[1], parts[2])
		}
	}
	// IPv6 取前 /48
	mask := net.CIDRMask(48, 128)
	ip = ip.Mask(mask)
	return ip.String() + "/48"
}

func sqrtApprox(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x / 2
	for i := 0; i < 20; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}

// ResetTokenProfile 手动清除某 token 画像（管理员怀疑误判时使用）
func ResetTokenProfile(tokenID int) {
	globalStore.mu.Lock()
	defer globalStore.mu.Unlock()
	delete(globalStore.tokens, tokenID)
}
func CleanupExpired() {
	expiry := time.Now().Add(-time.Hour)
	globalStore.mu.Lock()
	defer globalStore.mu.Unlock()
	for id, p := range globalStore.tokens {
		if p.LastSeen.Before(expiry) {
			delete(globalStore.tokens, id)
		}
	}
	for id, p := range globalStore.users {
		if p.WindowStart.Add(time.Hour).Before(time.Now()) {
			delete(globalStore.users, id)
		}
	}
}

// ProfileStats 返回当前画像统计（用于管理界面）
func ProfileStats() map[string]interface{} {
	globalStore.mu.RLock()
	defer globalStore.mu.RUnlock()
	return map[string]interface{}{
		"active_tokens": len(globalStore.tokens),
		"active_users":  len(globalStore.users),
	}
}

// TopRiskTokens 返回风险最高的前 N 个 token profile 快照（不含敏感内容）
func TopRiskTokens(n int) []map[string]interface{} {
	globalStore.mu.RLock()
	defer globalStore.mu.RUnlock()
	type entry struct {
		tokenID int
		score   int
		profile *TokenProfile
	}
	entries := make([]entry, 0, len(globalStore.tokens))
	for id, p := range globalStore.tokens {
		lastScore := 0
		if len(p.QYScoreSequence) > 0 {
			lastScore = p.QYScoreSequence[len(p.QYScoreSequence)-1]
		}
		entries = append(entries, entry{id, lastScore + p.JailbreakAttempts*10, p})
	}
	// 简单选择排序取 top-N（N通常很小）
	for i := 0; i < len(entries) && i < n; i++ {
		maxIdx := i
		for j := i + 1; j < len(entries); j++ {
			if entries[j].score > entries[maxIdx].score {
				maxIdx = j
			}
		}
		entries[i], entries[maxIdx] = entries[maxIdx], entries[i]
	}
	if len(entries) > n {
		entries = entries[:n]
	}
	result := make([]map[string]interface{}, 0, len(entries))
	for _, e := range entries {
		e.profile.mu.Lock()
		result = append(result, map[string]interface{}{
			"token_id":        e.profile.TokenID,
			"user_id":         e.profile.UserID,
			"request_count":   e.profile.RequestCount,
			"quota_burned":    e.profile.QuotaBurned,
			"model_count":     len(e.profile.ModelSet),
			"ip_cidr_count":   len(e.profile.IPCIDRSet),
			"jailbreak_count": e.profile.JailbreakAttempts,
			"last_seen":       e.profile.LastSeen.Unix(),
		})
		e.profile.mu.Unlock()
	}
	return result
}
