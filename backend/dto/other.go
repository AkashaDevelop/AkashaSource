package dto

// --- Embedding Response ---

type EmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  Usage           `json:"usage"`
}

// --- Image Generation ---

type ImageData struct {
	Url           string `json:"url,omitempty"`
	B64Json       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type ImageGenerationResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
}

// --- Legacy Completions ---

type CompletionChoice struct {
	Text         string `json:"text"`
	Index        int    `json:"index"`
	FinishReason string `json:"finish_reason"`
}

type CompletionResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []CompletionChoice `json:"choices"`
	Usage   Usage              `json:"usage"`
}

// --- Rerank ---

type RerankRequest struct {
	Documents       []interface{} `json:"documents"`
	Query           string        `json:"query"`
	Model           string        `json:"model"`
	TopN            *int          `json:"top_n,omitempty"`
	ReturnDocuments *bool         `json:"return_documents,omitempty"`
	MaxChunkPerDoc  *int          `json:"max_chunk_per_doc,omitempty"`
	OverlapTokens   *int          `json:"overlap_tokens,omitempty"`
}

type RerankResult struct {
	Document       interface{} `json:"document,omitempty"`
	Index          int         `json:"index"`
	RelevanceScore float64     `json:"relevance_score"`
}

type RerankResponse struct {
	Object  string         `json:"object"`
	Results []RerankResult `json:"results"`
	Model   string         `json:"model"`
	Usage   Usage          `json:"usage"`
}
