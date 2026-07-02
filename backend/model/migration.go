package model

import (
	"STfreApi/common"
	"time"
)

type Migration struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Version   int    `json:"version" gorm:"uniqueIndex"`
	Name      string `json:"name"`
	AppliedAt int64  `json:"applied_at"`
}

// Initialize tables
func InitSchema() {
	err := common.DB.AutoMigrate(&User{}, &Channel{}, &Token{}, &Log{}, &Option{}, &Redemption{}, &Invitation{}, &MidjourneyTask{})
	if err != nil {
		panic(err)
	}
	ApplyMigrations()
	if err := ApplySQLFileMigrations(); err != nil {
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
	// 预留校验逻辑
	return nil
}

type MigrationStep struct {
	Version int
	Name    string
	Apply   func() error
}

func ApplyMigrations() {
	common.DB.AutoMigrate(&Migration{})

	steps := []MigrationStep{
		{
			Version: 1,
			Name:    "初始化文件与支付表",
			Apply: func() error {
				return common.DB.AutoMigrate(&StoredFile{}, &PaymentOrder{})
			},
		},
		{
			Version: 2,
			Name:    "新增渠道自定义请求头与令牌限制字段",
			Apply: func() error {
				return common.DB.AutoMigrate(&Channel{}, &Token{})
			},
		},
		{
			Version: 3,
			Name:    "新增用户2FA字段",
			Apply: func() error {
				return common.DB.AutoMigrate(&User{})
			},
		},
		{
			Version: 4,
			Name:    "新增渠道标签字段",
			Apply: func() error {
				return common.DB.AutoMigrate(&Channel{})
			},
		},
		{
			Version: 5,
			Name:    "新增用户分组与签到表",
			Apply: func() error {
				return common.DB.AutoMigrate(&Group{}, &CheckIn{})
			},
		},
		{
			Version: 6,
			Name:    "新增模型配置表",
			Apply: func() error {
				return common.DB.AutoMigrate(&ModelConfig{})
			},
		},
		{
			Version: 7,
			Name:    "新增上下文净化策略与事件表",
			Apply: func() error {
				return common.DB.AutoMigrate(&ContextSanitizationPolicy{}, &ContextSanitizationPolicyRevision{}, &ContextSanitizationEvent{})
			},
		},
		{
			Version: 8,
			Name:    "新增日志审计字段与管理员操作审计表",
			Apply: func() error {
				return common.DB.AutoMigrate(&Log{}, &AdminAuditLog{})
			},
		},
	}

	for _, step := range steps {
		var count int64
		common.DB.Model(&Migration{}).Where("version = ?", step.Version).Count(&count)
		if count > 0 {
			continue
		}
		if err := step.Apply(); err != nil {
			panic(err)
		}
		common.DB.Create(&Migration{
			Version:   step.Version,
			Name:      step.Name,
			AppliedAt: time.Now().Unix(),
		})
	}
}
