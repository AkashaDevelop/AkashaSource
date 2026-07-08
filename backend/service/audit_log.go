package service

import (
	"STfreApi/common"
	"STfreApi/model"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"
)

var sensitiveBodyKeys = []string{"password", "secret", "key", "token", "credential"}

const maxAuditBodyLength = 4000

// AuditEntry 审计日志入参（中间件 → service 层传递）
type AuditEntry struct {
	OperatorId       int
	OperatorUsername string
	OperatorRole     string
	Method           string
	Path             string
	Route            string
	Action           string
	TargetType       string
	TargetId         string
	Success          bool
	StatusCode       int
	IP               string
	AuthMethod       string
	RequestId        string
	RawBody          []byte
}

// RecordAudit 写入审计日志
func RecordAudit(e AuditEntry) {
	entry := model.AuditLog{
		OperatorId:       e.OperatorId,
		OperatorUsername: e.OperatorUsername,
		OperatorRole:     e.OperatorRole,
		Method:           e.Method,
		Path:             e.Path,
		Route:            e.Route,
		Action:           e.Action,
		TargetType:       e.TargetType,
		TargetId:         e.TargetId,
		Success:          e.Success,
		StatusCode:       e.StatusCode,
		IP:               e.IP,
		AuthMethod:       e.AuthMethod,
		RequestId:        e.RequestId,
		RequestBody:      sanitizeAuditBody(e.RawBody),
		CreatedAt:        time.Now().Unix(),
	}
	if err := common.DB.Create(&entry).Error; err != nil {
		log.Printf("[操作审计] 写入失败: %v", err)
	}
}

// RecordAuditManual handler 手动埋点（设置 ContextKeyAuditLogged 防止中间件重复记录）
func RecordAuditManual(operatorId int, username, operatorRole, action, targetType, targetId, ip string, success bool, detail string) {
	entry := model.AuditLog{
		OperatorId:       operatorId,
		OperatorUsername: username,
		OperatorRole:     operatorRole,
		Action:           action,
		TargetType:       targetType,
		TargetId:         targetId,
		Success:          success,
		IP:               ip,
		RequestBody:      truncateAuditBody(detail),
		CreatedAt:        time.Now().Unix(),
	}
	if err := common.DB.Create(&entry).Error; err != nil {
		log.Printf("[操作审计] 手动记录失败: %v", err)
	}
}

// sanitizeAuditBody 敏感字段打码 + 超长截断
func sanitizeAuditBody(rawBody []byte) string {
	if len(rawBody) == 0 {
		return ""
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(rawBody, &parsed); err != nil {
		return truncateAuditBody("<非 JSON 请求体，长度: " + strconv.Itoa(len(rawBody)) + " 字节>")
	}

	for key := range parsed {
		if isSensitiveKey(key) {
			parsed[key] = "***"
		}
	}

	masked, err := json.Marshal(parsed)
	if err != nil {
		return truncateAuditBody("<脱敏失败，原始长度: " + strconv.Itoa(len(rawBody)) + " 字节>")
	}
	return truncateAuditBody(string(masked))
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, s := range sensitiveBodyKeys {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

func truncateAuditBody(body string) string {
	if len(body) <= maxAuditBodyLength {
		return body
	}
	return body[:maxAuditBodyLength] + "...(已截断)"
}
