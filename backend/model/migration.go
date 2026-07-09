package model

import (
	"STfreApi/common"
)

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
