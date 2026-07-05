package qingyuan

import "strings"

// 宸汐清源 · 尾部缓冲核心引擎（格式无关版）
// stream.go 里原来的 StreamProcessor 是绑死 OpenAI SSE JSON 形状的，
// 这里把"攒住尾巴、判断完再放行"这套逻辑抽成不认识任何具体协议格式的核心，
// 各协议（Claude/Responses/Gemini）自己的流式处理器只管喂文本增量 + 自己的待发送单元，
// 核心只负责算"现在能不能安全放行"，序列化/怎么发出去仍然交给各协议自己～

// pendingChunk 一个尚未放行的待发送单元，emit 是调用方自定义的"发送载荷"（比如一整条 SSE 事件字节）
type pendingChunk struct {
	emit       any
	contentLen int
}

// TailBufferCore 格式无关的尾部延迟放行引擎
type TailBufferCore struct {
	policy       ResolvedPolicy
	bufferSize   int
	delayEnabled bool
	aggregated   strings.Builder
	pending      []pendingChunk
	pendingLen   int
}

// NewTailBufferCore 创建核心引擎，delayEnabled 的判断跟 stream.go 的 StreamProcessor 保持一致
func NewTailBufferCore(policy ResolvedPolicy) *TailBufferCore {
	bufferSize := 2048
	if policy.Config.Response.StreamTailBufferSize > 0 {
		bufferSize = policy.Config.Response.StreamTailBufferSize
	}
	delayEnabled := policy.Config.Response.DetectAds &&
		policy.Config.Response.AdPolicy == "strip_known_suffix" &&
		policy.Policy != nil &&
		(policy.Policy.Mode == ModeProtect || policy.Policy.Mode == ModeStrict)

	return &TailBufferCore{
		policy:       policy,
		bufferSize:   bufferSize,
		delayEnabled: delayEnabled,
	}
}

// Feed 喂入一段文本增量和它对应的"待发送单元"，返回现在可以安全放行的单元列表（可能为空，表示先攒着）
func (t *TailBufferCore) Feed(deltaText string, emit any) []any {
	t.aggregated.WriteString(deltaText)

	if !t.delayEnabled {
		return []any{emit}
	}

	t.pending = append(t.pending, pendingChunk{emit: emit, contentLen: len(deltaText)})
	t.pendingLen += len(deltaText)

	var released []any
	for t.pendingLen > t.bufferSize && len(t.pending) > 0 {
		front := t.pending[0]
		t.pending = t.pending[1:]
		t.pendingLen -= front.contentLen
		released = append(released, front.emit)
	}
	return released
}

// Finalize 流结束后跑完整检测（广告尾巴/工具调用注入/响应注入），
// 返回还需要放行的单元列表、完整聚合文本、以及本次检测到的所有 Finding
func (t *TailBufferCore) Finalize(requestTools []interface{}) (toRelease []any, fullText string, findings []Finding) {
	fullText = t.aggregated.String()
	if fullText == "" {
		return t.releaseTail(0), fullText, nil
	}

	if t.policy.Config.Response.DetectAds {
		cleaned, adFindings := detectAndStripSuffixAd(fullText, t.policy)
		findings = append(findings, adFindings...)
		if len(adFindings) > 0 && t.delayEnabled {
			findings = append(findings, detectStreamToolCallInjection(fullText, requestTools)...)
			findings = append(findings, detectResponseInjection(fullText, t.policy)...)
			return t.releaseTail(len(fullText) - len(cleaned)), fullText, findings
		}
	}
	findings = append(findings, detectStreamToolCallInjection(fullText, requestTools)...)
	findings = append(findings, detectResponseInjection(fullText, t.policy)...)
	return t.releaseTail(0), fullText, findings
}

// releaseTail 放行 pending 队列，stripLen>0 时从队尾丢弃刚好覆盖这个长度的单元（即广告尾巴本身）
func (t *TailBufferCore) releaseTail(stripLen int) []any {
	if len(t.pending) == 0 {
		return nil
	}
	drop := 0
	if stripLen > 0 {
		accumulated := 0
		for i := len(t.pending) - 1; i >= 0 && accumulated < stripLen; i-- {
			accumulated += t.pending[i].contentLen
			drop++
		}
	}
	keep := t.pending[:len(t.pending)-drop]
	out := make([]any, 0, len(keep))
	for _, c := range keep {
		out = append(out, c.emit)
	}
	t.pending = nil
	t.pendingLen = 0
	return out
}

// AggregatedText 取当前已聚合的完整文本（流还没结束时也能取，用于日志展示）
func (t *TailBufferCore) AggregatedText() string {
	return t.aggregated.String()
}
