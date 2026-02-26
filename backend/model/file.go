package model

type StoredFile struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId    int    `json:"user_id" gorm:"index"`
	Purpose   string `json:"purpose"`
	FileName  string `json:"filename"`
	Bytes     int64  `json:"bytes"`
	Content   []byte `json:"-"`
	CreatedAt int64  `json:"created_at" gorm:"index"`
}
