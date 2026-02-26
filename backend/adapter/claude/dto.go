package claude

import "encoding/json"

// --- Request Types ---

type ClaudeRequest struct {
	Model         string            `json:"model"`
	Messages      []ClaudeMessage   `json:"messages"`
	System        json.RawMessage   `json:"system,omitempty"`
	MaxTokens     int               `json:"max_tokens,omitempty"`
	StopSequences []string          `json:"stop_sequences,omitempty"`
	Temperature   *float64          `json:"temperature,omitempty"`
	TopP          *float64          `json:"top_p,omitempty"`
	TopK          int               `json:"top_k,omitempty"`
	Stream        bool              `json:"stream,omitempty"`
	Tools         []ClaudeTool      `json:"tools,omitempty"`
	ToolChoice    json.RawMessage   `json:"tool_choice,omitempty"`
	Thinking      *ClaudeThinking   `json:"thinking,omitempty"`
	Metadata      json.RawMessage   `json:"metadata,omitempty"`
}

type ClaudeMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type ClaudeTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
	Type        string          `json:"type,omitempty"`
}

type ClaudeThinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

// --- Content Block Types ---

type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	Source    *ImageSource    `json:"source,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
}

type ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// --- Response Types ---

type ClaudeResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	StopSequence string         `json:"stop_sequence"`
	Usage        ClaudeUsage    `json:"usage"`
}

type ClaudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

type ClaudeError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type ClaudeErrorResponse struct {
	Type  string      `json:"type"`
	Error ClaudeError `json:"error"`
}

// --- Stream Event Types ---

type ClaudeStreamEvent struct {
	Type         string          `json:"type"`
	Message      *ClaudeResponse `json:"message,omitempty"`
	Index        int             `json:"index,omitempty"`
	ContentBlock *ContentBlock   `json:"content_block,omitempty"`
	Delta        *ClaudeDelta    `json:"delta,omitempty"`
	Usage        *ClaudeUsage    `json:"usage,omitempty"`
}

type ClaudeDelta struct {
	Type        string          `json:"type,omitempty"`
	Text        string          `json:"text,omitempty"`
	StopReason  string          `json:"stop_reason,omitempty"`
	PartialJSON string          `json:"partial_json,omitempty"`
	Thinking    string          `json:"thinking,omitempty"`
}
