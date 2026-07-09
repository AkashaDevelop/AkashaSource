package model

import "STfreApi/common"

// QingyuanRule 宸汐清源规则库
type QingyuanRule struct {
	Id              int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Category        string `json:"category" gorm:"type:varchar(64);not null;index"`
	Name            string `json:"name" gorm:"type:varchar(128);not null"`
	Description     string `json:"description" gorm:"type:text"`
	Score           int    `json:"score" gorm:"not null;default:50"`
	Keywords        string `json:"keywords" gorm:"type:json;not null"` // JSON 数组
	ContextRequired string `json:"context_required" gorm:"type:varchar(256)"`
	MatchMode       string `json:"match_mode" gorm:"type:varchar(32);default:'any'"`
	Enabled         bool   `json:"enabled" gorm:"default:true;index"`
	Language        string `json:"language" gorm:"type:varchar(16);default:'all'"`
	SortOrder       int    `json:"sort_order" gorm:"default:0"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
	CreatedBy       int    `json:"created_by"`
}

func (QingyuanRule) TableName() string {
	return "qingyuan_rules"
}

func (r *QingyuanRule) Insert() error {
	return common.DB.Create(r).Error
}

func (r *QingyuanRule) Update() error {
	return common.DB.Model(r).Where("id = ?", r.Id).Updates(r).Error
}

func GetAllQingyuanRules(page, pageSize int, category string, enabled *bool) ([]QingyuanRule, int64, error) {
	var rules []QingyuanRule
	var total int64

	query := common.DB.Model(&QingyuanRule{})
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Order("category, sort_order, id").Offset(offset).Limit(pageSize).Find(&rules).Error
	return rules, total, err
}

func GetQingyuanRuleById(id int) (*QingyuanRule, error) {
	var rule QingyuanRule
	err := common.DB.First(&rule, id).Error
	return &rule, err
}

func DeleteQingyuanRule(id int) error {
	return common.DB.Delete(&QingyuanRule{}, id).Error
}

// QingyuanRuleCategory 规则分类元数据
type QingyuanRuleCategory struct {
	Id             int    `json:"id" gorm:"primaryKey;autoIncrement"`
	CategoryKey    string `json:"category_key" gorm:"type:varchar(64);unique;not null"`
	DisplayName    string `json:"display_name" gorm:"type:varchar(128);not null"`
	ParentCategory string `json:"parent_category" gorm:"type:varchar(64)"`
	SortOrder      int    `json:"sort_order" gorm:"default:0"`
	Description    string `json:"description" gorm:"type:text"`
}

func (QingyuanRuleCategory) TableName() string {
	return "qingyuan_rule_categories"
}

func GetAllQingyuanCategories() ([]QingyuanRuleCategory, error) {
	var categories []QingyuanRuleCategory
	err := common.DB.Order("sort_order, id").Find(&categories).Error
	return categories, err
}
