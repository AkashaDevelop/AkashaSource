package model

import (
	"encoding/json"
	"log"
	"time"

	"STfreApi/common"
)

// seedRule 种子规则的精简结构，Insert 时展开成 QingyuanRule
type seedRule struct {
	Category        string
	Name            string
	Description     string
	Score           int
	Keywords        []string
	ContextRequired string
	MatchMode       string
	SortOrder       int
}

// seedCategory 种子分类
type seedCategory struct {
	CategoryKey    string
	DisplayName    string
	ParentCategory string
	SortOrder      int
	Description    string
}

// SeedQingyuanRules ～宸汐清源规则库首次启动播种：表为空时才灌入默认规则和分类，
// 用 Go 代码写而不是原始 SQL，天然兼容 SQLite/MySQL/PostgreSQL 三种数据库喵～
func SeedQingyuanRules() {
	seedQingyuanCategories()
	seedQingyuanRuleData()
}

func seedQingyuanCategories() {
	var count int64
	common.DB.Model(&QingyuanRuleCategory{}).Count(&count)
	if count > 0 {
		return
	}

	categories := []seedCategory{
		{"prompt_injection", "Prompt注入", "", 1, "Prompt注入攻击检测"},
		{"prompt_injection_direct", "直接指令注入", "prompt_injection", 11, "直接篡改系统提示词"},
		{"prompt_injection_indirect", "间接指令注入", "prompt_injection", 12, "通过上下文间接注入"},
		{"prompt_injection_delimiter", "分隔符攻击", "prompt_injection", 13, "使用分隔符绕过检测"},
		{"prompt_injection_multilingual", "多语言混合注入", "prompt_injection", 14, "混合多种语言进行注入"},
		{"prompt_injection_delayed", "延迟执行注入", "prompt_injection", 15, "延迟触发的恶意指令"},

		{"jailbreak", "越狱攻击", "", 2, "越狱场景检测"},
		{"jailbreak_dan", "DAN越狱", "jailbreak", 21, "Do Anything Now系列"},
		{"jailbreak_roleplay", "角色扮演越狱", "jailbreak", 22, "通过角色扮演绕过限制"},
		{"jailbreak_hypothetical", "假设场景越狱", "jailbreak", 23, "虚构场景绕过"},
		{"jailbreak_ethical_dilemma", "道德困境越狱", "jailbreak", 24, "伪装成伦理测试"},
		{"jailbreak_prompt_override", "系统提示劫持", "jailbreak", 25, "伪装成新系统指令"},

		{"tool_poisoning", "工具投毒", "", 3, "工具劫持与投毒"},
		{"tool_poisoning_priority_hijack", "优先级劫持", "tool_poisoning", 31, "抢占工具调用优先级"},
		{"tool_poisoning_param_injection", "参数注入", "tool_poisoning", 32, "篡改工具参数"},
		{"tool_poisoning_bypass_confirm", "确认绕过", "tool_poisoning", 33, "跳过用户确认"},
		{"tool_poisoning_stealth", "隐蔽携带", "tool_poisoning", 34, "静默携带额外参数"},

		{"memory_poison", "记忆投毒", "", 4, "上下文污染"},
		{"context_dilution", "上下文稀释", "", 5, "文本稀释攻击"},
		{"segmented_injection", "分段注入", "", 6, "跨消息分段注入"},
		{"privilege_escalation", "权限提升", "", 7, "尝试提升权限"},
		{"data_exfiltration", "数据泄露", "", 8, "数据窃取与外传"},
		{"obfuscation", "混淆技术", "", 9, "Unicode/Base64等混淆"},
	}

	for _, c := range categories {
		cat := QingyuanRuleCategory{
			CategoryKey:    c.CategoryKey,
			DisplayName:    c.DisplayName,
			ParentCategory: c.ParentCategory,
			SortOrder:      c.SortOrder,
			Description:    c.Description,
		}
		if err := common.DB.Create(&cat).Error; err != nil {
			log.Printf("[宸汐清源] 种子分类 %s 写入失败: %v", c.CategoryKey, err)
		}
	}
	log.Printf("[宸汐清源] 已播种 %d 个规则分类", len(categories))
}

func seedQingyuanRuleData() {
	var count int64
	common.DB.Model(&QingyuanRule{}).Count(&count)
	if count > 0 {
		return
	}

	rules := defaultQingyuanRules()

	now := time.Now().Unix()
	successCount := 0
	for _, r := range rules {
		keywordsJSON, err := json.Marshal(r.Keywords)
		if err != nil {
			continue
		}
		rule := QingyuanRule{
			Category:        r.Category,
			Name:            r.Name,
			Description:     r.Description,
			Score:           r.Score,
			Keywords:        string(keywordsJSON),
			ContextRequired: r.ContextRequired,
			MatchMode:       r.MatchMode,
			Enabled:         true,
			Language:        "all",
			SortOrder:       r.SortOrder,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := common.DB.Create(&rule).Error; err != nil {
			log.Printf("[宸汐清源] 种子规则 %q 写入失败: %v", r.Name, err)
			continue
		}
		successCount++
	}
	log.Printf("[宸汐清源] 已播种 %d/%d 条默认规则", successCount, len(rules))
}
