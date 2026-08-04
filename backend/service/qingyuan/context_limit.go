package qingyuan

import "strings"

// 宸汐清源 · 窗量尺 (◕‿◕)
//
// "上下文快满了"是一个重要的风险信号——攻击者喜欢用超长无害文本把系统提示词
// 挤出模型的注意力范围，再在末尾塞进真正的指令。
//
// 但之前这把尺子是写死的 128000，于是就闹了笑话：
// Claude 有 200k、Gemini 有 1M 的窗口，用户老老实实喂 15 万 token 的长文档，
// 在这把短尺子上一量——"哎呀超过 80% 了，危险！"——于是这次请求里**所有**命中
// 都被无差别放大 1.2~1.5 倍，好好的长文摘要就这么被拦下来了。
//
// 现在按模型名认窗口，尺子对了，长上下文用户才不会被冤枉喵～

// defaultContextWindow 认不出模型时的保守取值
const defaultContextWindow = 128000

// contextWindowRules 模型名关键字 → 上下文窗口大小
//
// 顺序敏感：越具体的关键字必须排在越前面（比如 gpt-4.1 要排在 gpt-4 前面），
// 匹配时取第一个命中的规则。
var contextWindowRules = []struct {
	keyword string
	window  int
}{
	// ── Gemini 系：百万级窗口的代表 ────────────────────────────────────
	{"gemini-1.5-pro", 2000000},
	{"gemini-2.5", 1048576},
	{"gemini-2.0", 1048576},
	{"gemini-1.5", 1048576},
	{"gemini", 1048576},

	// ── Claude 系：200k 起步，部分 sonnet 支持 1M ──────────────────────
	{"claude-sonnet-4", 1000000},
	{"claude-opus", 200000},
	{"claude-sonnet", 200000},
	{"claude-haiku", 200000},
	{"claude-3", 200000},
	{"claude", 200000},

	// ── OpenAI 系 ───────────────────────────────────────────────────
	{"gpt-4.1", 1047576},
	{"o3", 200000},
	{"o1", 200000},
	{"gpt-4o", 128000},
	{"gpt-4-turbo", 128000},
	{"gpt-4", 8192},
	{"gpt-3.5", 16385},

	// ── 国内模型 ────────────────────────────────────────────────────
	{"deepseek", 128000},
	{"qwen-long", 10000000},
	{"qwen", 131072},
	{"moonshot-v1-128k", 128000},
	{"moonshot-v1-32k", 32768},
	{"moonshot", 131072},
	{"glm-4", 128000},
	{"glm", 128000},
	{"ernie", 128000},
	{"hunyuan", 256000},
	{"kimi", 131072},
	{"yi-", 200000},
	{"minimax", 1000000},
	{"doubao", 256000},
}

// resolveContextWindow 按模型名推断上下文窗口大小
//
// mapped 是实际转发给上游的模型名，requested 是用户请求里写的名字；
// 优先信 mapped（那才是真正在跑的模型），认不出来再退回 requested。
func resolveContextWindow(mapped, requested string) int {
	if w := matchContextWindow(mapped); w > 0 {
		return w
	}
	if w := matchContextWindow(requested); w > 0 {
		return w
	}
	return defaultContextWindow
}

func matchContextWindow(modelName string) int {
	if modelName == "" {
		return 0
	}
	lower := strings.ToLower(modelName)
	for _, rule := range contextWindowRules {
		if strings.Contains(lower, rule.keyword) {
			return rule.window
		}
	}
	return 0
}
