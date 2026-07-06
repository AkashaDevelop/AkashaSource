package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetAuditLogs ～翻一翻审计小本本，把所有操作者（用户/管理员/超管）的操作足迹分页交出来～
func GetAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var logs []model.AuditLog
	var total int64
	db := buildAuditLogQuery(c, common.DB.Model(&model.AuditLog{}))
	db.Count(&total)
	if err := db.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&logs).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取审计日志失败")
		return
	}
	common.OK(c, gin.H{"data": logs, "total": total, "page": page, "size": pageSize})
}

func buildAuditLogQuery(c *gin.Context, db *gorm.DB) *gorm.DB {
	if v := c.Query("admin_username"); v != "" {
		db = db.Where("admin_username LIKE ?", "%"+v+"%")
	}
	if v := c.Query("role"); v != "" {
		db = db.Where("operator_role = ?", v)
	}
	if v := c.Query("method"); v != "" {
		db = db.Where("method = ?", v)
	}
	if v := c.Query("path"); v != "" {
		db = db.Where("path LIKE ?", "%"+v+"%")
	}
	if v := c.Query("target_type"); v != "" {
		db = db.Where("target_type = ?", v)
	}
	if v := c.Query("ip"); v != "" {
		db = db.Where("ip = ?", v)
	}
	if v := c.Query("start_time"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			db = db.Where("created_at >= ?", n)
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			db = db.Where("created_at >= ?", t.Unix())
		}
	}
	if v := c.Query("end_time"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			db = db.Where("created_at <= ?", n)
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			db = db.Where("created_at <= ?", t.Add(24*time.Hour).Unix())
		}
	}
	return db
}
