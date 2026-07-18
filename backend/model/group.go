package model

import (
	"time"

	"STfreApi/common"
)

type Group struct {
	Id             int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name           string `json:"name" gorm:"uniqueIndex;type:varchar(100)"`
	Description    string `json:"description" gorm:"type:varchar(500)"`
	ModelRatios    string `json:"model_ratios" gorm:"type:text"`    // JSON: {"gpt-4": 1.5}
	AllowedChannels string `json:"allowed_channels" gorm:"type:text"` // 逗号分隔渠道ID
	QPM            int    `json:"qpm" gorm:"default:0"`              // RPM：每分钟请求数，0=不限
	TPM            int    `json:"tpm" gorm:"default:0"`              // 🎀 每分钟 token 数，0=不限～
	RPD            int    `json:"rpd" gorm:"default:0"`              // 🎀 每日请求数，0=不限～
	CreatedAt      int64  `json:"created_at"`
}

// SeedDefaultGroups 播种 default / vip / svip 三颗默认分组小种子～
// 用 options 表的一次性标记控制：没播过就把缺的名字补上（已有同名的不动），
// 播过之后管理员随意改名/删除都不会被重复创建打扰哦～
func SeedDefaultGroups() {
	const seedFlag = "default_groups_seeded"

	var flag Option
	if err := common.DB.Where(Option{Key: seedFlag}).First(&flag).Error; err == nil {
		return // 已经播种过啦～
	}

	now := time.Now().Unix()
	seeds := []Group{
		{Name: "default", Description: "默认分组", ModelRatios: "{}", CreatedAt: now},
		{Name: "vip", Description: "VIP 分组", ModelRatios: "{}", CreatedAt: now},
		{Name: "svip", Description: "SVIP 分组", ModelRatios: "{}", CreatedAt: now},
	}
	for i := range seeds {
		var cnt int64
		common.DB.Model(&Group{}).Where("name = ?", seeds[i].Name).Count(&cnt)
		if cnt == 0 {
			common.DB.Create(&seeds[i])
		}
	}

	common.DB.Create(&Option{Key: seedFlag, Value: "true"})
}
