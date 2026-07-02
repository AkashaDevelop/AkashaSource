package model

// CustomChannelConfig 自定义渠道配置（可视化配置，无需JSON）
type CustomChannelConfig struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string `json:"name" gorm:"type:varchar(255)"` // 配置名称
	Description string `json:"description" gorm:"type:text"`  // 配置描述

	// 适配器类型
	AdapterType string `json:"adapter_type" gorm:"type:varchar(50);default:'openai_compatible'"` // openai_compatible, anthropic_compatible, custom

	// 请求配置
	RequestMethod       string `json:"request_method" gorm:"type:varchar(10);default:'POST'"` // HTTP 方法
	RequestEndpoint     string `json:"request_endpoint" gorm:"type:varchar(500)"`             // 请求端点，如 /v1/chat/completions
	RequestContentType  string `json:"request_content_type" gorm:"type:varchar(100);default:'application/json'"`
	AuthType            string `json:"auth_type" gorm:"type:varchar(50);default:'bearer'"` // bearer, api_key, custom
	AuthHeaderName      string `json:"auth_header_name" gorm:"type:varchar(100);default:'Authorization'"`
	AuthHeaderTemplate  string `json:"auth_header_template" gorm:"type:varchar(255);default:'Bearer {key}'"` // 支持 {key} 占位符
	CustomHeaders       string `json:"custom_headers" gorm:"type:text"`                                      // JSON: [{"key":"X-Custom","value":"test"}]

	// 请求体字段映射（OpenAI -> 目标API）
	FieldModel       string `json:"field_model" gorm:"type:varchar(100);default:'model'"`
	FieldMessages    string `json:"field_messages" gorm:"type:varchar(100);default:'messages'"`
	FieldTemperature string `json:"field_temperature" gorm:"type:varchar(100);default:'temperature'"`
	FieldMaxTokens   string `json:"field_max_tokens" gorm:"type:varchar(100);default:'max_tokens'"`
	FieldStream      string `json:"field_stream" gorm:"type:varchar(100);default:'stream'"`
	FieldTopP        string `json:"field_top_p" gorm:"type:varchar(100);default:'top_p'"`
	FieldStop        string `json:"field_stop" gorm:"type:varchar(100);default:'stop'"`

	// 响应字段路径（使用 JSONPath 语法）
	ResponseContentPath      string `json:"response_content_path" gorm:"type:varchar(255);default:'choices.0.message.content'"`
	ResponseUsagePath        string `json:"response_usage_path" gorm:"type:varchar(255);default:'usage'"`
	ResponsePromptTokensPath string `json:"response_prompt_tokens_path" gorm:"type:varchar(255);default:'usage.prompt_tokens'"`
	ResponseCompletionTokensPath string `json:"response_completion_tokens_path" gorm:"type:varchar(255);default:'usage.completion_tokens'"`
	ResponseTotalTokensPath  string `json:"response_total_tokens_path" gorm:"type:varchar(255);default:'usage.total_tokens'"`
	ResponseErrorPath        string `json:"response_error_path" gorm:"type:varchar(255);default:'error.message'"`

	// 流式响应配置
	StreamEnabled         int    `json:"stream_enabled" gorm:"default:1"`                                                 // 是否支持流式
	StreamDataPrefix      string `json:"stream_data_prefix" gorm:"type:varchar(50);default:'data: '"`                    // SSE 数据前缀
	StreamEndMarker       string `json:"stream_end_marker" gorm:"type:varchar(50);default:'[DONE]'"`                     // 流结束标记
	StreamContentPath     string `json:"stream_content_path" gorm:"type:varchar(255);default:'choices.0.delta.content'"` // 流式内容路径

	// 功能支持
	SupportFunctionCall int `json:"support_function_call" gorm:"default:0"` // 是否支持函数调用
	SupportVision       int `json:"support_vision" gorm:"default:0"`        // 是否支持视觉模型
	SupportEmbedding    int `json:"support_embedding" gorm:"default:0"`     // 是否支持 Embedding

	// 其他配置
	Timeout       int    `json:"timeout" gorm:"default:120"`           // 请求超时时间（秒）
	RetryCount    int    `json:"retry_count" gorm:"default:3"`         // 重试次数
	RetryInterval int    `json:"retry_interval" gorm:"default:1"`      // 重试间隔（秒）
	IsPublic      int    `json:"is_public" gorm:"default:0"`           // 是否公开（供其他用户使用）
	CreatorId     int    `json:"creator_id"`                           // 创建者ID
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
}
