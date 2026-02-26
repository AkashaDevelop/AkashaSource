package model

type SunoTask struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	TaskId    string `json:"task_id" gorm:"uniqueIndex"`
	UserId    int    `json:"user_id" gorm:"index"`
	Action    string `json:"action"` // generate, extend, etc.
	Status    string `json:"status" gorm:"default:'pending'"` // pending, processing, completed, failed
	Input     string `json:"input" gorm:"type:text"`
	Result    string `json:"result" gorm:"type:text"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
	Quota     int64  `json:"quota" gorm:"default:0"`
}
