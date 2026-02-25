package model

import (
	"STfreApi/common"
)

// Initialize tables
func InitSchema() {
	err := common.DB.AutoMigrate(&User{}, &Channel{}, &Token{}, &Log{}, &Option{}, &Redemption{})
	if err != nil {
		panic(err)
	}
}

// User methods
func (user *User) Insert() error {
	var err error
	if user.Password != "" {
		user.Password, err = common.Password2Hash(user.Password)
		if err != nil {
			return err
		}
	}
	return common.DB.Create(user).Error
}

func (user *User) ValidateAndFill() error {
	// Add validation logic here
	return nil
}
