package context_sanitizer

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"STfreApi/dto"
)

// StreamProcessor 流式响应处理器
// 设计文档 8.2 & 8.4: 流式响应净化
type StreamProcessor struct {
	policy         ResolvedPolicy
	requestTools   []interface{}
	tailBuffer     *bytes.Buffer
	tailBufferSize int
	aggregated     strings.Builder
	findings       []Finding
}

// NewStreamProcessor 创建流式处理器
func NewStreamProcessor(policy ResolvedPolicy, requestTools []interface{}) *StreamProcessor {
	tailBufferSize := 2048 // 默认保留最后 2KB
	if policy.Config.Response.StreamTailBufferSize > 0 {
		tailBufferSize = policy.Config.Response.StreamTailBufferSize
	}

	return &StreamProcessor{
		policy:         policy,
		requestTools:   requestTools,
		tailBuffer:     bytes.NewBuffer(make([]byte, 0, tailBufferSize)),
		tailBufferSize: tailBufferSize,
		findings:       make([]Finding, 0),
	}
}

// ProcessChunk 处理单个 SSE 数据块
func (sp *StreamProcessor) ProcessChunk(chunk []byte) ([]byte, error) {
	// 解析 SSE 事件
	scanner := bufio.NewScanner(bytes.NewReader(chunk))
	var output bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()

		// SSE 格式: data: {...}
		if !strings.HasPrefix(line, "data: ") {
			output.WriteString(line)
			output.WriteString("\n")
			continue
		}

		dataContent := strings.TrimPrefix(line, "data: ")

		// 流结束标记
		if dataContent == "[DONE]" {
			output.WriteString(line)
			output.WriteString("\n")
			continue
		}

		// 解析 JSON
		var delta dto.ChatCompletionStreamResponse
		if err := json.Unmarshal([]byte(dataContent), &delta); err != nil {
			// 无法解析,原样输出
			output.WriteString(line)
			output.WriteString("\n")
			continue
		}

		// 提取内容增量
		if len(delta.Choices) > 0 {
			deltaContent := delta.Choices[0].Delta.Content

			// 聚合内容用于后处理
			sp.aggregated.WriteString(deltaContent)

			// 更新 tail buffer
			sp.updateTailBuffer(deltaContent)
		}

		// 原样输出 (一期不修改流)
		output.WriteString(line)
		output.WriteString("\n")
	}

	return output.Bytes(), scanner.Err()
}

// updateTailBuffer 更新尾部缓冲区
func (sp *StreamProcessor) updateTailBuffer(content string) {
	if content == "" {
		return
	}

	sp.tailBuffer.WriteString(content)

	// 超出大小限制时裁剪前面的内容
	if sp.tailBuffer.Len() > sp.tailBufferSize {
		excess := sp.tailBuffer.Len() - sp.tailBufferSize
		sp.tailBuffer.Next(excess) // 丢弃前面的字节
	}
}

// Finalize 流结束后的最终检测
func (sp *StreamProcessor) Finalize() []Finding {
	fullContent := sp.aggregated.String()

	// 对完整聚合内容做检测
	if fullContent != "" {
		// 广告检测
		if sp.policy.Config.Response.DetectAds {
			_, adFindings := detectAndStripSuffixAd(fullContent, sp.policy)
			sp.findings = append(sp.findings, adFindings...)
		}

		// 工具调用注入检测
		toolFindings := detectToolCallsInText(fullContent, sp.policy)
		sp.findings = append(sp.findings, toolFindings...)

		// 响应注入检测
		injectionFindings := detectResponseInjection(fullContent, sp.policy)
		sp.findings = append(sp.findings, injectionFindings...)
	}

	return sp.findings
}

// GetTailBuffer 获取尾部缓冲区内容
func (sp *StreamProcessor) GetTailBuffer() string {
	return sp.tailBuffer.String()
}

// GetAggregatedContent 获取聚合的完整内容
func (sp *StreamProcessor) GetAggregatedContent() string {
	return sp.aggregated.String()
}

// ProcessSSEStream 处理整个 SSE 流 (包装函数)
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

	findings := processor.Finalize()

	return &StreamResult{
		ProcessedBody:      allOutput.Bytes(),
		Findings:           findings,
		RiskScore:          maxScore(findings),
		AggregatedContent:  processor.GetAggregatedContent(),
		TailBufferContent:  processor.GetTailBuffer(),
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
// 在聚合后的文本中检测
func detectStreamToolCallInjection(aggregatedContent string, requestTools []interface{}) []Finding {
	findings := []Finding{}

	// 检测文本中的工具调用模式
	toolPatterns := detectToolCallsInText(aggregatedContent, ResolvedPolicy{
		Config: DefaultConfig(),
	})
	findings = append(findings, toolPatterns...)

	// 如果请求中没有声明 tools,但输出包含工具调用语法,风险更高
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

// CleanStreamTailAd 清理流式输出尾部广告 (二期功能)
func CleanStreamTailAd(tailBuffer string, policy ResolvedPolicy) (string, []Finding) {
	if !policy.Config.Response.DetectAds || policy.Config.Response.AdPolicy != "strip_known_suffix" {
		return tailBuffer, nil
	}

	return detectAndStripSuffixAd(tailBuffer, policy)
}
