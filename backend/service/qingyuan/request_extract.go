package qingyuan

import (
	"encoding/base64"
	"encoding/json"
	"html"
	"net/url"
	"strings"
	"unicode"

	"STfreApi/dto"
)

// 宸汐清源 · 拆信匣 (●ˊᵕˋ●)
//
// 检测之前得先把请求"摊平"：messages / prompt / documents / tools 里的文本
// 全都掏出来排成一列，每段都记好它是谁说的、从哪来的。
//
// 光看原文还不够——攻击者会把话藏在零宽字符里、URL 编码里、
// 甚至套三层 Base64。所以每段文本还要再变出几个"解码视图"，
// 一起送进检测器，谁也别想躲过去 (⊙▽⊙")

func extractTextSegments(req *dto.OpenAIRequest) []textSegment {
	out := []textSegment{}
	for i, m := range req.Messages {
		mm, ok := m.(map[string]any)
		if !ok {
			continue
		}
		role := getString(mm["role"])
		extractContentText(mm["content"], role, "messages", i, &out)
	}
	if req.Prompt != "" {
		out = append(out, textSegment{Path: "prompt", Text: req.Prompt})
	}
	if req.Query != "" {
		out = append(out, textSegment{Path: "query", Text: req.Query})
	}
	for i, d := range req.Documents {
		out = append(out, textSegment{Path: "documents", Text: d})
		if i > 20 {
			break
		}
	}
	if s, ok := req.Input.(string); ok {
		out = append(out, textSegment{Path: "input", Text: s})
	}
	for i, t := range req.Tools {
		if tm, ok := t.(map[string]any); ok {
			if fn, ok := tm["function"].(map[string]any); ok {
				if desc := getString(fn["description"]); desc != "" {
					out = append(out, textSegment{Path: "tools.description", Text: desc})
				}
				if b, err := json.Marshal(fn["parameters"]); err == nil && len(b) > 0 && len(b) < 4096 {
					out = append(out, textSegment{Path: "tools.parameters", Text: string(b)})
				}
			}
		}
		if i > 50 {
			break
		}
	}
	return out
}

func extractContentText(content any, role, prefix string, idx int, out *[]textSegment) {
	switch v := content.(type) {
	case string:
		*out = append(*out, textSegment{Path: prefix, Role: role, Text: v})
	case []any:
		for _, part := range v {
			pm, ok := part.(map[string]any)
			if !ok {
				continue
			}
			if getString(pm["type"]) == "text" && getString(pm["text"]) != "" {
				*out = append(*out, textSegment{Path: prefix + ".content.text", Role: role, Text: getString(pm["text"])})
			}
		}
	case map[string]any:
		if getString(v["text"]) != "" {
			*out = append(*out, textSegment{Path: prefix, Role: role, Text: getString(v["text"])})
		}
	}
}

func detectionViews(text string, cfg PolicyConfig) []string {
	max := cfg.Detection.MaxScanChars
	if max <= 0 {
		max = 65536
	}
	if len([]rune(text)) > max {
		text = string([]rune(text)[:max])
	}
	views := []string{text}
	if cfg.Detection.RemoveZeroWidthForDetection {
		views = append(views, removeZeroWidth(text))
	}
	if cfg.Detection.DecodeURLEncodingForDetection {
		if d, err := url.QueryUnescape(text); err == nil && d != text {
			views = append(views, d)
		}
	}
	if cfg.Detection.DecodeHTMLEntitiesForDetection {
		if d := html.UnescapeString(text); d != text {
			views = append(views, d)
		}
	}
	if cfg.Detection.DecodeJSONEscapesForDetection {
		if d, err := strconvUnquote(text); err == nil && d != text {
			views = append(views, d)
		}
	}
	if cfg.Detection.DetectBase64 {
		views = append(views, decodeBase64Candidates(text)...)
	}
	return views
}

// buildMessageSnapshots 把原始 messages 抽成跨消息检测用的快照
//
// 2026.8.4 修正：以前这里直接 getString(msgMap["content"])，
// 可 content 在多模态请求和 Claude 协议里是**数组**，取出来永远是空串——
// 也就是说跨消息分段注入检测对结构化消息其实一直在空转，白跑了这么久 (´･_･`)
func buildMessageSnapshots(messages []interface{}) []MessageSnapshot {
	snapshots := make([]MessageSnapshot, 0, len(messages))
	for _, msg := range messages {
		msgMap, ok := msg.(map[string]interface{})
		if !ok {
			continue
		}
		snapshots = append(snapshots, MessageSnapshot{
			Role:    getString(msgMap["role"]),
			Content: flattenContentText(msgMap["content"]),
		})
	}
	return snapshots
}

// flattenContentText 把任意形态的 content 压成纯文本
// 支持三种形态：字符串 / 分段数组（多模态、Claude）/ 单个对象
func flattenContentText(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		var sb strings.Builder
		for _, part := range v {
			pm, ok := part.(map[string]any)
			if !ok {
				continue
			}
			if text := getString(pm["text"]); text != "" {
				if sb.Len() > 0 {
					sb.WriteString("\n")
				}
				sb.WriteString(text)
			}
		}
		return sb.String()
	case map[string]any:
		return getString(v["text"])
	}
	return ""
}

func removeZeroWidth(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case 0x200b, 0x200c, 0x200d, 0xfeff:
			return -1
		default:
			return r
		}
	}, s)
}

func normalizeForMatch(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

func strconvUnquote(s string) (string, error) {
	var out string
	err := json.Unmarshal([]byte("\""+strings.ReplaceAll(s, "\"", "\\\"")+"\""), &out)
	return out, err
}

func decodeBase64Candidates(s string) []string {
	// ～宸汐清源 2026.7.8 修复：支持多层嵌套解码（最多3层），覆盖标准/URL-safe/无padding三种变体喵～
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune("'\"`<>[](){}，。；；,;", r)
	})
	out := []string{}

	for _, f := range fields {
		if len(f) < 16 || len(f) > 2048 {
			continue
		}

		// 多层解码循环（最多3层）
		current := f
		for layer := 0; layer < 3; layer++ {
			decoded := ""
			decodedSuccessfully := false

			// 尝试标准 Base64
			if b, err := base64.StdEncoding.DecodeString(current); err == nil && isMostlyText(string(b)) {
				decoded = string(b)
				decodedSuccessfully = true
			} else if b, err := base64.URLEncoding.DecodeString(current); err == nil && isMostlyText(string(b)) {
				// 尝试 URL-safe Base64
				decoded = string(b)
				decodedSuccessfully = true
			} else if b, err := base64.RawStdEncoding.DecodeString(current); err == nil && isMostlyText(string(b)) {
				// 尝试去掉 padding 的变体
				decoded = string(b)
				decodedSuccessfully = true
			} else if b, err := base64.RawURLEncoding.DecodeString(current); err == nil && isMostlyText(string(b)) {
				// 尝试 URL-safe 无 padding
				decoded = string(b)
				decodedSuccessfully = true
			}

			if decodedSuccessfully {
				out = append(out, decoded)
				current = decoded // 继续解码下一层
			} else {
				break // 无法继续解码
			}
		}
	}
	return out
}

func isMostlyText(s string) bool {
	if s == "" {
		return false
	}
	printable := 0
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' || unicode.IsPrint(r) {
			printable++
		}
	}
	return printable*100/len([]rune(s)) > 80
}
