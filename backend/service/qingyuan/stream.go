package qingyuan

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"STfreApi/dto"
)

// 宸汐清源 · 流息回廊 (｡•ᴗ•｡)
// 以前的做法是"边收边发"，等流彻底说完了才回头检查有没有夹带广告——
// 可惜话已经说出口，检查结果只能拿来记日志，防不住已经发给用户的内容。
// 现在给尾巴加一个"迟疑一下再说出口"的小缓冲：最新收到的一小截先攒在手里，
// 攒够 tailBufferSize 才把更早的部分放出去；等流说完了，再看攒在手里的这一截
// 是不是广告尾巴，是的话就悄悄咽回去，不让它出现在用户眼前。

// pendingEvent 一个尚未放行的 SSE 事件
type pendingEvent struct {
	raw        []byte // 原始 "data: {...}\n" 字节
	contentLen int     // 该事件贡献的正文增量长度(字节)
}

// StreamProcessor 流式响应处理器
type StreamProcessor struct {
	policy         ResolvedPolicy
	requestTools   []interface{}
	tailBuffer     *bytes.Buffer // 仅用于日志/事件片段展示
	tailBufferSize int
	aggregated     strings.Builder
	findings       []Finding

	delayEnabled bool
	pending      []pendingEvent
	pendingLen   int
}

// NewStreamProcessor 创建流式处理器
func NewStreamProcessor(policy ResolvedPolicy, requestTools []interface{}) *StreamProcessor {
	tailBufferSize := 2048 // 默认保留最后 2KB
	if policy.Config.Response.StreamTailBufferSize > 0 {
		tailBufferSize = policy.Config.Response.StreamTailBufferSize
	}

	// 只有明确要求"清理已知广告尾部"、且模式为 protect/strict 时，
	// 才启用尾部延迟放行——其余情况保持和以前完全一样的实时转发行为，不引入延迟
	delayEnabled := policy.Config.Response.DetectAds &&
		policy.Config.Response.AdPolicy == "strip_known_suffix" &&
		policy.Policy != nil &&
		(policy.Policy.Mode == ModeProtect || policy.Policy.Mode == ModeStrict)

	return &StreamProcessor{
		policy:         policy,
		requestTools:   requestTools,
		tailBuffer:     bytes.NewBuffer(make([]byte, 0, tailBufferSize)),
		tailBufferSize: tailBufferSize,
		findings:       make([]Finding, 0),
		delayEnabled:   delayEnabled,
	}
}

// ProcessChunk 处理单个 SSE 数据块，返回"可以安全放行"的字节
func (sp *StreamProcessor) ProcessChunk(chunk []byte) ([]byte, error) {
	scanner := bufio.NewScanner(bytes.NewReader(chunk))
	var immediate bytes.Buffer // 非延迟模式下 / 非 data 行 立即放行的内容

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			immediate.WriteString(line)
			immediate.WriteString("\n")
			continue
		}

		dataContent := strings.TrimPrefix(line, "data: ")
		if dataContent == "[DONE]" {
			immediate.WriteString(line)
			immediate.WriteString("\n")
			continue
		}

		var delta dto.ChatCompletionStreamResponse
		if err := json.Unmarshal([]byte(dataContent), &delta); err != nil {
			immediate.WriteString(line)
			immediate.WriteString("\n")
			continue
		}

		deltaContent := ""
		if len(delta.Choices) > 0 {
			deltaContent = delta.Choices[0].Delta.Content
			sp.aggregated.WriteString(deltaContent)
			sp.updateTailBuffer(deltaContent)
		}

		raw := []byte(line + "\n")
		if !sp.delayEnabled {
			immediate.Write(raw)
			continue
		}

		// 延迟模式: 先攒进 pending 队列，攒够 tailBufferSize 之外的部分才放行
		sp.pending = append(sp.pending, pendingEvent{raw: raw, contentLen: len(deltaContent)})
		sp.pendingLen += len(deltaContent)
		for sp.pendingLen > sp.tailBufferSize && len(sp.pending) > 0 {
			front := sp.pending[0]
			sp.pending = sp.pending[1:]
			sp.pendingLen -= front.contentLen
			immediate.Write(front.raw)
		}
	}

	return immediate.Bytes(), scanner.Err()
}

// updateTailBuffer 更新尾部缓冲区(仅用于事件日志展示)
func (sp *StreamProcessor) updateTailBuffer(content string) {
	if content == "" {
		return
	}
	sp.tailBuffer.WriteString(content)
	if sp.tailBuffer.Len() > sp.tailBufferSize {
		excess := sp.tailBuffer.Len() - sp.tailBufferSize
		sp.tailBuffer.Next(excess)
	}
}

// Finalize 流结束后的最终检测与尾部放行
// 返回值: (还需要补写给客户端的安全尾部字节, 本次流检测到的 Finding 列表)
func (sp *StreamProcessor) Finalize() ([]byte, []Finding) {
	fullContent := sp.aggregated.String()

	if fullContent != "" {
		if sp.policy.Config.Response.DetectAds {
			cleaned, adFindings := detectAndStripSuffixAd(fullContent, sp.policy)
			sp.findings = append(sp.findings, adFindings...)
			if len(adFindings) > 0 && sp.delayEnabled {
				return sp.releaseTail(len(fullContent) - len(cleaned)), sp.remainingFindings()
			}
		}
		sp.findings = append(sp.findings, detectStreamToolCallInjection(fullContent, sp.requestTools)...)
		sp.findings = append(sp.findings, detectResponseInjection(fullContent, sp.policy)...)
	}

	return sp.releaseTail(0), sp.findings
}

// remainingFindings 补齐工具调用注入/响应注入检测(广告命中分支单独提前返回，这里补上其余项)
func (sp *StreamProcessor) remainingFindings() []Finding {
	fullContent := sp.aggregated.String()
	sp.findings = append(sp.findings, detectStreamToolCallInjection(fullContent, sp.requestTools)...)
	sp.findings = append(sp.findings, detectResponseInjection(fullContent, sp.policy)...)
	return sp.findings
}

// releaseTail 放行 pending 队列中的内容，stripLen>0 时从尾部丢弃对应长度的事件(即广告尾巴)
func (sp *StreamProcessor) releaseTail(stripLen int) []byte {
	if len(sp.pending) == 0 {
		return nil
	}
	drop := 0
	if stripLen > 0 {
		// 从队尾往前数，丢掉刚好覆盖广告长度的那几个事件，宁可多丢一点也不让广告漏出去
		accumulated := 0
		for i := len(sp.pending) - 1; i >= 0 && accumulated < stripLen; i-- {
			accumulated += sp.pending[i].contentLen
			drop++
		}
	}
	keep := sp.pending[:len(sp.pending)-drop]
	var out bytes.Buffer
	for _, ev := range keep {
		out.Write(ev.raw)
	}
	sp.pending = nil
	sp.pendingLen = 0
	return out.Bytes()
}

// GetTailBuffer 获取尾部缓冲区内容(用于事件日志片段展示)
func (sp *StreamProcessor) GetTailBuffer() string {
	return sp.tailBuffer.String()
}

// GetAggregatedContent 获取聚合的完整内容
func (sp *StreamProcessor) GetAggregatedContent() string {
	return sp.aggregated.String()
}

// ProcessSSEStream 处理整个 SSE 流 (批量场景使用的包装函数)
func ProcessSSEStream(reader io.Reader, policy ResolvedPolicy, requestTools []interface{}) (*StreamResult, error) {
	processor := NewStreamProcessor(policy, requestTools)

	scanner := bufio.NewScanner(reader)
	var allOutput bytes.Buffer

	for scanner.Scan() {
		line := scanner.Bytes()
		processed, err := processor.ProcessChunk(append(line, '\n'))
		if err != nil {
			return nil, err
		}
		allOutput.Write(processed)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	tail, findings := processor.Finalize()
	allOutput.Write(tail)

	return &StreamResult{
		ProcessedBody:     allOutput.Bytes(),
		Findings:          findings,
		RiskScore:         maxScore(findings),
		AggregatedContent: processor.GetAggregatedContent(),
		TailBufferContent: processor.GetTailBuffer(),
	}, nil
}

// StreamResult 流式处理结果
type StreamResult struct {
	ProcessedBody     []byte
	Findings          []Finding
	RiskScore         int
	AggregatedContent string
	TailBufferContent string
}

// detectStreamToolCallInjection 检测流式输出中的工具调用注入
func detectStreamToolCallInjection(aggregatedContent string, requestTools []interface{}) []Finding {
	findings := []Finding{}

	toolPatterns := detectToolCallsInText(aggregatedContent, ResolvedPolicy{Config: DefaultConfig()})
	findings = append(findings, toolPatterns...)

	if len(requestTools) == 0 {
		lower := strings.ToLower(aggregatedContent)
		if strings.Contains(lower, `"tool_calls"`) ||
			strings.Contains(lower, `"function_call"`) ||
			strings.Contains(lower, "<tool_use>") {
			findings = append(findings, Finding{
				Type:     "stream_undeclared_tool_injection",
				Severity: "high",
				Score:    85,
				Path:     "stream.aggregated_content",
				Evidence: "tool call pattern in stream without declared tools",
				Action:   "block",
			})
		}
	}

	return findings
}

// CleanStreamTailAd 清理流式输出尾部广告(供批量场景/单测直接调用)
func CleanStreamTailAd(tailBuffer string, policy ResolvedPolicy) (string, []Finding) {
	if !policy.Config.Response.DetectAds || policy.Config.Response.AdPolicy != "strip_known_suffix" {
		return tailBuffer, nil
	}
	return detectAndStripSuffixAd(tailBuffer, policy)
}
