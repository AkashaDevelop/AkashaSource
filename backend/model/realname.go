package model

import (
	"STfreApi/common"
	"time"
)

// RealnameAuth 实名认证记录
type RealnameAuth struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId      int    `json:"user_id" gorm:"index"`
	RealName    string `json:"real_name" gorm:"type:varchar(64)"`
	IdCardHash  string `json:"-" gorm:"type:varchar(128);index"` // 身份证号哈希，不返回明文
	CertifyId   string `json:"certify_id" gorm:"type:varchar(128);index"` // 阿里云认证流水号
	Provider    string `json:"provider" gorm:"type:varchar(32);default:'aliyun'"`
	Status      int    `json:"status" gorm:"default:0"` // 0=待认证 1=已通过 2=未通过 3=已过期
	Reason      string `json:"reason" gorm:"type:varchar(500)"` // 未通过原因
	SceneId     string `json:"scene_id" gorm:"type:varchar(128)"` // 认证场景ID
	CreatedAt   int64  `json:"created_at"`
	VerifiedAt  int64  `json:"verified_at"`
}

const (
	RealnameStatusPending  = 0 // 待认证
	RealnameStatusPassed   = 1 // 已通过
	RealnameStatusFailed   = 2 // 未通过
	RealnameStatusExpired  = 3 // 已过期
)

// 场景类型常量
const (
	RealnameScenarioModelCall  = "model_call"   // 模型调用
	RealnameScenarioRecharge   = "recharge"     // 充值
	RealnameScenarioDoubleBlind = "double_blind" // 双盲类型
)

func (RealnameAuth) TableName() string {
	return "realname_auths"
}

// Create 创建实名认证记录
func (r *RealnameAuth) Create() error {
	r.CreatedAt = time.Now().Unix()
	return common.DB.Create(r).Error
}

// Update 更新实名认证记录
func (r *RealnameAuth) Update() error {
	return common.DB.Save(r).Error
}

// GetRealnameAuthByUserId 根据用户ID获取最新认证记录
func GetRealnameAuthByUserId(userId int) (*RealnameAuth, error) {
	var record RealnameAuth
	err := common.DB.Where("user_id = ?", userId).Order("id desc").First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetRealnameAuthByCertifyId 根据认证流水号获取记录
func GetRealnameAuthByCertifyId(certifyId string) (*RealnameAuth, error) {
	var record RealnameAuth
	err := common.DB.Where("certify_id = ?", certifyId).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetAllRealnameRecords 管理员查询所有认证记录
func GetAllRealnameRecords(page, pageSize int, status int) ([]RealnameAuth, int64, error) {
	var records []RealnameAuth
	var total int64
	query := common.DB.Model(&RealnameAuth{})
	if status >= 0 {
		query = query.Where("status = ?", status)
	}
	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Order("id desc").Offset(offset).Limit(pageSize).Find(&records).Error
	return records, total, err
}
