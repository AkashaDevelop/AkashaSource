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

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
}

type ChatCompletionStreamResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
			Role    string `json:"role,omitempty"`
		} `json:"delta"`
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type ImageGenerationResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		Url           string `json:"url,omitempty"`
		B64Json       string `json:"b64_json,omitempty"`
		RevisedPrompt string `json:"revised_prompt,omitempty"`
	} `json:"data"`
}
