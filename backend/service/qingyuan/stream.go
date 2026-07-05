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
// 这套"攒尾巴"的核心逻辑已经抽到 stream_core.go 的 TailBufferCore 里，
// 这里只负责 OpenAI SSE JSON 的具体解析/序列化，方便 Claude/Responses/Gemini 各自复用同一份核心。

// StreamProcessor 流式响应处理器（OpenAI SSE 格式）
type StreamProcessor struct {
	core         *TailBufferCore
	requestTools []interface{}
	tailBuffer   *bytes.Buffer // 仅用于日志/事件片段展示
}

// NewStreamProcessor 创建流式处理器
func NewStreamProcessor(policy ResolvedPolicy, requestTools []interface{}) *StreamProcessor {
	core := NewTailBufferCore(policy)
	return &StreamProcessor{
		core:         core,
		requestTools: requestTools,
		tailBuffer:   bytes.NewBuffer(make([]byte, 0, core.bufferSize)),
	}
}

// ProcessChunk 处理单个 SSE 数据块，返回"可以安全放行"的字节
func (sp *StreamProcessor) ProcessChunk(chunk []byte) ([]byte, error) {
	scanner := bufio.NewScanner(bytes.NewReader(chunk))
	var immediate bytes.Buffer

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
		}
		sp.updateTailBuffer(deltaContent)

		raw := []byte(line + "\n")
		for _, released := range sp.core.Feed(deltaContent, raw) {
			immediate.Write(released.([]byte))
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
	if sp.tailBuffer.Len() > sp.core.bufferSize {
		excess := sp.tailBuffer.Len() - sp.core.bufferSize
		sp.tailBuffer.Next(excess)
	}
}

// Finalize 流结束后的最终检测与尾部放行
// 返回值: (还需要补写给客户端的安全尾部字节, 本次流检测到的 Finding 列表)
func (sp *StreamProcessor) Finalize() ([]byte, []Finding) {
	released, _, findings := sp.core.Finalize(sp.requestTools)
	var out bytes.Buffer
	for _, r := range released {
		out.Write(r.([]byte))
	}
	return out.Bytes(), findings
}

// GetTailBuffer 获取尾部缓冲区内容(用于事件日志片段展示)
func (sp *StreamProcessor) GetTailBuffer() string {
	return sp.tailBuffer.String()
}

// GetAggregatedContent 获取聚合的完整内容
func (sp *StreamProcessor) GetAggregatedContent() string {
	return sp.core.AggregatedText()
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
