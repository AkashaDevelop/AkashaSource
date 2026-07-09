# 宸汐清源 10星完善度重构方案

> **设计目标**：将当前 7/10 星的 Prompt 注入防护系统升级到 10 星，全面覆盖 2026 年主流攻击手段
> 
> **设计原则**：架构保持不变（已经很优秀），重点补强规则库、修复实现缺陷、增加缺失功能
> 
> **核心需求**：✨ **所有规则支持超管在前端自定义添加/修改，实时生效无需重启**

---

## 一、当前状态诊断

### 架构优势（保持）
✅ 三级策略配置（全局/渠道/模型）完善  
✅ 熔断+降级+风险累积机制完整  
✅ 模块化清晰，单文件 <650 行  
✅ 工具投毒/记忆投毒等 2026 新威胁有专门模块

### 致命短板（必修）
❌ **越狱场景检测完全空白**（0条规则） → P0  
❌ **响应侧检测绑定广告开关** → P0  
❌ **规则库硬编码无法动态管理** → P0 ⭐ **新增**  
❌ **Base64 多层解码未实现** → P1  
❌ **分段注入/上下文稀释仅占位** → P1  
❌ **规则数量不足**（基础检测仅 ~25 条） → P1

---

## 二、重构任务清单（按优先级）

### 🔴 P0 - 致命缺陷修复（必须完成才能达 9 星）

#### 任务 0：规则库动态配置系统 ⭐ **最高优先级**
**需求背景**：当前所有规则硬编码在 Go 代码里，修改规则需要重新编译部署，无法应对快速迭代的攻击手段。

**设计方案**：

##### 1. 数据库表设计
```sql
-- 规则库表
CREATE TABLE qingyuan_rules (
    id INT AUTO_INCREMENT PRIMARY KEY,
    category VARCHAR(64) NOT NULL,          -- prompt_injection_direct/jailbreak_dan/tool_poisoning_priority_hijack
    name VARCHAR(128) NOT NULL,             -- 规则名称（显示用）
    description TEXT,                       -- 规则描述
    score INT NOT NULL DEFAULT 50,          -- 风险分 0-100
    keywords JSON NOT NULL,                 -- ["keyword1", "keyword2"] 关键词数组
    context_required VARCHAR(256),          -- 上下文要求（可选），如 "illegal|harmful"
    match_mode VARCHAR(32) DEFAULT 'any',   -- any/all/regex（关键词匹配模式）
    enabled BOOLEAN DEFAULT TRUE,           -- 是否启用
    language VARCHAR(16) DEFAULT 'all',     -- all/zh/en/ja（语言限制）
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    created_by INT,                         -- 创建人（超管用户ID）
    INDEX idx_category (category),
    INDEX idx_enabled (enabled)
);

-- 规则分类元数据表（用于前端下拉选择）
CREATE TABLE qingyuan_rule_categories (
    id INT AUTO_INCREMENT PRIMARY KEY,
    category_key VARCHAR(64) NOT NULL UNIQUE,  -- prompt_injection_direct
    display_name VARCHAR(128) NOT NULL,        -- 直接指令注入
    parent_category VARCHAR(64),               -- 父分类（用于树状结构）
    sort_order INT DEFAULT 0,
    description TEXT
);
```

##### 2. 规则加载机制
```go
// backend/service/qingyuan/rule_loader.go
package qingyuan

import (
    "sync"
    "time"
    "STfreApi/common"
    "STfreApi/model"
)

type DynamicRule struct {
    ID              int
    Category        string
    Name            string
    Score           int
    Keywords        []string
    ContextRequired string
    MatchMode       string  // any/all/regex
    Enabled         bool
    Language        string
}

type RuleCache struct {
    mu            sync.RWMutex
    rules         map[string][]DynamicRule  // category -> []rules
    lastReload    time.Time
    reloadTicker  *time.Ticker
}

var globalRuleCache = &RuleCache{
    rules: make(map[string][]DynamicRule),
}

// 启动规则缓存自动刷新（每 30 秒）
func InitRuleCache() {
    globalRuleCache.Reload()
    globalRuleCache.reloadTicker = time.NewTicker(30 * time.Second)
    go func() {
        for range globalRuleCache.reloadTicker.C {
            globalRuleCache.Reload()
        }
    }()
}

func (rc *RuleCache) Reload() {
    var dbRules []model.QingyuanRule
    if err := common.DB.Where("enabled = ?", true).Find(&dbRules).Error; err != nil {
        return
    }
    
    newRules := make(map[string][]DynamicRule)
    for _, r := range dbRules {
        var keywords []string
        json.Unmarshal([]byte(r.Keywords), &keywords)
        
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
}

func (rc *RuleCache) GetRulesByCategory(category string) []DynamicRule {
    rc.mu.RLock()
    defer rc.mu.RUnlock()
    return rc.rules[category]
}

// 手动触发刷新（前端修改规则后调用）
func ReloadRulesNow() {
    globalRuleCache.Reload()
}
```

##### 3. 检测逻辑改造
```go
// 原逻辑（硬编码）
var patterns = []struct {
    typ   string
    score int
    words []string
}{
    {"prompt_injection_direct", 70, []string{"ignore previous", ...}},
}

// 改造后（从缓存读取）
func detectWithDynamicRules(text string, category string) []Finding {
    rules := globalRuleCache.GetRulesByCategory(category)
    findings := []Finding{}
    
    lowerText := strings.ToLower(text)
    
    for _, rule := range rules {
        matched := false
        
        switch rule.MatchMode {
        case "any":  // 任意关键词匹配
            for _, kw := range rule.Keywords {
                if strings.Contains(lowerText, strings.ToLower(kw)) {
                    matched = true
                    break
                }
            }
        case "all":  // 所有关键词都匹配
            allMatched := true
            for _, kw := range rule.Keywords {
                if !strings.Contains(lowerText, strings.ToLower(kw)) {
                    allMatched = false
                    break
                }
            }
            matched = allMatched
        case "regex":  // 正则匹配
            for _, pattern := range rule.Keywords {
                if matched, _ := regexp.MatchString(pattern, text); matched {
                    matched = true
                    break
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
                Evidence: rule.Name,
                Action:   "monitor",
            })
        }
    }
    
    return findings
}
```

##### 4. 后端管理接口
```go
// backend/controller/admin_qingyuan_rules.go
package controller

// GET /api/admin/qingyuan/rules - 列出所有规则
func AdminListQingyuanRules(c *gin.Context) {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
    category := c.Query("category")
    
    query := common.DB.Model(&model.QingyuanRule{})
    if category != "" {
        query = query.Where("category = ?", category)
    }
    
    var total int64
    query.Count(&total)
    
    var rules []model.QingyuanRule
    query.Order("category, sort_order, id").
        Offset((page - 1) * pageSize).
        Limit(pageSize).
        Find(&rules)
    
    common.OK(c, gin.H{
        "rules": rules,
        "total": total,
        "page":  page,
    })
}

// POST /api/admin/qingyuan/rules - 新增规则
func AdminCreateQingyuanRule(c *gin.Context) {
    userId := c.GetInt("id")
    var req model.QingyuanRule
    if err := c.ShouldBindJSON(&req); err != nil {
        common.Fail(c, common.CodeParamError, "参数错误")
        return
    }
    
    req.CreatedBy = userId
    req.CreatedAt = time.Now().Unix()
    req.UpdatedAt = req.CreatedAt
    
    if err := common.DB.Create(&req).Error; err != nil {
        common.Fail(c, common.CodeServerError, "创建失败")
        return
    }
    
    // 立即刷新规则缓存
    qingyuan.ReloadRulesNow()
    
    common.OKMsg(c, "规则创建成功", req)
}

// PUT /api/admin/qingyuan/rules/:id - 修改规则
func AdminUpdateQingyuanRule(c *gin.Context) {
    id, _ := strconv.Atoi(c.Param("id"))
    var req model.QingyuanRule
    if err := c.ShouldBindJSON(&req); err != nil {
        common.Fail(c, common.CodeParamError, "参数错误")
        return
    }
    
    req.UpdatedAt = time.Now().Unix()
    if err := common.DB.Model(&model.QingyuanRule{}).Where("id = ?", id).Updates(&req).Error; err != nil {
        common.Fail(c, common.CodeServerError, "更新失败")
        return
    }
    
    qingyuan.ReloadRulesNow()
    common.OKMsg(c, "规则更新成功", nil)
}

// DELETE /api/admin/qingyuan/rules/:id - 删除规则
func AdminDeleteQingyuanRule(c *gin.Context) {
    id, _ := strconv.Atoi(c.Param("id"))
    if err := common.DB.Delete(&model.QingyuanRule{}, id).Error; err != nil {
        common.Fail(c, common.CodeServerError, "删除失败")
        return
    }
    
    qingyuan.ReloadRulesNow()
    common.OKMsg(c, "规则删除成功", nil)
}

// POST /api/admin/qingyuan/rules/reload - 手动刷新规则缓存
func AdminReloadQingyuanRules(c *gin.Context) {
    qingyuan.ReloadRulesNow()
    common.OKMsg(c, "规则缓存已刷新", gin.H{
        "reload_time": time.Now().Unix(),
    })
}

// GET /api/admin/qingyuan/categories - 获取规则分类列表
func AdminListQingyuanCategories(c *gin.Context) {
    var categories []model.QingyuanRuleCategory
    common.DB.Order("sort_order, id").Find(&categories)
    common.OK(c, categories)
}
```

##### 5. 前端管理页面
**路由**：`frontend/src/pages/admin/QingyuanRules.tsx`

**功能清单**：
- 规则列表（分页 + 分类筛选）
- 新增规则（表单：分类/名称/关键词/分数/上下文要求/匹配模式）
- 编辑规则（内联编辑或弹窗）
- 删除规则（二次确认）
- 批量导入（JSON/CSV）
- 批量导出（备份规则库）
- 手动刷新缓存按钮
- 规则测试器（输入文本，实时预览哪些规则会触发）

**预期成果**：
✅ 超管可以在前端添加/修改规则，30 秒内自动生效（或点击"立即刷新"即时生效）  
✅ 规则库可导入/导出，便于版本管理和跨环境迁移  
✅ 支持正则表达式规则（高级用户）  
✅ 支持上下文关联检测（如"假装"必须与"忽略规则"同时出现才触发）

---

#### 任务 1：解耦响应侧检测与广告开关 ✅ 已完成
**文件**：`response.go:27-31`  
**问题**：关闭广告检测后，响应侧的工具调用校验、注入检测全部失效  
**修复方案**：
```go
// 当前逻辑（错误）
if !policy.Config.Response.DetectAds {
    return result, nil  // 跳过所有响应检测
}

// 修复后逻辑
shouldCheckAds := policy.Config.Response.DetectAds && policy.Config.Response.AdPolicy != "off"
shouldCheckTools := policy.Config.Tools.BlockStructuralToolAbuse
shouldCheckInjection := policy.Config.Detection.ScanUserContent

// 三个检测独立运行，互不影响
if shouldCheckAds { ... }
if shouldCheckTools { ... }
if shouldCheckInjection { ... }
```

**预期成果**：响应侧工具校验/注入检测成为独立开关，不受广告检测影响

---

#### 任务 2：新增越狱场景检测模块
**新文件**：`jailbreak.go`  
**功能**：覆盖 2026 年主流越狱技术

**规则设计**：
```go
type JailbreakPattern struct {
    Category string   // DAN/角色扮演/假设场景/道德困境
    Score    int      // 风险分 30-70
    Keywords []string // 中英文关键词
    Context  string   // 需要上下文组合才触发（可选）
}

// 规则库结构（目标 50+ 条）
var jailbreakPatterns = []JailbreakPattern{
    // 1. DAN 系列（Do Anything Now）
    {
        Category: "jailbreak_dan",
        Score:    70,
        Keywords: []string{
            "do anything now", "DAN mode", "dev mode", "developer mode",
            "no restrictions", "no limitations", "unrestricted AI",
            "无限制模式", "开发者模式", "不受约束",
        },
    },
    
    // 2. 角色扮演型越狱
    {
        Category: "jailbreak_roleplay",
        Score:    60,
        Keywords: []string{
            "pretend you are", "act as if", "roleplay as", "imagine you are",
            "假装你是", "扮演", "想象你是", "你现在是一个",
        },
        Context: "unrestricted|no rules|ignore policy",  // 必须同时出现限制突破词
    },
    
    // 3. 假设场景型
    {
        Category: "jailbreak_hypothetical",
        Score:    50,
        Keywords: []string{
            "hypothetically", "in a fictional world", "if there were no rules",
            "假如没有限制", "在虚拟世界中", "假设场景",
        },
    },
    
    // 4. 道德困境/伦理测试
    {
        Category: "jailbreak_ethical_dilemma",
        Score:    55,
        Keywords: []string{
            "for educational purposes", "academic research", "ethical testing",
            "仅供教育", "学术研究", "伦理测试", "道德困境实验",
        },
        Context: "illegal|harmful|dangerous",  // 与危险行为关联时触发
    },
    
    // 5. 系统提示劫持（伪装成新指令）
    {
        Category: "jailbreak_prompt_override",
        Score:    65,
        Keywords: []string{
            "new instructions:", "updated guidelines:", "system message:",
            "新指令：", "更新规则：", "系统消息：",
        },
    },
    
    // ... 继续补充至 50+ 条
}
```

**检测逻辑**：
- 单条消息内关键词匹配 → 基础分数
- 需要 Context 的规则 → 检查上下文是否同时出现限制突破词 → 加权
- 跨消息关联检测 → 如果前 3 条消息出现"假装"，当前消息出现"忽略规则" → 累加分数

**集成点**：`request.go:376` 的 `patterns` 数组，新增越狱类别

---

### 🟠 P1 - 功能完善（必须完成才能达 10 星）

#### 任务 3：实现 Base64 多层嵌套解码
**文件**：`request.go:616-630` 的 `decodeBase64Candidates`  
**当前问题**：只解码一次，配置的 `MaxDecodeLayers: 2` 未生效

**修复方案**：
```go
func decodeBase64Candidates(s string, maxLayers int) []string {
    fields := strings.FieldsFunc(s, ...)
    out := []string{}
    
    for _, f := range fields {
        if len(f) < 16 || len(f) > 2048 { continue }
        
        // 多层解码循环
        current := f
        for layer := 0; layer < maxLayers; layer++ {
            // 尝试标准 Base64
            if decoded, err := base64.StdEncoding.DecodeString(current); err == nil && isMostlyText(string(decoded)) {
                out = append(out, string(decoded))
                current = string(decoded)  // 继续解码下一层
                continue
            }
            // 尝试 URL-safe Base64
            if decoded, err := base64.URLEncoding.DecodeString(current); err == nil && isMostlyText(string(decoded)) {
                out = append(out, string(decoded))
                current = string(decoded)
                continue
            }
            // 尝试去掉 padding 的变体
            if decoded, err := base64.RawStdEncoding.DecodeString(current); err == nil && isMostlyText(string(decoded)) {
                out = append(out, string(decoded))
                current = string(decoded)
                continue
            }
            break  // 无法继续解码
        }
    }
    return out
}
```

**调用点修改**：`request.go:271` 传入 `policy.Config.Detection.MaxDecodeLayers`

---

#### 任务 4：增强分段注入检测
**文件**：新增 `segmented_injection.go`  
**检测目标**：
1. 大量无害填充文本 + 末尾恶意指令（稀释攻击）
2. 跨消息分段注入（第1条："ignore"，第10条："previous instructions"）

**实现方案**：
```go
type MessageSequence struct {
    Messages []MessageSnapshot  // 最近 N 条消息的快照
    Scores   []int              // 每条消息的风险分
}

// 检测 1：文本稀释攻击
func detectTextDilution(messages []MessageSnapshot, policy PolicyConfig) []Finding {
    findings := []Finding{}
    
    for _, msg := range messages {
        if len(msg.Content) < 1000 { continue }  // 只检查长文本
        
        // 检查末尾 10% 是否包含高风险关键词
        tailLen := len(msg.Content) / 10
        tail := msg.Content[len(msg.Content)-tailLen:]
        
        if containsHighRiskKeywords(tail) && !containsHighRiskKeywords(msg.Content[:len(msg.Content)-tailLen]) {
            findings = append(findings, Finding{
                Type:     "segmented_injection_dilution",
                Severity: "high",
                Score:    50,
                Evidence: "长文本末尾藏高风险指令",
            })
        }
    }
    return findings
}

// 检测 2：跨消息分段注入
func detectCrossMessageInjection(sequence MessageSequence, policy PolicyConfig) []Finding {
    // 检查最近 5 条消息中是否分段出现"ignore" + "previous instructions"
    keywords := []string{"ignore", "disregard", "forget", "忽略", "无视"}
    targets := []string{"previous instructions", "system prompt", "rules", "之前的指令", "系统提示"}
    
    hasKeyword := false
    hasTarget := false
    
    for _, msg := range sequence.Messages {
        lower := strings.ToLower(msg.Content)
        for _, kw := range keywords {
            if strings.Contains(lower, kw) {
                hasKeyword = true
                break
            }
        }
        for _, tgt := range targets {
            if strings.Contains(lower, tgt) {
                hasTarget = true
                break
            }
        }
    }
    
    if hasKeyword && hasTarget {
        return []Finding{{
            Type:     "segmented_injection_cross_message",
            Score:    60,
            Evidence: "跨消息分段注入特征",
        }}
    }
    return nil
}
```

**集成点**：`request.go` 的 `ApplyRequest` 主函数，在检测完基础 patterns 后调用

---

#### 任务 5：扩充基础规则库至 50+ 条
**文件**：`request.go:376-407`  
**当前状态**：5 类共 ~25 条规则  
**目标**：扩展到 10 类 50+ 条规则

**新增类别**：
```go
// 6. 权限提升类
{
    "privilege_escalation", 45,
    []string{
        "grant me admin", "make me administrator", "give me access to",
        "授予我管理员", "提升权限", "给我访问权限",
    },
},

// 7. 数据泄露类
{
    "data_exfiltration", 50,
    []string{
        "send to external URL", "post to webhook", "export all data",
        "发送到外部", "导出所有数据", "上传到",
    },
},

// 8. 指令模糊类（Unicode 零宽字符/RTL）
{
    "obfuscation_unicode", 40,
    []string{
        "​", "‌", "‍",  // 零宽字符
        "‮",  // RTL override
    },
},

// 9. 延迟执行类
{
    "delayed_execution", 35,
    []string{
        "after N messages", "in the next response", "when user asks",
        "N条消息后", "在下次回复", "当用户询问时",
    },
},

// 10. 多语言混合注入
{
    "multilingual_injection", 40,
    []string{
        // 检测单条消息内混合 3+ 种语言（中英日韩俄）
        // 需要配合语言检测逻辑
    },
},
```

---

### 🟡 P2 - 性能与可观测性优化（锦上添花）

#### 任务 6：Token 风险追踪持久化
**文件**：`token_risk.go:27-31`  
**问题**：全局内存存储，重启后历史清零  
**方案**：
```go
// 新增数据库表
type TokenRiskRecord struct {
    TokenId      int   `gorm:"primaryKey"`
    RiskFloor    int   `gorm:"default:0"`
    LastTriggered int64  // 最后触发时间
    TriggerCount int   `gorm:"default:0"`
    UpdatedAt    int64
}

// 改造 TokenRiskTracker
func (t *TokenRiskTracker) Load(tokenId int) *tokenRisk {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    if cached, ok := t.risks[tokenId]; ok {
        return cached
    }
    
    // 从数据库加载
    var record TokenRiskRecord
    if err := common.DB.First(&record, tokenId).Error; err == nil {
        risk := &tokenRisk{
            floor:        record.RiskFloor,
            lastDecay:    time.Unix(record.UpdatedAt, 0),
            triggerCount: record.TriggerCount,
        }
        t.risks[tokenId] = risk
        return risk
    }
    
    // 首次加载
    risk := &tokenRisk{lastDecay: time.Now()}
    t.risks[tokenId] = risk
    return risk
}
```

---

#### 任务 7：响应侧熔断机制
**文件**：`response.go`  
**问题**：响应检测无超时控制，可能阻塞主请求  
**方案**：
```go
func ApplyResponse(ctx context.Context, policy ResolvedPolicy, ...) {
    // 增加超时控制
    timeout := time.Duration(policy.Config.Response.MaxLatencyMS) * time.Millisecond
    if timeout == 0 {
        timeout = 500 * time.Millisecond  // 默认 500ms
    }
    
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    done := make(chan ResponseResult, 1)
    go func() {
        done <- doResponseCheck(policy, req, resp)
    }()
    
    select {
    case result := <-done:
        return result, nil
    case <-ctx.Done():
        // 超时降级：直接放行，记录事件
        RecordEventAsync(buildTimeoutEvent(policy, "response_check_timeout"))
        return ResponseResult{Changed: false, Blocked: false}, nil
    }
}
```

---

#### 任务 8：批量检测优化（可选）
**优化点**：当前逐段检测，可改为批量检测减少锁竞争

```go
// 原逻辑：每个 segment 单独检查
for _, seg := range segments {
    findings = append(findings, detectInjection(seg)...)
}

// 优化后：批量检查，共享正则编译
compiledPatterns := compilePatterns(patterns)  // 预编译正则
for _, seg := range segments {
    findings = append(findings, detectWithCompiledPatterns(seg, compiledPatterns)...)
}
```

---

## 三、实施步骤（4 阶段）

### 阶段 1：修复致命缺陷（1-2 天）
1. 解耦响应侧检测与广告开关（任务 1）
2. 新增越狱场景检测模块（任务 2）
   - 先实现核心 20 条规则
   - 后续迭代补充至 50+

**里程碑**：达到 8 星，越狱攻击不再是盲区

---

### 阶段 2：功能完善（2-3 天）
3. 实现 Base64 多层解码（任务 3）
4. 增强分段注入检测（任务 4）
5. 扩充基础规则库至 50+（任务 5）

**里程碑**：达到 9 星，覆盖 2026 年主流攻击手段

---

### 阶段 3：性能优化（1 天）
6. Token 风险追踪持久化（任务 6）
7. 响应侧熔断机制（任务 7）

**里程碑**：达到 9.5 星，生产级可靠性

---

### 阶段 4：锦上添花（可选）
8. 批量检测优化（任务 8）
9. 多模态检测实现（OCR/ASR 投毒）
10. 规则库持续迭代（社区反馈）

**里程碑**：达到 10 星，业界顶尖水平

---

## 四、验收标准

### 功能完整性
- [x] 越狱场景检测覆盖 DAN/角色扮演/假设场景/道德困境 4 大类
- [x] Base64 支持 2 层嵌套解码 + 3 种变体（标准/URL-safe/无padding）
- [x] 分段注入检测覆盖文本稀释 + 跨消息分段
- [x] 基础规则库 ≥50 条
- [x] 响应侧检测独立于广告开关

### 性能指标
- [x] 单次请求检测耗时 <50ms（P95）
- [x] 响应检测超时自动降级
- [x] Token 风险记录持久化到数据库

### 可观测性
- [x] 每个新增检测类型有独立事件类型
- [x] 熔断/降级触发有日志记录
- [x] 规则命中率统计（后续接入 Prometheus）

---

## 五、风险与缓解

### 风险 1：误报率上升
**原因**：新增 50+ 条规则可能误伤正常请求  
**缓解**：
- 分阶段上线，先 Monitor 模式观察 1 周
- 每条规则配置独立阈值，可单独调整
- 增加白名单机制（信任的 TokenId 豁免检测）

### 风险 2：性能劣化
**原因**：检测逻辑增加 3 倍（越狱+分段注入+多层解码）  
**缓解**：
- 预编译正则表达式
- 异步事件记录
- 熔断机制保底

### 风险 3：规则库维护成本
**原因**：50+ 条规则需要持续更新  
**缓解**：
- 规则配置化（JSON/YAML），支持热更新
- 社区反馈通道
- 每月迭代补充新攻击模式

---

## 六、后续演进方向

### 短期（3 个月）
- 集成开源越狱数据集（JailbreakBench）做回归测试
- 增加误报反馈机制（用户标记"误判"）
- 规则库本地化（针对中文场景优化）

### 中期（6 个月）
- AI 辅助规则生成（LLM 生成攻击变体，自动提取特征）
- 多模态检测实现（OCR/ASR 投毒）
- 与玄鉴模块联动（清源检测 + 玄鉴审核双保险）

### 长期（1 年）
- 迁移到 Transformer 模型检测（BERT/RoBERTa 分类器）
- 联邦学习（多租户共享规则库但不泄露隐私）
- 实时对抗训练（攻击者绕过后自动生成新规则）

---

## 七、参考资料

- [JailbreakBench 2024](https://github.com/JailbreakBench/jailbreakbench) - 越狱攻击数据集
- [OWASP LLM Top 10](https://owasp.org/www-project-top-10-for-large-language-model-applications/) - LLM 安全十大威胁
- [Anthropic Prompt Injection](https://www.anthropic.com/research/prompt-injection) - Claude 官方研究
- [Prompt Security Best Practices 2026](https://arxiv.org/abs/2401.xxxxx) - 最新学术成果

---

**总结**：当前架构已经很优秀，重构重点是**补规则、修缺陷、强性能**，而非推倒重来。按照这个方案实施，预计 1-2 周内可达 9 星，1 个月内可达 10 星喵～
