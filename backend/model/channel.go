package model

import (
	"errors"

	"STfreApi/common"
)

type Channel struct {
	Id                     int     `json:"id" gorm:"primaryKey;autoIncrement"`
	Type                   int     `json:"type" gorm:"default:1"` // 1: OpenAI, 2: API2D, 3: Azure, 等
	Key                    string  `json:"key" gorm:"type:text"`  // API Key
	BaseURL                string  `json:"base_url" gorm:"column:base_url;default:''"`
	Name                   string  `json:"name" gorm:"index"`
	Group                  string  `json:"group" gorm:"default:'default'"` // 支持多组
	Models                 string  `json:"models" gorm:"type:text"`        // 支持的模型 (逗号分隔)
	ModelMapping           string  `json:"model_mapping" gorm:"type:text"` // 模型映射 JSON 字符串
	Headers                string  `json:"headers" gorm:"type:text"`
	Priority               int     `json:"priority" gorm:"default:10"`     // 值越大优先级越高
	Weight                 int     `json:"weight" gorm:"default:1"`        // 用于负载均衡
	AutoBan                int     `json:"auto_ban" gorm:"default:1"`      // 0: 不自动禁用, 1: 自动禁用
	Proxy                  string  `json:"proxy" gorm:"type:varchar(255)"` // HTTP 代理地址
	Status                 int     `json:"status" gorm:"default:1"`        // 1: 正常, 2: 禁用, 3: 自动禁用
	TestTime               int64   `json:"test_time"`                      // 上次测试时间戳
	ResponseTime           int     `json:"response_time"`                  // 响应时间 (ms)
	Balance                float64 `json:"balance"`                        // 渠道余额
	UsedQuota              int64   `json:"used_quota" gorm:"default:0"`
	Tags                   string  `json:"tags" gorm:"type:varchar(500);default:''"`   // 逗号分隔标签
	MultiKeyStatus         string  `json:"multi_key_status" gorm:"type:text"`          // JSON: {"0":1,"1":2}
	MultiKeyDisabledTime   string  `json:"multi_key_disabled_time" gorm:"type:text"`   // JSON: {"1":1700000000}
	MultiKeyDisabledReason string  `json:"multi_key_disabled_reason" gorm:"type:text"` // JSON: {"1":"manual"}
	CheckinEnabled         int     `json:"checkin_enabled" gorm:"default:0"`           // 是否启用自动签到
	LastCheckinAt          int64   `json:"last_checkin_at"`                            // 上次签到时间戳
	BalanceRefreshAt       int64   `json:"balance_refresh_at"`                         // 上次余额刷新时间戳
	AccountUsername        string  `json:"account_username" gorm:"type:varchar(255)"`  // 账号用户名
	AccountPassword        string  `json:"account_password" gorm:"type:text"`          // 账号密码(加密存储)
	AccessToken            string  `json:"access_token" gorm:"type:text"`              // 会话令牌
	PlatformUserId         int     `json:"platform_user_id"`                           // 平台用户ID
}

const (
	ChannelTypeOpenAI      = 1
	ChannelTypeAzure       = 3
	ChannelTypeCustom      = 8
	ChannelTypeAnthropic   = 14
	ChannelTypeGemini      = 18
	ChannelTypeMidjourney  = 30 // Midjourney
	ChannelTypeQwen        = 40
	ChannelTypeHunyuan     = 41
	ChannelTypeWenxin      = 42
	ChannelTypeSpark       = 43
	ChannelTypeDeepseek    = 44
	ChannelTypeZhipu       = 45
	ChannelTypeMoonshot    = 46
	ChannelTypeOllama      = 47
	ChannelTypeSuno        = 50
	ChannelTypeDify        = 51
	ChannelTypeSiliconFlow = 52
	ChannelTypeMistral     = 53
	ChannelTypeXAI         = 54
	ChannelTypeGroq        = 55
	ChannelTypeTogether    = 56
	ChannelTypePerplexity  = 57
	ChannelTypeCodex       = 58
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
		return nil, errors.New("Unknown channel type: " + typeStr)
	}

	var channel Channel
	err := common.DB.Where("type = ? AND status = ?", typeInt, ChannelStatusActive).Order("RANDOM()").First(&channel).Error
	return &channel, err
}
