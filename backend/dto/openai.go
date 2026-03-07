package dto

type OpenAIRequest struct {
	Model            string        `json:"model"`
	Messages         []interface{} `json:"messages"`
	Stream           bool          `json:"stream"`
	MaxTokens        int           `json:"max_tokens,omitempty"`
	Temperature      float64       `json:"temperature,omitempty"`
	TopP             float64       `json:"top_p,omitempty"`
	N                int           `json:"n,omitempty"`
	Stop             interface{}   `json:"stop,omitempty"`
	PresencePenalty  float64       `json:"presence_penalty,omitempty"`
	FrequencyPenalty float64       `json:"frequency_penalty,omitempty"`
	User             string        `json:"user,omitempty"`
	Tools            []any         `json:"tools,omitempty"`
	ToolChoice       any           `json:"tool_choice,omitempty"`

	// Image Generation
	Prompt string `json:"prompt,omitempty"`
	Size   string `json:"size,omitempty"`

	// Embeddings
	Input any `json:"input,omitempty"`

	// Audio (TTS & Whisper)
	Voice          string  `json:"voice,omitempty"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Speed          float64 `json:"speed,omitempty"`

	// Rerank
	Query     string   `json:"query,omitempty"`
	Documents []string `json:"documents,omitempty"`
	TopN      int      `json:"top_n,omitempty"`

	// Reasoning
	ReasoningEffort string `json:"reasoning_effort,omitempty"`

	// Internal
	RawBody      []byte `json:"-"`
	ContentType  string `json:"-"`
	ThinkingMode bool   `json:"-"` // Parsed from -thinking suffix
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	CachedTokens     int `json:"cached_tokens,omitempty"`

	// Claude format
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`

	// DeepSeek/Qwen format
	PromptCacheHitTokens int `json:"prompt_cache_hit_tokens,omitempty"`

	// Detailed token breakdown
	PromptTokensDetails *PromptTokensDetails `json:"prompt_tokens_details,omitempty"`
	InputTokensDetails  *InputTokensDetails  `json:"input_tokens_details,omitempty"`
}

type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"`
}

type InputTokensDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"`
}

type OpenAIError struct {
	Message string      `json:"message"`
	Type    string      `json:"type"`
	Param   string      `json:"param"`
	Code    interface{} `json:"code"`
}

type OpenAIErrorResponse struct {
	Error OpenAIError `json:"error"`
}
