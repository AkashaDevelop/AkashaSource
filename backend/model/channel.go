package model

type Channel struct {
	Id           int     `json:"id" gorm:"primaryKey;autoIncrement"`
	Type         int     `json:"type" gorm:"default:1"` // 1: OpenAI, 2: API2D, 3: Azure, 等
	Key          string  `json:"key" gorm:"type:text"`  // API Key
	BaseURL      string  `json:"base_url" gorm:"column:base_url;default:''"`
	Name         string  `json:"name" gorm:"index"`
	Group        string  `json:"group" gorm:"default:'default'"` // 支持多组
	Models       string  `json:"models" gorm:"type:text"`        // 支持的模型 (逗号分隔)
	ModelMapping string  `json:"model_mapping" gorm:"type:text"` // 模型映射 JSON 字符串
	Priority     int     `json:"priority" gorm:"default:10"`     // 值越大优先级越高
	Weight       int     `json:"weight" gorm:"default:1"`        // 用于负载均衡
	Status       int     `json:"status" gorm:"default:1"`        // 1: 正常, 2: 禁用, 3: 自动禁用
	TestTime     int64   `json:"test_time"`                      // 上次测试时间戳
	ResponseTime int     `json:"response_time"`                  // 响应时间 (ms)
	Balance      float64 `json:"balance"`                        // 渠道余额
	UsedQuota    int64   `json:"used_quota" gorm:"default:0"`
}

const (
	ChannelTypeOpenAI     = 1
	ChannelTypeAzure      = 3
	ChannelTypeCustom     = 8
	ChannelTypeAnthropic  = 14
	ChannelTypeGemini     = 18
	ChannelTypeMidjourney = 30 // Midjourney
)

const (
	ChannelStatusActive       = 1
	ChannelStatusDisabled     = 2
	ChannelStatusAutoDisabled = 3
)

func GetRandomChannel(typeStr string) (*Channel, error) {
	var typeInt int
	switch typeStr {
	case "openai":
		typeInt = ChannelTypeOpenAI
	case "midjourney":
		typeInt = ChannelTypeMidjourney
	// Add other types as needed
	default:
		return nil, common.NewError("Unknown channel type: " + typeStr)
	}

	var channel Channel
	err := common.DB.Where("type = ? AND status = ?", typeInt, ChannelStatusActive).Order("RANDOM()").First(&channel).Error
	return &channel, err
}
