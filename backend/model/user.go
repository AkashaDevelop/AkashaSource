package model

import (
	"encoding/json"
	"STfreApi/dto"
)

type User struct {
	Id                int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Username          string `json:"username" gorm:"unique;index"`
	Password          string `json:"password" gorm:"not null"`
	DisplayName       string `json:"display_name" gorm:"index"`
	Role              int    `json:"role" gorm:"default:1"`   // 1: 普通用户, 10: 管理员, 100: 超级管理员
	Status            int    `json:"status" gorm:"default:1"` // 1: 正常, 2: 封禁
	Email             string `json:"email" gorm:"index"`
	GithubId          string `json:"github_id" gorm:"index"`
	LinuxDOId         string `json:"linuxdo_id" gorm:"index"`
	LinuxDOLevel      int    `json:"linuxdo_level" gorm:"default:0"`
	DiscordId         string `json:"discord_id" gorm:"index"`
	OIDCId            string `json:"oidc_id" gorm:"index"`
	TelegramId        string `json:"telegram_id" gorm:"index"`
	WechatId          string `json:"wechat_id" gorm:"index"`
	AccessToken       string `json:"access_token"` // 用于 API 访问
	Quota             int64  `json:"quota" gorm:"default:0"`
	UsedQuota         int64  `json:"used_quota" gorm:"default:0"` // 历史用量
	RequestCount      int    `json:"request_count" gorm:"default:0"`
	Group             string `json:"group" gorm:"default:'default'"`
	ExtraGroups       string `json:"extra_groups" gorm:"type:varchar(500);default:''"` // 🌸 直接授予的额外分组，逗号分隔(与基础分组共存)
	BillingPreference string `json:"billing_preference" gorm:"type:varchar(32);default:'subscription_first'"`
	AffCode           string `json:"aff_code" gorm:"index"`
	AffQuota          int64  `json:"aff_quota" gorm:"default:0"` // 邀请返利余额，需通过转账接口转入可用 Quota
	InviterId         int    `json:"inviter_id" gorm:"index"`

	// 2FA / TOTP
	TOTPSecret    string `json:"-" gorm:"type:varchar(255)"`
	TOTPEnabled   bool   `json:"totp_enabled" gorm:"default:false"`
	BackupCodes   string `json:"-" gorm:"type:text"` // JSON array of hashed backup codes
	TOTPFailCount int    `json:"totp_fail_count" gorm:"default:0"`
	TOTPLockedAt  int64  `json:"totp_locked_at" gorm:"default:0"`

	// User Settings
	Setting string `json:"setting" gorm:"type:text"`

	// 实名认证状态: 0=未认证 1=已认证 2=认证失败
	RealnameStatus int `json:"realname_status" gorm:"default:0"`
}

const (
	RoleUser  = 1
	RoleAdmin = 10
	RoleRoot  = 100
)

const (
	UserStatusActive = 1
	UserStatusBanned = 2
)

// MarshalJSON ～这里我亲自把关，密码哈希绝对不会偷溜出去哦，再也不用每个 handler 手动清空了～
func (u User) MarshalJSON() ([]byte, error) {
	type Alias User // 新类型，无 MarshalJSON 方法，防止递归
	a := Alias(u)
	a.Password = ""
	return json.Marshal(a)
}

func (u *User) GetSetting() dto.UserSetting {
	var setting dto.UserSetting
	if u.Setting != "" {
		json.Unmarshal([]byte(u.Setting), &setting)
	}
	return setting
}

func (u *User) SetSetting(setting dto.UserSetting) error {
	data, err := json.Marshal(setting)
	if err != nil {
		return err
	}
	u.Setting = string(data)
	return nil
}
