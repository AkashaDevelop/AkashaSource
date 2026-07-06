package model

type ModelConfig struct {
	Id                  int     `json:"id" gorm:"primaryKey;autoIncrement"`
	ModelName           string  `json:"model_name" gorm:"uniqueIndex;type:varchar(200)"`
	DisplayName         string  `json:"display_name" gorm:"type:varchar(200)"`
	Category            string  `json:"category" gorm:"type:varchar(50);index"` // chat, embedding, image, audio, etc.
	InputRatio          float64 `json:"input_ratio" gorm:"default:1"`
	OutputRatio         float64 `json:"output_ratio" gorm:"default:1"`
	UpstreamInputPrice  float64 `json:"upstream_input_price" gorm:"default:0"`  // 上游输入价格（元/百万tokens）
	UpstreamOutputPrice float64 `json:"upstream_output_price" gorm:"default:0"` // 上游输出价格（元/百万tokens）
	// ～模型定价管理页新增的几个维度，喵～
	CacheRatio           float64 `json:"cache_ratio" gorm:"default:0.5"`          // 缓存 token 折扣倍率
	ImageRatio           float64 `json:"image_ratio" gorm:"default:1"`            // 图像 token 倍率
	AudioRatio           float64 `json:"audio_ratio" gorm:"default:1"`            // 音频输入 token 倍率
	AudioCompletionRatio float64 `json:"audio_completion_ratio" gorm:"default:1"` // 音频输出 token 倍率
	IsFixedPrice         bool    `json:"is_fixed_price" gorm:"default:false"`     // 是否按次计费
	FixedPrice           float64 `json:"fixed_price" gorm:"default:0"`            // 按次计费单价（美元/次）
	MaxContext           int     `json:"max_context" gorm:"default:4096"`
	Enabled              bool    `json:"enabled" gorm:"default:true"`
	CreatedAt            int64   `json:"created_at"`
}
