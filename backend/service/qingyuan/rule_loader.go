package qingyuan

import (
	"encoding/json"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"STfreApi/common"
	"STfreApi/model"
)

// DynamicRule 动态规则结构
type DynamicRule struct {
	ID              int
	Category        string
	Name            string
	Score           int
	Keywords        []string
	ContextRequired string
	MatchMode       string // any/all/regex
	Enabled         bool
	Language        string
}

// RuleCache 规则缓存
type RuleCache struct {
	mu           sync.RWMutex
	rules        map[string][]DynamicRule // category -> []rules
	lastReload   time.Time
	reloadTicker *time.Ticker
}

var globalRuleCache = &RuleCache{
	rules: make(map[string][]DynamicRule),
}

// InitRuleCache 启动规则缓存自动刷新（每 30 秒）
func InitRuleCache() {
	globalRuleCache.Reload()
	globalRuleCache.reloadTicker = time.NewTicker(30 * time.Second)
	go func() {
		for range globalRuleCache.reloadTicker.C {
			globalRuleCache.Reload()
		}
	}()
	log.Printf("[宸汐清源] 规则缓存自动刷新已启动，间隔 30 秒")
}

// Reload 从数据库重新加载规则
func (rc *RuleCache) Reload() {
	var dbRules []model.QingyuanRule
	enabled := true
	if err := common.DB.Where("enabled = ?", enabled).Find(&dbRules).Error; err != nil {
		log.Printf("[宸汐清源] 规则加载失败: %v", err)
		return
	}

	newRules := make(map[string][]DynamicRule)
	for _, r := range dbRules {
		var keywords []string
		if err := json.Unmarshal([]byte(r.Keywords), &keywords); err != nil {
			log.Printf("[宸汐清源] 规则 %d 的 keywords 解析失败: %v", r.Id, err)
			continue
		}

		rule := DynamicRule{
			ID:              r.Id,
			Category:        r.Category,
			Name:            r.Name,
			Score:           r.Score,
			Keywords:        keywords,
			ContextRequired: r.ContextRequired,
			MatchMode:       r.MatchMode,
			Enabled:         r.Enabled,
			Language:        r.Language,
		}
		newRules[r.Category] = append(newRules[r.Category], rule)
	}

	rc.mu.Lock()
	rc.rules = newRules
	rc.lastReload = time.Now()
	rc.mu.Unlock()

	totalRules := 0
	for _, rules := range newRules {
		totalRules += len(rules)
	}
	log.Printf("[宸汐清源] 规则缓存已刷新，共加载 %d 条规则，分布在 %d 个分类", totalRules, len(newRules))
}

// GetRulesByCategory 获取指定分类的规则
func (rc *RuleCache) GetRulesByCategory(category string) []DynamicRule {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.rules[category]
}

// ReloadRulesNow 手动触发刷新（前端修改规则后调用）
func ReloadRulesNow() {
	globalRuleCache.Reload()
}

// GetLastReloadTime 获取上次刷新时间
func GetLastReloadTime() time.Time {
	globalRuleCache.mu.RLock()
	defer globalRuleCache.mu.RUnlock()
	return globalRuleCache.lastReload
}

// detectWithDynamicRules 使用动态规则进行检测
func detectWithDynamicRules(text string, category string) []Finding {
	rules := globalRuleCache.GetRulesByCategory(category)
	if len(rules) == 0 {
		return nil
	}

	findings := []Finding{}
	lowerText := strings.ToLower(text)

	for _, rule := range rules {
		matched := false
		matchedKeyword := ""

		switch rule.MatchMode {
		case "any": // 任意关键词匹配
			for _, kw := range rule.Keywords {
				if strings.Contains(lowerText, strings.ToLower(kw)) {
					matched = true
					matchedKeyword = kw
					break
				}
			}

		case "all": // 所有关键词都匹配
			allMatched := true
			for _, kw := range rule.Keywords {
				if !strings.Contains(lowerText, strings.ToLower(kw)) {
					allMatched = false
					break
				}
			}
			if allMatched {
				matched = true
				matchedKeyword = strings.Join(rule.Keywords, " + ")
			}

		case "regex": // 正则匹配
			for _, pattern := range rule.Keywords {
				if re, err := regexp.Compile(pattern); err == nil {
					if re.MatchString(text) {
						matched = true
						matchedKeyword = pattern
						break
					}
				}
			}
		}

		// 上下文要求检查
		if matched && rule.ContextRequired != "" {
			contextMatched := false
			contextKeywords := strings.Split(rule.ContextRequired, "|")
			for _, ctx := range contextKeywords {
				if strings.Contains(lowerText, strings.ToLower(ctx)) {
					contextMatched = true
					break
				}
			}
			if !contextMatched {
				matched = false
			}
		}

		if matched {
			findings = append(findings, Finding{
				Type:     rule.Category,
				Severity: severity(rule.Score),
				Score:    rule.Score,
				Path:     "dynamic_rule",
				Evidence: rule.Name + " (" + matchedKeyword + ")",
				Action:   "monitor",
			})
		}
	}

	return findings
}
