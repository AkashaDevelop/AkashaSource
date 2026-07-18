package model

import (
	"STfreApi/common"
	"time"
)

type Vendor struct {
	Id          int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string    `json:"name" gorm:"size:128;not null;index"`
	Code        string    `json:"code" gorm:"size:64;uniqueIndex"`
	BaseURL     string    `json:"base_url" gorm:"size:255"`
	ApiVersion  string    `json:"api_version" gorm:"size:64"`
	Icon        string    `json:"icon" gorm:"size:191"` // 🎀 @lobehub/icons 图标键名～
	Status      int       `json:"status" gorm:"default:1;index"`
	Description string    `json:"description" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

const (
	VendorStatusEnabled  = 1
	VendorStatusDisabled = 2
)

func (v *Vendor) Insert() error {
	if v.Status == 0 {
		v.Status = VendorStatusEnabled
	}
	return common.DB.Create(v).Error
}

func (v *Vendor) Update() error {
	return common.DB.Model(&Vendor{}).Where("id = ?", v.Id).Updates(v).Error
}

func (v *Vendor) Delete() error {
	return common.DB.Delete(v).Error
}

func GetVendorByID(id int) (*Vendor, error) {
	var v Vendor
	if err := common.DB.First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func IsVendorNameDuplicated(id int, name string) (bool, error) {
	if name == "" {
		return false, nil
	}
	var cnt int64
	err := common.DB.Model(&Vendor{}).Where("name = ? AND id <> ?", name, id).Count(&cnt).Error
	return cnt > 0, err
}

func GetAllVendors(offset int, limit int) ([]*Vendor, error) {
	var vendors []*Vendor
	err := common.DB.Order("id desc").Offset(offset).Limit(limit).Find(&vendors).Error
	return vendors, err
}

func SearchVendors(keyword string, offset int, limit int) ([]*Vendor, int64, error) {
	db := common.DB.Model(&Vendor{})
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR code LIKE ? OR base_url LIKE ? OR description LIKE ?", like, like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var vendors []*Vendor
	if err := db.Order("id desc").Offset(offset).Limit(limit).Find(&vendors).Error; err != nil {
		return nil, 0, err
	}
	return vendors, total, nil
}
