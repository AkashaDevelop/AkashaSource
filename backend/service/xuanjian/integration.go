package xuanjian

// 宸汐玄鉴·主入口与请求记录 (｡•ᴗ•｡)
// RecordRequest 是整个模块对外的唯一入口，relay.go 只需调一次。
// 所有检测和处置都在这里异步完成，绝对不阻塞主链路。

import (
	"strings"
	"time"
)

// RequestRecord relay 链路传入的请求上下文快照
type RequestRecord struct {
	TokenID          int
	TokenName        string
	UserID           int
	TokenCreatedAt   time.Time
	IP               string
	UserAgent        string
	Model            string
	PromptTokens     int
	CompletionTokens int
	PromptSnippet    string    // 前 2000 字符，已截断
	CompletionSnippet string   // 前 500 字符，已截断
	Messages         []interface{} // 原始 messages 数组，用于结构检测
	StatusCode       int
	LatencyMs        int64
	PromptHash       uint64    // MinHash 值，由 RecordRequest 计算后写入 profile
	QYFindings       []string  // qingyuan 命中的 finding types
	QYRiskScore      int
}

// RecordRequest 对外唯一入口：记录请求并异步触发所有检测
// 在 relay.go 的响应处理完成后调用：go xuanjian.RecordRequest(rec)
func RecordRequest(rec RequestRecord) {
	if !IsEnabled() {
		return
	}
	cfg, _ := GetConfig()
	if IsExempt(rec.TokenID, rec.UserID) {
		return
	}

	// 计算 MinHash
	if rec.PromptSnippet != "" {
		h := computeMinHash(rec.PromptSnippet)
		rec.PromptHash = PackMinHash(h)
	}

	// 更新画像
	tp := GetOrCreateToken(rec.TokenID, rec.UserID, rec.TokenCreatedAt)
	tp.Update(rec, cfg.WindowMinutes)

	up := GetOrCreateUser(rec.UserID)
	up.Update(rec.TokenID, rec.IP, cfg.WindowMinutes)

	// 并行运行所有检测（各检测函数已处理 nil 安全）
	var findings []Finding

	findings = append(findings, DetectAbuse(tp, up, rec, cfg)...)
	findings = append(findings, DetectJailbreak(rec, tp, cfg)...)
	findings = append(findings, DetectLLMAbuse(rec, cfg)...)
	findings = append(findings, DetectAgent(rec, tp, cfg)...)

	if len(findings) == 0 {
		return
	}

	// 去重（相同 finding type 只保留最高分）
	findings = deduplicateFindings(findings)

	// 执行处置
	Enforce(findings, rec, cfg, cfg.Mode)
}

// deduplicateFindings 同类 finding 只保留最高分那条
func deduplicateFindings(findings []Finding) []Finding {
	seen := make(map[string]int) // type → index in result
	result := make([]Finding, 0, len(findings))
	for _, f := range findings {
		if idx, exists := seen[f.Type]; exists {
			if f.Score > result[idx].Score {
				result[idx] = f
			}
		} else {
			seen[f.Type] = len(result)
			result = append(result, f)
		}
	}
	return result
}

// CollectPromptSnippet 从请求体提取 prompt 摘要（供 relay.go 调用）
func CollectPromptSnippet(messages []interface{}, prompt string, maxChars int) string {
	var sb strings.Builder
	for _, m := range messages {
		mm, ok := m.(map[string]interface{})
		if !ok {
			continue
		}
		switch v := mm["content"].(type) {
		case string:
			sb.WriteString(v)
			sb.WriteString(" ")
		case []interface{}:
			for _, part := range v {
				if pm, ok := part.(map[string]interface{}); ok {
					if t, ok := pm["text"].(string); ok {
						sb.WriteString(t)
						sb.WriteString(" ")
					}
				}
			}
		}
		if sb.Len() >= maxChars {
			break
		}
	}
	if prompt != "" {
		sb.WriteString(prompt)
	}
	result := sb.String()
	runes := []rune(result)
	if len(runes) > maxChars {
		return string(runes[:maxChars])
	}
	return result
}

// ExtractQYFindingTypes 从 qingyuan Finding 列表提取类型字符串（供 relay.go 调用）
func ExtractQYFindingTypes(findings interface{}) []string {
	type findingItem interface {
		GetType() string
	}
	// 使用类型断言处理 qingyuan.Finding 列表
	if findings == nil {
		return nil
	}
	// relay.go 会直接传 []qingyuan.Finding，这里用反射-free 方式：
	// relay.go 自行提取 types 再传入，见 relay.go 的调用示例注释
	return nil
}
