package model

// AuditLog 操作审计日志 — 记录全站用户/管理员/超管的写操作
type AuditLog struct {
	Id               int    `json:"id" gorm:"primaryKey;autoIncrement"`
	OperatorId       int    `json:"operator_id" gorm:"column:admin_id;index"`
	OperatorUsername string `json:"operator_username" gorm:"column:admin_username;index"`
	OperatorRole     string `json:"operator_role" gorm:"index"` // user / admin / root
	Method           string `json:"method"`
	Path             string `json:"path" gorm:"index"`
	Route            string `json:"route" gorm:"column:route_template;index"` // 路由模板，如 /api/user/:id
	Action           string `json:"action" gorm:"index"`                     // 语义动作，如 user.create / channel.delete
	TargetType       string `json:"target_type" gorm:"index"`
	TargetId         string `json:"target_id" gorm:"index"` // 被操作资源 ID（从路径参数提取）
	Success          bool   `json:"success" gorm:"index"`   // 业务成功（解析响应体 JSON 的 code 字段）
	StatusCode       int    `json:"status_code"`
	IP               string `json:"ip" gorm:"index"`
	AuthMethod       string `json:"auth_method"` // session / access_token / passkey / oauth
	RequestId        string `json:"request_id" gorm:"index"`
	RequestBody      string `json:"request_body" gorm:"type:text"`
	CreatedAt        int64  `json:"created_at" gorm:"index"`
}

func (AuditLog) TableName() string {
	return "admin_audit_logs"
}
