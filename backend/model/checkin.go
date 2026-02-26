package model

type CheckIn struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId    int    `json:"user_id" gorm:"index"`
	CheckDate string `json:"check_date" gorm:"type:varchar(10);uniqueIndex:idx_user_date"` // YYYY-MM-DD
	Reward    int64  `json:"reward"`
	CreatedAt int64  `json:"created_at"`
}
