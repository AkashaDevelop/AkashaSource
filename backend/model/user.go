package model

type User struct {
	Id           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Username     string `json:"username" gorm:"unique;index"`
	Password     string `json:"password" gorm:"not null"` // 加密后的密码
	DisplayName  string `json:"display_name" gorm:"index"`
	Role         int    `json:"role" gorm:"default:1"`   // 1: 普通用户, 10: 管理员, 100: 超级管理员
	Status       int    `json:"status" gorm:"default:1"` // 1: 正常, 2: 封禁
	Email        string `json:"email" gorm:"index"`
	GithubId     string `json:"github_id" gorm:"index"`
	LinuxDOId    string `json:"linuxdo_id" gorm:"index"`
	LinuxDOLevel int    `json:"linuxdo_level" gorm:"default:0"`
	WechatId     string `json:"wechat_id" gorm:"index"`
	AccessToken  string `json:"access_token"` // 用于 API 访问
	Quota        int64  `json:"quota" gorm:"default:0"`
	UsedQuota    int64  `json:"used_quota" gorm:"default:0"` // 历史用量
	RequestCount int    `json:"request_count" gorm:"default:0"`
	Group        string `json:"group" gorm:"default:'default'"`
	AffCode      string `json:"aff_code" gorm:"index"`
	InviterId    int    `json:"inviter_id" gorm:"index"`
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
