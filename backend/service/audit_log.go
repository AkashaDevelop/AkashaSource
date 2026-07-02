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

// sensitiveBodyKeys ～这些字段太隐私啦，记录的时候要蒙上小马甲～
var sensitiveBodyKeys = []string{"password", "secret", "key", "token", "credential"}

// maxAuditBodyLength ～请求体太长会撑爆数据库肚子，超过就要剪短哦～
const maxAuditBodyLength = 4000

// RecordAdminAudit ～把管理员大人的这一步操作悄悄记进小本本～
func RecordAdminAudit(adminId int, username, method, path, targetType, ip, requestId string, status int, rawBody []byte) {
	entry := model.AdminAuditLog{
		AdminId:       adminId,
		AdminUsername: username,
		Method:        method,
		Path:          path,
		TargetType:    targetType,
		StatusCode:    status,
		IP:            ip,
		RequestId:     requestId,
		RequestBody:   sanitizeAuditBody(rawBody),
		CreatedAt:     time.Now().Unix(),
	}
	if err := common.DB.Create(&entry).Error; err != nil {
		log.Printf("[操作审计] 写入失败: %v", err)
	}
}

// sanitizeAuditBody ～给请求体做个安全体检：敏感字段打码、超长内容剪裁～
func sanitizeAuditBody(rawBody []byte) string {
	if len(rawBody) == 0 {
		return ""
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(rawBody, &parsed); err != nil {
		// 不是合法 JSON（比如文件上传），就只留个长度提示，别硬存原文
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
