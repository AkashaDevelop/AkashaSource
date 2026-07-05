package common

import (
	"encoding/json"
	"sync"
)

var (
	ModelRatio            = make(map[string]float64)
	CompletionRatio       = make(map[string]float64)
	GroupRatio            = make(map[string]float64)
	ImageRatio            = make(map[string]float64)      // 图像 token 倍率
	AudioRatio            = make(map[string]float64)      // 音频输入 token 倍率
	AudioCompletionRatio  = make(map[string]float64)      // 音频输出 token 倍率
	ModelPrice            = make(map[string]float64)      // 按次计费价格，单位美元/次
	PricingLock           sync.RWMutex
)

var defaultGroupRatio = map[string]float64{
	"default": 1,
	"vip":     1,
	"svip":    1,
}

// Default values for initialization if DB is empty
var defaultModelRatio = map[string]float64{
	// GPT-3.5
	"gpt-3.5-turbo":          0.5,
	"gpt-3.5-turbo-0125":     0.5,
	"gpt-3.5-turbo-1106":     1.0,
	"gpt-3.5-turbo-instruct": 1.5,
	"gpt-3.5-turbo-16k":      1.5,

	// GPT-4
	"gpt-4":                  30,
	"gpt-4-0613":             30,
	"gpt-4-32k":              60,
	"gpt-4-turbo":            10,
	"gpt-4-turbo-2024-04-09": 10,
	"gpt-4-turbo-preview":    10,
	"gpt-4-0125-preview":     10,
	"gpt-4-1106-preview":     10,
	"gpt-4-vision-preview":   10,

	// GPT-4o
	"gpt-4o":                 5,
	"gpt-4o-2024-05-13":      5,
	"gpt-4o-mini":            0.15,
	"gpt-4o-mini-2024-07-18": 0.15,

	// Embedding
	"text-embedding-ada-002": 0.1,
	"text-embedding-3-small": 0.02,
	"text-embedding-3-large": 0.13,

	// Claude
	"claude-instant-1":           0.8,
	"claude-2":                   8,
	"claude-2.0":                 8,
	"claude-2.1":                 8,
	"claude-3-opus-20240229":     15,
	"claude-3-sonnet-20240229":   3,
	"claude-3-haiku-20240307":    0.25,
	"claude-3-5-sonnet-20240620": 3,

	// Gemini
	"gemini-pro":         0.125,
	"gemini-pro-vision":  0.25,
	"gemini-1.5-pro":     3.5,
	"gemini-1.5-flash":   0.35,
	"gemini-ultra":       10.0,
	"text-embedding-004": 0.05,

	// DALL-E (Image) - Ratio here represents cost per image relative to 1k tokens of GPT-3.5?
	// No, image pricing is fixed per image usually.
	// But our logic uses: quota = promptTokens * ratio.
	// If we set promptTokens = quota cost directly in relay, we should set ratio to 1.
	"dall-e-2": 1,
	"dall-e-3": 1,

	// GPT-4.1
	"gpt-4.1":             5,
	"gpt-4.1-mini":        0.2,
	"gpt-4.1-nano":        0.05,
	"gpt-4.5-preview":     7.5,

	// o1/o3 series
	"o1":                  60,
	"o1-pro":              75,
	"o3":                  30,
	"o3-pro":              75,
	"o3-mini":             1.1,
	"o4-mini":             1.1,

	// GPT-5
	"gpt-5":               5,
	"gpt-5-mini":          0.25,

	// Claude 3.5/4
	"claude-3-5-sonnet-20241022": 3,
	"claude-3-5-haiku-20241022":  0.4,
	"claude-3-7-sonnet-20250219": 3,
	"claude-sonnet-4-20250514":   6,
	"claude-opus-4-20250514":     15,
	"claude-opus-4-1-20250805":   15,

	// Gemini 2
	"gemini-2.0-flash":           0.1,
	"gemini-2.0-flash-thinking":  0.35,
	"gemini-2.5-pro":             1.25,
	"gemini-2.5-flash":           0.075,

	// Deepseek
	"deepseek-chat":              0.14,
	"deepseek-reasoner":          0.28,

	// Qwen
	"qwen-max":                   1.2,
	"qwen-plus":                  0.14,
	"qwen-turbo":                 0.05,

	// Groq
	"llama-3.3-70b-versatile":    0.13,
	"llama-3.1-8b-instant":       0.02,
}

var defaultCompletionRatio = map[string]float64{
	// GPT-4
	"gpt-4":       2,
	"gpt-4-32k":   2,
	"gpt-4-turbo": 3,
	"gpt-4o":      3,
	"gpt-4o-mini": 4,
	"o1-preview":  4,
	"o1-mini":     4,

	// Claude
	"claude-3-opus-20240229":     5,
	"claude-3-sonnet-20240229":   5,
	"claude-3-haiku-20240307":    5,
	"claude-3-5-sonnet-20240620": 5,

	// Gemini
	"gemini-1.5-pro":   3,
	"gemini-1.5-flash": 3,
	"gemini-ultra":     3,

	// New models
	"gpt-4.1":                     4,
	"gpt-4.1-mini":                4,
	"gpt-4.1-nano":                4,
	"gpt-4.5-preview":             2,
	"o1":                          4,
	"o3":                          4,
	"o3-pro":                      4,
	"o4-mini":                     3,
	"gpt-5":                       4,
	"gpt-5-mini":                  4,
	"claude-3-5-sonnet-20241022":  5,
	"claude-3-5-haiku-20241022":   5,
	"claude-3-7-sonnet-20250219":  5,
	"claude-sonnet-4-20250514":    5,
	"claude-opus-4-20250514":      5,
	"claude-opus-4-1-20250805":    5,
	"gemini-2.0-flash":            3,
	"gemini-2.5-pro":              4,
	"gemini-2.5-flash":            4,
	"deepseek-reasoner":           2,
}

var defaultImageRatio = map[string]float64{
	"gpt-4o":      2.07,
	"gpt-4o-mini": 3.38,
	"gpt-4-turbo": 2.07,
}

var defaultAudioRatio = map[string]float64{
	"gpt-4o":               2.5,
	"gpt-4o-mini":          2.5,
	"gpt-4o-audio-preview": 2.5,
}

var defaultAudioCompletionRatio = map[string]float64{
	"gpt-4o":               8,
	"gpt-4o-mini":          8,
	"gpt-4o-audio-preview": 8,
}

var defaultModelPrice = map[string]float64{
	"dall-e-2":           0.02,
	"dall-e-3":           0.04,
	"dall-e-3-hd":        0.08,
	"mj_imagine":         0.1,
	"mj_variation":       0.08,
	"mj_reroll":          0.08,
	"mj_blend":           0.08,
	"mj_upscale":         0.05,
	"mj_describe":        0.03,
	"suno_music":         0.1,
	"suno_lyrics":        0.02,
	"kling_text2video":   0.35,
	"kling_image2video":  0.4,
	"jimeng_text2video":  0.3,
	"jimeng_image2video": 0.35,
}

func GetModelRatio(modelName string) float64 {
	PricingLock.RLock()
	defer PricingLock.RUnlock()
	if ratio, ok := ModelRatio[modelName]; ok {
		return ratio
	}
	return 1.0
}

// GetCacheRatio returns the discount ratio for cached tokens (default 0.5 = 50% discount)
func GetCacheRatio() float64 {
	return 0.5
}

func GetCompletionRatio(modelName string) float64 {
	PricingLock.RLock()
	defer PricingLock.RUnlock()
	if ratio, ok := CompletionRatio[modelName]; ok {
		return ratio
	}
	return 1.0
}

func GetGroupRatio(group string) float64 {
	PricingLock.RLock()
	defer PricingLock.RUnlock()
	if ratio, ok := GroupRatio[group]; ok {
		return ratio
	}
	return 1.0
}

func GetImageRatio(modelName string) float64 {
	PricingLock.RLock()
	defer PricingLock.RUnlock()
	if ratio, ok := ImageRatio[modelName]; ok {
		return ratio
	}
	return 1.0
}

func GetAudioRatio(modelName string) float64 {
	PricingLock.RLock()
	defer PricingLock.RUnlock()
	if ratio, ok := AudioRatio[modelName]; ok {
		return ratio
	}
	return 1.0
}

func GetAudioCompletionRatio(modelName string) float64 {
	PricingLock.RLock()
	defer PricingLock.RUnlock()
	if ratio, ok := AudioCompletionRatio[modelName]; ok {
		return ratio
	}
	return 1.0
}

func GetModelPrice(modelName string) (float64, bool) {
	PricingLock.RLock()
	defer PricingLock.RUnlock()
	if price, ok := ModelPrice[modelName]; ok {
		return price, true
	}
	return 0, false
}

func ModelRatio2JSONString() string {
	PricingLock.RLock()
	defer PricingLock.RUnlock()
	jsonBytes, _ := json.Marshal(ModelRatio)
	return string(jsonBytes)
}

func CompletionRatio2JSONString() string {
	PricingLock.RLock()
	defer PricingLock.RUnlock()
	jsonBytes, _ := json.Marshal(CompletionRatio)
	return string(jsonBytes)
}

func GroupRatio2JSONString() string {
	PricingLock.RLock()
	defer PricingLock.RUnlock()
	jsonBytes, _ := json.Marshal(GroupRatio)
	return string(jsonBytes)
}

func ImageRatio2JSONString() string {
	PricingLock.RLock()
	defer PricingLock.RUnlock()
	jsonBytes, _ := json.Marshal(ImageRatio)
	return string(jsonBytes)
}

func AudioRatio2JSONString() string {
	PricingLock.RLock()
	defer PricingLock.RUnlock()
	jsonBytes, _ := json.Marshal(AudioRatio)
	return string(jsonBytes)
}

func ModelPrice2JSONString() string {
	PricingLock.RLock()
	defer PricingLock.RUnlock()
	jsonBytes, _ := json.Marshal(ModelPrice)
	return string(jsonBytes)
}

func UpdatePricing(modelRatioStr string, completionRatioStr string, groupRatioStr string, imageRatioStr string, audioRatioStr string, modelPriceStr string) {
	PricingLock.Lock()
	defer PricingLock.Unlock()

	if modelRatioStr != "" {
		var newRatio map[string]float64
		if err := json.Unmarshal([]byte(modelRatioStr), &newRatio); err == nil {
			ModelRatio = newRatio
		}
	} else {
		// Initialize with defaults if empty
		ModelRatio = defaultModelRatio
	}

	if completionRatioStr != "" {
		var newRatio map[string]float64
		if err := json.Unmarshal([]byte(completionRatioStr), &newRatio); err == nil {
			CompletionRatio = newRatio
		}
	} else {
		CompletionRatio = defaultCompletionRatio
	}

	if groupRatioStr != "" {
		var newRatio map[string]float64
		if err := json.Unmarshal([]byte(groupRatioStr), &newRatio); err == nil {
			GroupRatio = newRatio
		}
	} else {
		GroupRatio = defaultGroupRatio
	}

	if imageRatioStr != "" {
		var newRatio map[string]float64
		if err := json.Unmarshal([]byte(imageRatioStr), &newRatio); err == nil {
			ImageRatio = newRatio
		}
	} else {
		ImageRatio = defaultImageRatio
	}

	if audioRatioStr != "" {
		var newRatio map[string]float64
		if err := json.Unmarshal([]byte(audioRatioStr), &newRatio); err == nil {
			AudioRatio = newRatio
		}
	} else {
		AudioRatio = defaultAudioRatio
	}

	if modelPriceStr != "" {
		var newPrice map[string]float64
		if err := json.Unmarshal([]byte(modelPriceStr), &newPrice); err == nil {
			ModelPrice = newPrice
		}
	} else {
		ModelPrice = defaultModelPrice
	}
}

func UpdateGroupRatio(groupRatioStr string) {
	PricingLock.Lock()
	defer PricingLock.Unlock()
	if groupRatioStr != "" {
		var newRatio map[string]float64
		if err := json.Unmarshal([]byte(groupRatioStr), &newRatio); err == nil {
			GroupRatio = newRatio
		}
	} else {
		GroupRatio = defaultGroupRatio
	}
}

func UpdateImageRatio(imageRatioStr string) {
	PricingLock.Lock()
	defer PricingLock.Unlock()
	if imageRatioStr != "" {
		var newRatio map[string]float64
		if err := json.Unmarshal([]byte(imageRatioStr), &newRatio); err == nil {
			ImageRatio = newRatio
		}
	} else {
		ImageRatio = defaultImageRatio
	}
}

func UpdateAudioRatio(audioRatioStr string) {
	PricingLock.Lock()
	defer PricingLock.Unlock()
	if audioRatioStr != "" {
		var newRatio map[string]float64
		if err := json.Unmarshal([]byte(audioRatioStr), &newRatio); err == nil {
			AudioRatio = newRatio
		}
	} else {
		AudioRatio = defaultAudioRatio
	}
}

func UpdateModelPrice(modelPriceStr string) {
	PricingLock.Lock()
	defer PricingLock.Unlock()
	if modelPriceStr != "" {
		var newPrice map[string]float64
		if err := json.Unmarshal([]byte(modelPriceStr), &newPrice); err == nil {
			ModelPrice = newPrice
		}
	} else {
		ModelPrice = defaultModelPrice
	}
}
