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
				return common.DB.AutoMigrate(&Log{}, &AuditLog{})
			},
		},
		{
			Version: 9,
			Name:    "新增自定义渠道支持字段",
			Apply: func() error {
				return common.DB.AutoMigrate(&Channel{}, &CustomChannelConfig{})
			},
		},
		{
			Version: 10,
			Name:    "支付订单新增 trade_no 字段",
			Apply: func() error {
				return common.DB.AutoMigrate(&PaymentOrder{})
			},
		},
		{
			Version: 11,
			Name:    "支付订单新增 quota_added/completed_at，trade_no 改 uniqueIndex",
			Apply: func() error {
				return common.DB.AutoMigrate(&PaymentOrder{})
			},
		},
		{
			Version: 12,
			Name:    "新增宸汐玄鉴行为风控事件表",
			Apply: func() error {
				return common.DB.AutoMigrate(&XuanJianEvent{})
			},
		},
		{
			Version: 13,
			Name:    "新增宸汐玄鉴行为风控规则表",
			Apply: func() error {
				return common.DB.AutoMigrate(&XuanJianRule{})
			},
		},
		{
			Version: 14,
			Name:    "新增渠道能力表(Ability)，令牌分组字段随通用迁移自动补齐",
			Apply: func() error {
				if err := common.DB.AutoMigrate(&Ability{}); err != nil {
					return err
				}
				// 新表刚建好，把现有渠道的能力全部回填一遍，免得升级完渠道全部选不到～
				FixAllAbilities()
				return nil
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
