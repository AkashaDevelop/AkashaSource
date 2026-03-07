package model

type ModelConfig struct {
	Id                int     `json:"id" gorm:"primaryKey;autoIncrement"`
	ModelName         string  `json:"model_name" gorm:"uniqueIndex;type:varchar(200)"`
	DisplayName       string  `json:"display_name" gorm:"type:varchar(200)"`
	Category          string  `json:"category" gorm:"type:varchar(50);index"` // chat, embedding, image, audio, etc.
	InputRatio        float64 `json:"input_ratio" gorm:"default:1"`
	OutputRatio       float64 `json:"output_ratio" gorm:"default:1"`
	UpstreamInputPrice  float64 `json:"upstream_input_price" gorm:"default:0"`   // 上游输入价格（元/百万tokens）
	UpstreamOutputPrice float64 `json:"upstream_output_price" gorm:"default:0"`  // 上游输出价格（元/百万tokens）
	MaxContext        int     `json:"max_context" gorm:"default:4096"`
	Enabled           bool    `json:"enabled" gorm:"default:true"`
	CreatedAt         int64   `json:"created_at"`
}
