package model

// AdminAuditLog ～悄悄记下管理员大人的每一步操作足迹，出了岔子好找线索呀～
type AdminAuditLog struct {
	Id            int    `json:"id" gorm:"primaryKey;autoIncrement"`
	AdminId       int    `json:"admin_id" gorm:"index"`
	AdminUsername string `json:"admin_username" gorm:"index"`
	Method        string `json:"method"`
	Path          string `json:"path" gorm:"index"`
	TargetType    string `json:"target_type" gorm:"index"` // 从路径里揪出来的资源名，比如 user / channel / option
	StatusCode    int    `json:"status_code"`
	IP            string `json:"ip" gorm:"index"`
	RequestId     string `json:"request_id" gorm:"index"`
	RequestBody   string `json:"request_body" gorm:"type:text"` // 已经脱掉敏感小马甲、也裁剪过长度的请求体
	CreatedAt     int64  `json:"created_at" gorm:"index"`
}
