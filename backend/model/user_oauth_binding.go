package model

import (
	"errors"
	"strings"
	"time"

	"STfreApi/common"

	"gorm.io/gorm"
)

// UserOAuthBinding stores relation between user and custom OAuth provider.
type UserOAuthBinding struct {
	Id             int       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId         int       `json:"user_id" gorm:"not null;uniqueIndex:ux_user_provider"`
	ProviderId     int       `json:"provider_id" gorm:"not null;uniqueIndex:ux_user_provider;uniqueIndex:ux_provider_userid"`
	ProviderUserId string    `json:"provider_user_id" gorm:"type:varchar(256);not null;uniqueIndex:ux_provider_userid"`
	CreatedAt      time.Time `json:"created_at"`
}

func (UserOAuthBinding) TableName() string {
	return "user_oauth_bindings"
}

func GetUserOAuthBindingsByUserId(userId int) ([]*UserOAuthBinding, error) {
	var bindings []*UserOAuthBinding
	err := common.DB.Where("user_id = ?", userId).Order("id asc").Find(&bindings).Error
	return bindings, err
}

func GetUserOAuthBinding(userId, providerId int) (*UserOAuthBinding, error) {
	var binding UserOAuthBinding
	err := common.DB.Where("user_id = ? AND provider_id = ?", userId, providerId).First(&binding).Error
	if err != nil {
		return nil, err
	}
	return &binding, nil
}

func GetUserByOAuthBinding(providerId int, providerUserId string) (*User, error) {
	var binding UserOAuthBinding
	err := common.DB.Where("provider_id = ? AND provider_user_id = ?", providerId, strings.TrimSpace(providerUserId)).First(&binding).Error
	if err != nil {
		return nil, err
	}
	var user User
	err = common.DB.First(&user, binding.UserId).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func IsProviderUserIdTaken(providerId int, providerUserId string) bool {
	var count int64
	common.DB.Model(&UserOAuthBinding{}).
		Where("provider_id = ? AND provider_user_id = ?", providerId, strings.TrimSpace(providerUserId)).
		Count(&count)
	return count > 0
}

func CreateUserOAuthBinding(binding *UserOAuthBinding) error {
	if binding.UserId == 0 {
		return errors.New("user ID is required")
	}
	if binding.ProviderId == 0 {
		return errors.New("provider ID is required")
	}
	binding.ProviderUserId = strings.TrimSpace(binding.ProviderUserId)
	if binding.ProviderUserId == "" {
		return errors.New("provider user ID is required")
	}
	if IsProviderUserIdTaken(binding.ProviderId, binding.ProviderUserId) {
		return errors.New("this OAuth account is already bound to another user")
	}
	binding.CreatedAt = time.Now()
	return common.DB.Create(binding).Error
}

func CreateUserOAuthBindingWithTx(tx *gorm.DB, binding *UserOAuthBinding) error {
	if binding.UserId == 0 {
		return errors.New("user ID is required")
	}
	if binding.ProviderId == 0 {
		return errors.New("provider ID is required")
	}
	binding.ProviderUserId = strings.TrimSpace(binding.ProviderUserId)
	if binding.ProviderUserId == "" {
		return errors.New("provider user ID is required")
	}

	var count int64
	tx.Model(&UserOAuthBinding{}).
		Where("provider_id = ? AND provider_user_id = ?", binding.ProviderId, binding.ProviderUserId).
		Count(&count)
	if count > 0 {
		return errors.New("this OAuth account is already bound to another user")
	}
	binding.CreatedAt = time.Now()
	return tx.Create(binding).Error
}

func UpdateUserOAuthBinding(userId, providerId int, newProviderUserId string) error {
	newProviderUserId = strings.TrimSpace(newProviderUserId)
	if newProviderUserId == "" {
		return errors.New("provider user ID is required")
	}

	var existing UserOAuthBinding
	err := common.DB.Where("provider_id = ? AND provider_user_id = ?", providerId, newProviderUserId).First(&existing).Error
	if err == nil && existing.UserId != userId {
		return errors.New("this OAuth account is already bound to another user")
	}

	var binding UserOAuthBinding
	err = common.DB.Where("user_id = ? AND provider_id = ?", userId, providerId).First(&binding).Error
	if err != nil {
		return CreateUserOAuthBinding(&UserOAuthBinding{
			UserId:         userId,
			ProviderId:     providerId,
			ProviderUserId: newProviderUserId,
		})
	}
	return common.DB.Model(&binding).Update("provider_user_id", newProviderUserId).Error
}

func DeleteUserOAuthBinding(userId, providerId int) error {
	return common.DB.Where("user_id = ? AND provider_id = ?", userId, providerId).Delete(&UserOAuthBinding{}).Error
}

func DeleteUserOAuthBindingsByUserId(userId int) error {
	return common.DB.Where("user_id = ?", userId).Delete(&UserOAuthBinding{}).Error
}

func GetBindingCountByProviderId(providerId int) (int64, error) {
	var count int64
	err := common.DB.Model(&UserOAuthBinding{}).Where("provider_id = ?", providerId).Count(&count).Error
	return count, err
}
