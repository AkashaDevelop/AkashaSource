package aireview

// ～阿卡夏 AI 审核公共调用层～ (｡•ᴗ•｡)
// 宸汐清源和宸汐玄鉴都要用 AI 模型给内容做二次审核，这部分"怎么调渠道、
// 怎么解析返回结果"的逻辑完全一样，抽出来放这里，两边各自拼自己的 prompt 就好啦～
//
// 设计原则：
//   - fail-open：审核服务不可用时放行，不阻塞正常请求
//   - 同步调用：调用方按需自己决定是同步等待还是丢进 goroutine 异步跑
//   - JSON 协议：要求 AI 返回结构化 JSON，便于精确判定

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"STfreApi/common"
	"STfreApi/model"
)

// Result AI 审核结果
type Result struct {
	Pass      bool   // true=放行，false=拦截
	RiskScore int    // 风险分 0-100
	Verdict   string // pass / block
	Reason    string // 判定理由
	Skipped   bool   // true=审核被跳过（配置缺失、服务不可用等），此时 Pass 恒为 true
	Stage     string // 调用方自定义的阶段标记，仅用于日志
}

// Call 调用一次 AI 模型执行审核，stage 只用于日志前缀区分调用方
func Call(channelID int, modelName, systemPrompt, userContent string, timeoutSec, blockScore int, stage string) Result {
	if channelID <= 0 || modelName == "" {
		log.Printf("[aireview] 审核(%s)跳过：未配置审核渠道或模型", stage)
		return Result{Pass: true, Skipped: true, Stage: stage}
	}

	// 从数据库加载审核渠道
	var channel model.Channel
	if err := common.DB.First(&channel, channelID).Error; err != nil {
		log.Printf("[aireview] 审核(%s)失败：审核渠道 %d 不存在: %v", stage, channelID, err)
		return Result{Pass: true, Skipped: true, Stage: stage}
	}
	if channel.Status != model.ChannelStatusActive {
		log.Printf("[aireview] 审核(%s)跳过：审核渠道 %d 未启用", stage, channelID)
		return Result{Pass: true, Skipped: true, Stage: stage}
	}

	if timeoutSec <= 0 {
		timeoutSec = 10
	}

	// 构造 OpenAI 兼容的 chat/completions 请求
	chatReq := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userContent},
		},
		"temperature": 0,
		"max_tokens":  200,
		"stream":      false,
	}
	reqBody, err := json.Marshal(chatReq)
	if err != nil {
		log.Printf("[aireview] 审核(%s)失败：构造请求失败: %v", stage, err)
		return Result{Pass: true, Skipped: true, Stage: stage}
	}

	baseUrl := channel.BaseURL
	if baseUrl == "" {
		baseUrl = "https://api.openai.com"
	}
	baseUrl = strings.TrimSuffix(baseUrl, "/")
	targetURL := fmt.Sprintf("%s/v1/chat/completions", baseUrl)

	httpReq, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(reqBody))
	if err != nil {
		log.Printf("[aireview] 审核(%s)失败：创建请求失败: %v", stage, err)
		return Result{Pass: true, Skipped: true, Stage: stage}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", channel.Key))
	common.ApplyHeaders(httpReq, channel.Headers)

	client := common.NewHTTPClient(channel.Proxy)
	client.Timeout = time.Duration(timeoutSec) * time.Second

	startTime := time.Now()
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("[aireview] 审核(%s)失败：请求超时或错误: %v", stage, err)
		return Result{Pass: true, Skipped: true, Stage: stage}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[aireview] 审核(%s)失败：读取响应失败: %v", stage, err)
		return Result{Pass: true, Skipped: true, Stage: stage}
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("[aireview] 审核(%s)失败：审核渠道返回状态码 %d", stage, resp.StatusCode)
		return Result{Pass: true, Skipped: true, Stage: stage}
	}

	// 解析 OpenAI 格式响应
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &chatResp); err != nil {
		log.Printf("[aireview] 审核(%s)失败：解析响应失败: %v", stage, err)
		return Result{Pass: true, Skipped: true, Stage: stage}
	}
	if len(chatResp.Choices) == 0 {
		log.Printf("[aireview] 审核(%s)失败：AI 返回空结果", stage)
		return Result{Pass: true, Skipped: true, Stage: stage}
	}

	content := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	result := parseResponse(content, stage)

	// 按阈值判定
	if result.RiskScore >= blockScore {
		result.Pass = false
		result.Verdict = "block"
	} else {
		result.Pass = true
		result.Verdict = "pass"
	}

	elapsed := time.Since(startTime)
	log.Printf("[aireview] 审核(%s)完成：score=%d verdict=%s 耗时=%s", stage, result.RiskScore, result.Verdict, elapsed.Round(time.Millisecond))

	return result
}

// JSONFormatSuffix JSON 返回格式要求，调用方自定义 prompt 后应自动追加此后缀
const JSONFormatSuffix = `

请以严格的 JSON 格式返回审核结果，不要包含任何其他文字或 markdown 标记：
{"risk_score": 0到100的整数, "verdict": "pass"或"block", "reason": "简短说明"}

- risk_score：风险等级，0=完全安全，100=极度危险
- verdict：pass=放行，block=拦截
- reason：简短说明判断依据（不超过100字）`

// parseResponse 从 AI 返回的文本中解析 JSON 审核结果
func parseResponse(content, stage string) Result {
	result := Result{Stage: stage}

	// 尝试直接解析 JSON
	content = stripMarkdownFence(content)

	var verdict struct {
		RiskScore int    `json:"risk_score"`
		Verdict   string `json:"verdict"`
		Reason    string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(content), &verdict); err != nil {
		// 尝试从文本中提取 JSON 片段
		jsonStr := extractJSON(content)
		if jsonStr == "" {
			log.Printf("[aireview] 审核(%s)解析失败：无法提取 JSON，原始内容: %s", stage, truncateForLog(content, 200))
			result.Pass = true
			result.Skipped = true
			return result
		}
		if err := json.Unmarshal([]byte(jsonStr), &verdict); err != nil {
			log.Printf("[aireview] 审核(%s)解析失败: %v", stage, err)
			result.Pass = true
			result.Skipped = true
			return result
		}
	}

	result.RiskScore = verdict.RiskScore
	result.Verdict = verdict.Verdict
	result.Reason = verdict.Reason
	return result
}

// stripMarkdownFence 去除 AI 返回内容外层的 markdown 代码围栏
func stripMarkdownFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// 去掉开头的 ``` 或 ```json
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
		s = strings.TrimSpace(s)
	}
	return s
}

// extractJSON 从文本中提取第一个 JSON 对象
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	if start == -1 {
		return ""
	}
	end := strings.LastIndex(s, "}")
	if end == -1 || end < start {
		return ""
	}
	return s[start : end+1]
}

// truncateForLog 截断日志内容，防止过长
func truncateForLog(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
