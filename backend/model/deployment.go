package model

import "time"

type Deployment struct {
	Id           int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Name         string    `json:"name" gorm:"size:191;index"`
	VendorId     int       `json:"vendor_id" gorm:"index"`
	ModelMetaId  int       `json:"model_meta_id" gorm:"index"`
	Region       string    `json:"region" gorm:"size:64"`
	HardwareType string    `json:"hardware_type" gorm:"size:64"`
	Replicas     int       `json:"replicas" gorm:"default:1"`
	Status       string    `json:"status" gorm:"size:32;default:'running';index"`
	Endpoint     string    `json:"endpoint" gorm:"size:255"`
	ApiKey       string    `json:"api_key" gorm:"size:255"`
	ExpireAt     int64     `json:"expire_at" gorm:"default:0"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
