package model

import (
	"encoding/json"
	"strings"

	"STfreApi/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Ability 渠道能力表：group×model×channel_id 的三元小索引，
// 是"这个分组下、这个模型、这条渠道能不能用"的唯一权威答案哦～
// 用它来代替直接整表扫 Channel + Go 里过滤模型名的笨办法～
type Ability struct {
	GroupName string `json:"group" gorm:"column:group_name;primaryKey;type:varchar(64)"`
	Model     string `json:"model" gorm:"primaryKey;type:varchar(255)"`
	ChannelId int    `json:"channel_id" gorm:"primaryKey;index"`
	Enabled   bool   `json:"enabled"`
	Priority  int    `json:"priority" gorm:"default:0;index"`
	Weight    int    `json:"weight" gorm:"default:0"`
}

func splitTrim(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// buildAbilities 把一条渠道展开成 group × model 的笛卡尔积小能力们～
// model 集合 = Models 原始逗号列表 ∪ ModelMapping 里的映射键（caller 眼里"看得到"的模型名）～
func buildAbilities(channel *Channel) []Ability {
	groups := channel.GetGroups()
	if len(groups) == 0 {
		groups = []string{"default"}
	}

	modelSet := make(map[string]struct{})
	for _, m := range splitTrim(channel.Models) {
		modelSet[m] = struct{}{}
	}
	if strings.TrimSpace(channel.ModelMapping) != "" {
		var mapping map[string]string
		if err := json.Unmarshal([]byte(channel.ModelMapping), &mapping); err == nil {
			for callerModel := range mapping {
				callerModel = strings.TrimSpace(callerModel)
				if callerModel != "" {
					modelSet[callerModel] = struct{}{}
				}
			}
		}
	}

	seen := make(map[string]struct{}, len(groups)*len(modelSet))
	abilities := make([]Ability, 0, len(groups)*len(modelSet))
	for m := range modelSet {
		for _, g := range groups {
			key := g + "|" + m
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			abilities = append(abilities, Ability{
				GroupName: g,
				Model:     m,
				ChannelId: channel.Id,
				Enabled:   channel.Status == ChannelStatusActive,
				Priority:  channel.Priority,
				Weight:    channel.Weight,
			})
		}
	}
	return abilities
}

// AddAbilities 渠道新建时，把它能提供的 group×model 能力统统种下去～
func (channel *Channel) AddAbilities(tx *gorm.DB) error {
	abilities := buildAbilities(channel)
	if len(abilities) == 0 {
		return nil
	}
	db := common.DB
	if tx != nil {
		db = tx
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&abilities).Error
}

// UpdateAbilities 渠道编辑后，先把旧能力清空，再按最新配置重新长出来～
func (channel *Channel) UpdateAbilities(tx *gorm.DB) error {
	db := common.DB
	if tx != nil {
		db = tx
	}
	if err := db.Where("channel_id = ?", channel.Id).Delete(&Ability{}).Error; err != nil {
		return err
	}
	abilities := buildAbilities(channel)
	if len(abilities) == 0 {
		return nil
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&abilities).Error
}

// DeleteAbilities 渠道被删除时，把它名下的能力一起打扫干净～
func DeleteAbilitiesByChannelId(channelId int) error {
	return common.DB.Where("channel_id = ?", channelId).Delete(&Ability{}).Error
}

// GetChannelCandidates 按分组+模型查出所有可用渠道 id、优先级、权重，按优先级降序排好队～
func GetChannelCandidates(group string, modelName string) ([]Ability, error) {
	var abilities []Ability
	err := common.DB.Where("group_name = ? AND model = ? AND enabled = ?", group, modelName, true).
		Order("priority DESC").
		Find(&abilities).Error
	return abilities, err
}

// FixAllAbilities 把数据库里所有渠道的能力表全量重建一遍，用于版本升级回填或后台"一键修复"～
func FixAllAbilities() (success int, fails int) {
	var channels []Channel
	if err := common.DB.Find(&channels).Error; err != nil {
		return 0, 0
	}
	if err := common.DB.Exec("DELETE FROM abilities").Error; err != nil {
		fails = len(channels)
		return 0, fails
	}
	for i := range channels {
		if err := channels[i].AddAbilities(nil); err != nil {
			fails++
			continue
		}
		success++
	}
	return success, fails
}
