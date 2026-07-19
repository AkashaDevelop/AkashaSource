package model

import (
	"encoding/json"
	"strings"
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
	// 🌸 重构新增字段～分组配置收归表内，不再散落 options JSON
	Ratio      float64 `json:"ratio" gorm:"default:1"`                        // 分组计费倍率(原 options.group_ratio 收归而来)
	Visibility string  `json:"visibility" gorm:"type:varchar(20);default:'public'"` // public=公开可选 / hidden=隐藏(仅特殊授权或直配可用)
	Sort       int     `json:"sort" gorm:"default:0"`                         // 排序权重，越大越靠前
	CreatedAt  int64   `json:"created_at"`
}

// 🌸 可见性常量～
const (
	GroupVisibilityPublic = "public" // 公开：普通用户可见可选
	GroupVisibilityHidden = "hidden" // 隐藏：只有被特殊授权/直接分配的用户才能用
)

// GroupSpecialGrant 特殊分组授权表～
// 语义：用户的「基础分组」= BaseGroup 时，自动解锁 SpecialGroup 这个额外分组。
// 例：{BaseGroup:"svip", SpecialGroup:"ex"} 表示用户到了 svip 就能选 ex 分组(哪怕 ex 是隐藏的)～
type GroupSpecialGrant struct {
	Id           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	BaseGroup    string `json:"base_group" gorm:"type:varchar(100);index"` // 基础分组名
	SpecialGroup string `json:"special_group" gorm:"type:varchar(100)"`    // 解锁的特殊分组名
	CreatedAt    int64  `json:"created_at"`
}

// GetSpecialGrantsByBase 查某基础分组能解锁哪些特殊分组～
func GetSpecialGrantsByBase(baseGroup string) []string {
	if baseGroup == "" {
		return nil
	}
	var grants []GroupSpecialGrant
	if err := common.DB.Where("base_group = ?", baseGroup).Find(&grants).Error; err != nil {
		return nil
	}
	out := make([]string, 0, len(grants))
	for _, g := range grants {
		if g.SpecialGroup != "" {
			out = append(out, g.SpecialGroup)
		}
	}
	return out
}

// GetAllPublicGroups 拿所有公开分组(名→描述)～游客/普通用户的基础可选集
func GetAllPublicGroups() map[string]string {
	out := make(map[string]string)
	var groups []Group
	if err := common.DB.Where("visibility = ?", GroupVisibilityPublic).Find(&groups).Error; err != nil {
		return out
	}
	for _, g := range groups {
		out[g.Name] = g.Description
	}
	return out
}

// GetGroupDescription 取分组描述(找不到返回空)～
func GetGroupDescription(name string) string {
	var g Group
	if err := common.DB.Where("name = ?", name).First(&g).Error; err != nil {
		return ""
	}
	return g.Description
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
		{Name: "default", Description: "默认分组", ModelRatios: "{}", Ratio: 1, Visibility: GroupVisibilityPublic, CreatedAt: now},
		{Name: "vip", Description: "VIP 分组", ModelRatios: "{}", Ratio: 1, Visibility: GroupVisibilityPublic, CreatedAt: now},
		{Name: "svip", Description: "SVIP 分组", ModelRatios: "{}", Ratio: 1, Visibility: GroupVisibilityPublic, CreatedAt: now},
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

// MigrateGroupConfigFromOptions 🌸 一次性数据迁移～
// 把旧的 options JSON 配置搬进新的 Group 表 & GroupSpecialGrant 关联表：
//   · common.GroupRatio(分组倍率)              → Group.Ratio
//   · common.GroupSpecialUsableGroup(特殊规则) → GroupSpecialGrant 行
// 用 options 表一次性标记防重复，迁移完不动旧 JSON(留作兜底)～
func MigrateGroupConfigFromOptions() {
	const migFlag = "group_config_migrated_v1"
	var flag Option
	if err := common.DB.Where(Option{Key: migFlag}).First(&flag).Error; err == nil {
		return // 迁过了～
	}

	now := time.Now().Unix()

	// ① 分组倍率：给每个分组填上旧倍率(仅当分组当前还是默认倍率 1 时才覆盖，避免踩掉已改的)
	var groups []Group
	common.DB.Find(&groups)
	for i := range groups {
		g := &groups[i]
		oldRatio := common.GetGroupRatio(g.Name)
		if oldRatio > 0 && g.Ratio == 1 && oldRatio != 1 {
			common.DB.Model(&Group{}).Where("id = ?", g.Id).Update("ratio", oldRatio)
		}
	}

	// ② 特殊可用分组规则：解析 {base: {"+:x":"desc","-:y":"remove","z":"desc"}}
	//    "+:" 和无前缀 → 生成 GroupSpecialGrant(解锁)；"-:"(移除) 语义在新模型里用隐藏分组表达，跳过～
	rawJSON := common.GroupSpecialUsableGroup2JSONString()
	if rawJSON != "" && rawJSON != "{}" {
		var rules map[string]map[string]string
		if err := json.Unmarshal([]byte(rawJSON), &rules); err == nil {
			for base, inner := range rules {
				for key := range inner {
					special := key
					switch {
					case strings.HasPrefix(key, "-:"):
						continue // 移除规则不迁移
					case strings.HasPrefix(key, "+:"):
						special = strings.TrimPrefix(key, "+:")
					}
					special = strings.TrimSpace(special)
					if special == "" {
						continue
					}
					// 去重：同 base+special 只建一条
					var cnt int64
					common.DB.Model(&GroupSpecialGrant{}).Where("base_group = ? AND special_group = ?", base, special).Count(&cnt)
					if cnt == 0 {
						common.DB.Create(&GroupSpecialGrant{BaseGroup: base, SpecialGroup: special, CreatedAt: now})
					}
				}
			}
		}
	}

	common.DB.Create(&Option{Key: migFlag, Value: "true"})

	// 迁移完立刻把倍率同步进内存计费表～
	SyncGroupRatioToMemory()
}

// SyncGroupRatioToMemory 🌸 把 Group 表里的分组倍率刷进 common 的内存计费表～
// Group 表是倍率权威，计费热路径仍读 common.GetGroupRatio(内存 map)，
// 所以每次分组增删改后都要调一次，保证账算得对喵～
func SyncGroupRatioToMemory() {
	var groups []Group
	if err := common.DB.Find(&groups).Error; err != nil {
		return
	}
	ratios := make(map[string]float64, len(groups))
	for _, g := range groups {
		r := g.Ratio
		if r <= 0 {
			r = 1
		}
		ratios[g.Name] = r
	}
	if jsonBytes, err := json.Marshal(ratios); err == nil {
		common.UpdateGroupRatio(string(jsonBytes))
	}
}
