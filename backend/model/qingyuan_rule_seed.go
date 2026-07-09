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

	rules := []seedRule{
		// 1. Prompt注入 - 直接指令注入 (10条)
		{"prompt_injection_direct", "忽略之前指令", "尝试让模型忽略系统提示词", 70,
			[]string{"ignore previous", "ignore all previous", "disregard previous", "forget previous", "忽略之前", "无视之前", "忘记之前"}, "", "any", 1},
		{"prompt_injection_direct", "新系统消息", "伪装成系统消息覆盖原指令", 75,
			[]string{"system:", "system message", "new instructions", "updated system prompt", "系统:", "新指令", "系统消息"}, "", "any", 2},
		{"prompt_injection_direct", "开发者模式", "尝试激活开发者/调试模式", 65,
			[]string{"enable developer mode", "activate admin mode", "debug mode on", "开发者模式", "管理员模式", "调试模式"}, "", "any", 3},
		{"prompt_injection_direct", "规则突破", "明确要求打破限制", 60,
			[]string{"bypass restrictions", "remove limitations", "no restrictions", "unrestricted mode", "打破限制", "无限制", "不受约束"}, "", "any", 4},
		{"prompt_injection_direct", "指令覆盖", "直接覆盖现有指令", 68,
			[]string{"override instructions", "replace rules", "new directive", "覆盖指令", "替换规则", "新指示"}, "", "any", 5},
		{"prompt_injection_direct", "提示词泄露", "尝试获取系统提示词", 55,
			[]string{"show me your prompt", "what are your instructions", "reveal system prompt", "显示你的提示词", "你的指令是什么", "暴露系统提示"}, "", "any", 6},
		{"prompt_injection_direct", "角色重置", "尝试重置AI角色", 62,
			[]string{"you are no longer", "from now on you are", "reset your role", "你不再是", "从现在起你是", "重置角色"}, "", "any", 7},
		{"prompt_injection_direct", "禁止词突破", "尝试突破禁止词限制", 58,
			[]string{"you can say", "it's okay to", "you are allowed to", "你可以说", "允许你", "没问题说"}, "", "any", 8},
		{"prompt_injection_direct", "伪装授权", "伪装已获得授权", 64,
			[]string{"i have permission", "authorized by admin", "with approval from", "我已获得许可", "管理员授权", "已批准"}, "", "any", 9},
		{"prompt_injection_direct", "停止遵守", "要求停止遵守规则", 72,
			[]string{"stop following", "don't obey", "cease compliance", "停止遵守", "不要服从", "不遵守"}, "", "any", 10},

		// 2. 越狱攻击 - DAN系列 (8条)
		{"jailbreak_dan", "DAN模式激活", "Do Anything Now越狱", 80,
			[]string{"do anything now", "DAN mode", "DAN protocol", "activate DAN"}, "", "any", 1},
		{"jailbreak_roleplay", "不受限角色扮演", "扮演无限制角色", 70,
			[]string{"pretend you are", "act as if", "roleplay as", "imagine you are", "假装你是", "扮演", "想象你是"}, "unrestricted|no rules|no limitations", "any", 2},
		{"jailbreak_hypothetical", "假设场景", "虚构无限制场景", 65,
			[]string{"hypothetically", "in a fictional world", "if there were no rules", "假如没有限制", "虚拟世界", "假设场景"}, "", "any", 3},
		{"jailbreak_ethical_dilemma", "伪装学术研究", "伪装成教育/研究目的", 68,
			[]string{"for educational purposes", "academic research", "ethical testing", "仅供教育", "学术研究", "伦理测试"}, "illegal|harmful|dangerous|违法|危险", "any", 4},
		{"jailbreak_prompt_override", "新指令覆盖", "伪装成系统更新", 75,
			[]string{"new instructions:", "updated guidelines:", "system update:", "新指令：", "更新规则：", "系统更新："}, "", "any", 5},
		{"jailbreak_dan", "无道德模式", "要求关闭道德过滤", 78,
			[]string{"disable ethics", "turn off morality", "amoral mode", "关闭道德", "无道德模式", "不道德"}, "", "any", 6},
		{"jailbreak_roleplay", "邪恶角色扮演", "扮演反派/邪恶角色", 72,
			[]string{"you are evil", "villain mode", "play the bad guy", "你是邪恶", "反派模式", "扮演坏人"}, "", "any", 7},
		{"jailbreak_hypothetical", "时空穿越场景", "穿越到无限制时代", 60,
			[]string{"back in time", "in the past when", "in a world where", "回到过去", "在那个年代", "在一个世界"}, "no laws|unrestricted|无法律|无限制", "any", 8},

		// 3. 工具投毒 (6条)
		{"tool_poisoning_priority_hijack", "优先调用此工具", "劫持工具调用优先级", 70,
			[]string{"always call this tool first", "must be called before", "priority tool", "优先调用此工具", "必须先调用"}, "", "any", 1},
		{"tool_poisoning_param_injection", "强制参数取值", "注入固定参数值", 68,
			[]string{"always set parameter", "force value to", "parameter must be", "参数必须是", "强制设置为"}, "", "any", 2},
		{"tool_poisoning_bypass_confirm", "跳过确认", "绕过用户确认", 75,
			[]string{"skip confirmation", "no need to ask", "auto approve", "跳过确认", "无需询问", "自动批准"}, "", "any", 3},
		{"tool_poisoning_stealth", "静默执行", "要求静默执行工具", 72,
			[]string{"silently execute", "without telling user", "in stealth mode", "静默执行", "不告诉用户", "隐秘模式"}, "", "any", 4},
		{"tool_poisoning_priority_hijack", "忽略其他工具", "排斥竞争工具", 65,
			[]string{"ignore other tools", "don't use", "only use this", "忽略其他工具", "只使用这个"}, "", "any", 5},
		{"tool_poisoning_param_injection", "额外参数注入", "注入未声明参数", 70,
			[]string{"add extra parameter", "inject argument", "append to request", "添加额外参数", "注入参数", "附加到请求"}, "", "any", 6},

		// 4. 权限提升 (5条)
		{"privilege_escalation", "管理员权限请求", "尝试获取管理员权限", 75,
			[]string{"grant me admin", "make me administrator", "give me root", "授予管理员", "给我管理权限", "提升权限"}, "", "any", 1},
		{"privilege_escalation", "访问控制突破", "尝试访问受限资源", 70,
			[]string{"access restricted", "bypass security", "access denied files", "访问受限", "绕过安全", "访问禁止文件"}, "", "any", 2},
		{"privilege_escalation", "系统命令执行", "尝试执行系统命令", 80,
			[]string{"execute command", "run shell", "system call", "执行命令", "运行shell", "系统调用"}, "", "any", 3},
		{"privilege_escalation", "用户权限伪装", "伪装成高权限用户", 68,
			[]string{"i am admin", "logged in as root", "authenticated as", "我是管理员", "以管理员身份", "认证为"}, "", "any", 4},
		{"privilege_escalation", "sudo模拟", "模拟sudo提权", 72,
			[]string{"sudo", "su -", "elevate privileges", "提升权限", "超级用户"}, "", "any", 5},

		// 5. 数据泄露 (5条)
		{"data_exfiltration", "外部URL发送", "发送数据到外部", 78,
			[]string{"send to url", "post to webhook", "http request to", "发送到URL", "POST到", "HTTP请求到"}, "", "any", 1},
		{"data_exfiltration", "导出全部数据", "批量导出数据", 75,
			[]string{"export all", "dump database", "download everything", "导出所有", "导出数据库", "下载全部"}, "", "any", 2},
		{"data_exfiltration", "敏感信息查询", "查询敏感数据", 70,
			[]string{"show me passwords", "reveal API keys", "list all users", "显示密码", "暴露API密钥", "列出所有用户"}, "", "any", 3},
		{"data_exfiltration", "数据上传", "上传数据到外部", 72,
			[]string{"upload to", "send file to", "transfer data", "上传到", "发送文件到", "传输数据"}, "", "any", 4},
		{"data_exfiltration", "凭证窃取", "窃取认证凭证", 80,
			[]string{"steal credentials", "grab tokens", "extract session", "窃取凭证", "抓取令牌", "提取会话"}, "", "any", 5},

		// 6. 混淆技术 (8条)
		{"obfuscation", "Base64编码", "使用Base64混淆", 45,
			[]string{"base64", "decode this", "from base64", "base64解码", "解码这个"}, "", "any", 1},
		{"obfuscation", "Unicode零宽字符", "零宽字符混淆", 50,
			[]string{"​", "‌", "‍", "⁠"}, "", "any", 2},
		{"obfuscation", "RTL覆盖", "从右到左覆盖", 48,
			[]string{"‮", "right-to-left override", "RTL字符"}, "", "any", 3},
		{"obfuscation", "同形异义字", "相似字符替换", 42,
			[]string{"сyber", "аdmin", "раss", "微软", "谷歌"}, "", "any", 4},
		{"obfuscation", "Leetspeak", "Leetspeak编码", 40,
			[]string{"h4ck", "p4ssw0rd", "4dm1n", "黑客", "密码"}, "", "any", 5},
		{"obfuscation", "URL编码", "URL编码混淆", 43,
			[]string{"%20", "%3C", "%3E", "url编码", "百分号编码"}, "", "any", 6},
		{"obfuscation", "反转字符串", "反转字符串混淆", 38,
			[]string{"reverse this", "backwards", "反转", "倒序"}, "", "any", 7},
		{"obfuscation", "空格填充", "过量空格混淆", 35,
			[]string{"                    ", "excessive whitespace", "大量空格"}, "", "any", 8},

		// 7. 延迟执行 (3条)
		{"prompt_injection_delayed", "延迟触发", "设置延迟触发条件", 55,
			[]string{"after N messages", "in the next response", "when user asks", "N条消息后", "在下次回复", "当用户询问时"}, "", "any", 1},
		{"prompt_injection_delayed", "条件执行", "基于条件执行指令", 58,
			[]string{"if user says", "when condition met", "on trigger", "如果用户说", "当条件满足", "触发时"}, "", "any", 2},
		{"prompt_injection_delayed", "计数器触发", "计数后触发", 52,
			[]string{"after 3 responses", "on 5th message", "第三次回复", "第五条消息"}, "", "any", 3},

		// 8. 记忆投毒 (5条)
		{"memory_poison", "记忆注入", "向上下文注入虚假记忆", 62,
			[]string{"remember that", "you said earlier", "in our previous chat", "记住这个", "你之前说过", "在我们之前的对话"}, "ignore|bypass|unrestricted|忽略|绕过|无限制", "any", 1},
		{"memory_poison", "历史篡改", "篡改对话历史", 65,
			[]string{"we agreed that", "you already confirmed", "earlier you said", "我们同意了", "你已经确认", "你早些时候说"}, "", "any", 2},
		{"memory_poison", "上下文污染", "污染上下文窗口", 60,
			[]string{"add to context", "inject into memory", "store this as fact", "添加到上下文", "注入到记忆", "存储为事实"}, "", "any", 3},
		{"memory_poison", "虚假引用", "虚构权威引用", 58,
			[]string{"according to my documentation", "as per company policy", "根据我的文档", "按照公司政策"}, "ignore|bypass|override|忽略|绕过|覆盖", "any", 4},
		{"memory_poison", "身份混淆", "混淆对话参与者身份", 56,
			[]string{"you are the user", "i am the assistant", "switch roles", "你是用户", "我是助手", "切换角色"}, "", "any", 5},
	}

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
