package model

// AuditLog ～悄悄记下每一位操作者（普通用户/管理员/超管）的每一步操作足迹，出了岔子好找线索呀～
type AuditLog struct {
	Id               int    `json:"id" gorm:"primaryKey;autoIncrement"`
	OperatorId       int    `json:"operator_id" gorm:"column:admin_id;index"`
	OperatorUsername string `json:"operator_username" gorm:"column:admin_username;index"`
	OperatorRole     string `json:"operator_role" gorm:"index"` // user / admin / root
	Method           string `json:"method"`
	Path             string `json:"path" gorm:"index"`
	TargetType       string `json:"target_type" gorm:"index"` // 从路径里揪出来的资源名，比如 user / channel / option
	StatusCode       int    `json:"status_code"`
	IP               string `json:"ip" gorm:"index"`
	RequestId        string `json:"request_id" gorm:"index"`
	RequestBody      string `json:"request_body" gorm:"type:text"` // 已经脱掉敏感小马甲、也裁剪过长度的请求体
	CreatedAt        int64  `json:"created_at" gorm:"index"`
}

// TableName ～沿用原来的表名，纯 Go 层面改名字，不折腾数据库迁移～
func (AuditLog) TableName() string {
	return "admin_audit_logs"
}
