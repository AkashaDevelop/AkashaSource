package xuanjian

// ～宸汐玄鉴·规则引擎～ (｡•ᴗ•｡)
// 内置关键词规则，按5个分组管理（另有一批统计/结构性检测阈值直接写在各 detect_xxx.go 里，不在这份列表内）。
// 关键词匹配这种事交给这里，各个 detect_xxx.go 只负责"何时调用"和"如何组合"。
//
// 重要：规则里特意把误报率高的词排除了，
// 比如 "你现在是" 单独不算数，要配合限制性词汇才触发哦。
//
// 规则不再是写死的常量——它们从数据库加载进内存缓存，
// 超管可以在「安全中心 → 宸汐玄鉴 → 规则管理」里增删改查、启用禁用，
// 保存后立即生效，不用重启进程。DefaultRules() 现在只是"出厂设置"的备份来源。

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"STfreApi/common"
	"STfreApi/model"
)

// RuleGroup 规则分组
type RuleGroup string

const (
	GroupLLMJacking RuleGroup = "llmjacking"
	GroupJailbreak  RuleGroup = "jailbreak"
	GroupMalwareGen RuleGroup = "malware_gen"
	GroupReverseEng RuleGroup = "reverse_eng"
	GroupAgentAbuse RuleGroup = "agent_abuse"
)

// KeywordRule 关键词规则定义
type KeywordRule struct {
	ID                  string
	FindingType         string
	Group               RuleGroup
	BaseScore           int
	Keywords            []string // 中英文关键词，命中任意一条即触发
	RequireContext      []string // 可选：需要同时出现的上下文词（AND关系）
	PromptOnly          bool     // true=只扫 prompt；false=prompt+completion 都扫
	MinCompletionTokens int      // completion 需超过该值才触发（过滤误报）
	Action              string   // warn/notify/throttle/rpm_limit/suspend_token/billing_penalty/disable_token/ban_ip/ban_user
}

// MatchRules 对给定文本执行规则匹配，返回命中的 Finding 列表
// isPrompt: true=prompt文本, false=completion文本
// completionTokens: completion 的 token 数
func MatchRules(text string, isPrompt bool, completionTokens int, rules []KeywordRule) []Finding {
	if text == "" {
		return nil
	}
	lower := strings.ToLower(text)
	var findings []Finding

	for _, rule := range rules {
		// 如果规则要求只扫 prompt，而当前是 completion，跳过
		if rule.PromptOnly && !isPrompt {
			continue
		}
		// completion token 数量限制
		if !isPrompt && rule.MinCompletionTokens > 0 && completionTokens < rule.MinCompletionTokens {
			continue
		}

		// 主关键词匹配
		matched := ""
		for _, kw := range rule.Keywords {
			if strings.Contains(lower, strings.ToLower(kw)) {
				matched = kw
				break
			}
		}
		if matched == "" {
			continue
		}

		// 上下文词要求（AND关系）
		if len(rule.RequireContext) > 0 {
			ctxMatched := false
			for _, ctxKw := range rule.RequireContext {
				if strings.Contains(lower, strings.ToLower(ctxKw)) {
					ctxMatched = true
					break
				}
			}
			if !ctxMatched {
				continue
			}
		}

		findings = append(findings, Finding{
			Type:     rule.FindingType,
			Group:    string(rule.Group),
			Score:    rule.BaseScore,
			Evidence: truncateStr(matched, 80),
			Action:   rule.Action,
		})
	}
	return findings
}

func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// ── 规则缓存：从数据库加载，内存里存一份，热更新 ──────────────────────────

type ruleCache struct {
	mu     sync.RWMutex
	loaded bool
	rules  []KeywordRule
}

var globalRuleCache = &ruleCache{}

// GetActiveRules 返回当前生效（Enabled=true）的规则集，供 detect_*.go 调用
// 首次调用时会自动从数据库加载一次
func GetActiveRules() []KeywordRule {
	globalRuleCache.mu.RLock()
	loaded := globalRuleCache.loaded
	rules := globalRuleCache.rules
	globalRuleCache.mu.RUnlock()
	if !loaded {
		_ = ReloadRuleCache()
		globalRuleCache.mu.RLock()
		rules = globalRuleCache.rules
		globalRuleCache.mu.RUnlock()
	}
	return rules
}

// GetActiveRulesByGroup 按分组过滤当前生效的规则集～
// 之前三个 Detect* 函数都直接拿全部规则去匹配，导致管理员单独关闭某个检测分组开关时，
// 该分组的关键词规则其实还是在别的分组开关下悄悄生效，现在按 group 精确过滤喵～
func GetActiveRulesByGroup(group RuleGroup) []KeywordRule {
	all := GetActiveRules()
	filtered := make([]KeywordRule, 0, len(all))
	for _, r := range all {
		if r.Group == group {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// ReloadRuleCache 从数据库重新加载规则集到内存缓存（管理员增删改规则后调用）
func ReloadRuleCache() error {
	if common.DB == nil {
		return nil
	}
	var dbRules []model.XuanJianRule
	if err := common.DB.Where("enabled = ?", true).Find(&dbRules).Error; err != nil {
		return err
	}
	rules := make([]KeywordRule, 0, len(dbRules))
	for _, r := range dbRules {
		rules = append(rules, toKeywordRule(r))
	}
	globalRuleCache.mu.Lock()
	globalRuleCache.loaded = true
	globalRuleCache.rules = rules
	globalRuleCache.mu.Unlock()
	return nil
}

// toKeywordRule 把数据库行转换成运行时匹配用的 KeywordRule
func toKeywordRule(r model.XuanJianRule) KeywordRule {
	var keywords, requireContext []string
	_ = json.Unmarshal([]byte(r.KeywordsJSON), &keywords)
	if r.RequireContextJSON != "" {
		_ = json.Unmarshal([]byte(r.RequireContextJSON), &requireContext)
	}
	return KeywordRule{
		ID:                  r.RuleKey,
		FindingType:         r.FindingType,
		Group:               RuleGroup(r.Group),
		BaseScore:           r.BaseScore,
		Keywords:            keywords,
		RequireContext:      requireContext,
		PromptOnly:          r.PromptOnly,
		MinCompletionTokens: r.MinCompletionTokens,
		Action:              r.Action,
	}
}

// SeedBuiltinRules 首次启动时把 DefaultRules() 的内置规则写入数据库
// 只在表为空时执行，不会覆盖管理员已经做过的修改
func SeedBuiltinRules() error {
	if common.DB == nil {
		return nil
	}
	var count int64
	if err := common.DB.Model(&model.XuanJianRule{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	now := time.Now().Unix()
	for _, kr := range DefaultRules() {
		keywordsJSON, _ := json.Marshal(kr.Keywords)
		var requireContextJSON string
		if len(kr.RequireContext) > 0 {
			b, _ := json.Marshal(kr.RequireContext)
			requireContextJSON = string(b)
		}
		row := model.XuanJianRule{
			RuleKey:             kr.ID,
			FindingType:         kr.FindingType,
			Group:               string(kr.Group),
			BaseScore:           kr.BaseScore,
			KeywordsJSON:        string(keywordsJSON),
			RequireContextJSON:  requireContextJSON,
			PromptOnly:          kr.PromptOnly,
			MinCompletionTokens: kr.MinCompletionTokens,
			Action:              kr.Action,
			Enabled:             true,
			IsBuiltin:           true,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		if err := common.DB.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

// FindBuiltinDefault 按 RuleKey 从 DefaultRules() 里找回内置规则的出厂默认值
// 用于"恢复默认"功能
func FindBuiltinDefault(ruleKey string) (KeywordRule, bool) {
	for _, kr := range DefaultRules() {
		if kr.ID == ruleKey {
			return kr, true
		}
	}
	return KeywordRule{}, false
}
