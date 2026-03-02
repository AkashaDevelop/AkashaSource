package model

import (
	"STfreApi/common"
	"time"
)

type ModelMeta struct {
	Id            int       `json:"id" gorm:"primaryKey;autoIncrement"`
	VendorId      int       `json:"vendor_id" gorm:"index"`
	ModelName     string    `json:"model_name" gorm:"size:191;index"`
	DisplayName   string    `json:"display_name" gorm:"size:191"`
	ModelType     string    `json:"model_type" gorm:"size:64;index"`
	ContextLength int       `json:"context_length" gorm:"default:0"`
	InputPrice    float64   `json:"input_price" gorm:"default:0"`
	OutputPrice   float64   `json:"output_price" gorm:"default:0"`
	Enabled       bool      `json:"enabled" gorm:"default:true;index"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (m *ModelMeta) Insert() error {
	if !m.Enabled {
		// 保持前端可控；仅当显式传 false 时不覆盖
	}
	return common.DB.Create(m).Error
}

func (m *ModelMeta) Update() error {
	return common.DB.Model(&ModelMeta{}).Where("id = ?", m.Id).Updates(m).Error
}

func (m *ModelMeta) Delete() error {
	return common.DB.Delete(m).Error
}

func GetModelMetaByID(id int) (*ModelMeta, error) {
	var item ModelMeta
	if err := common.DB.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func IsModelNameDuplicated(id int, modelName string) (bool, error) {
	if modelName == "" {
		return false, nil
	}
	var cnt int64
	err := common.DB.Model(&ModelMeta{}).Where("model_name = ? AND id <> ?", modelName, id).Count(&cnt).Error
	return cnt > 0, err
}

func GetAllModelsMeta(offset int, limit int) ([]*ModelMeta, error) {
	var items []*ModelMeta
	err := common.DB.Order("id desc").Offset(offset).Limit(limit).Find(&items).Error
	return items, err
}

func SearchModelsMeta(keyword string, offset int, limit int) ([]*ModelMeta, int64, error) {
	db := common.DB.Model(&ModelMeta{})
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("model_name LIKE ? OR display_name LIKE ? OR model_type LIKE ?", like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []*ModelMeta
	if err := db.Order("id desc").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
