package common

import (
	"encoding/json"
	"sync"
)

var (
	ModelRatio      = make(map[string]float64)
	CompletionRatio = make(map[string]float64)
	PricingLock     sync.RWMutex
)

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
}

func GetModelRatio(modelName string) float64 {
	PricingLock.RLock()
	defer PricingLock.RUnlock()
	if ratio, ok := ModelRatio[modelName]; ok {
		return ratio
	}
	return 1.0
}

func GetCompletionRatio(modelName string) float64 {
	PricingLock.RLock()
	defer PricingLock.RUnlock()
	if ratio, ok := CompletionRatio[modelName]; ok {
		return ratio
	}
	return 1.0
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

func UpdatePricing(modelRatioStr string, completionRatioStr string) {
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
}
