package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetAllLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 { page = 1 }
	if pageSize > 100 { pageSize = 100 }
	var logs []model.Log
	var total int64
	db := buildLogQuery(c, common.DB.Model(&model.Log{}))
	db.Count(&total)
	if err := db.Order("id desc").Limit(pageSize).Offset((page-1)*pageSize).Find(&logs).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取日志失败")
		return
	}
	common.OK(c, gin.H{"data": logs, "total": total, "page": page, "size": pageSize})
}

func GetUserLogs(c *gin.Context) {
	userId, _ := c.Get("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 { page = 1 }
	var logs []model.Log
	var total int64
	db := buildLogQuery(c, common.DB.Model(&model.Log{}).Where("user_id = ?", userId))
	db.Count(&total)
	if err := db.Order("id desc").Limit(pageSize).Offset((page-1)*pageSize).Find(&logs).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取日志失败")
		return
	}
	common.OK(c, gin.H{"data": logs, "total": total, "page": page, "size": pageSize})
}

func buildLogQuery(c *gin.Context, db *gorm.DB) *gorm.DB {
	if v := c.Query("username"); v != "" { db = db.Where("username LIKE ?", "%"+v+"%") }
	if v := c.Query("token_name"); v != "" { db = db.Where("token_name LIKE ?", "%"+v+"%") }
	if v := c.Query("model_name"); v != "" { db = db.Where("model_name LIKE ?", "%"+v+"%") }
	if v := c.Query("content"); v != "" { db = db.Where("content LIKE ?", "%"+v+"%") }
	if v := c.Query("type"); v != "" {
		if n, err := strconv.Atoi(v); err == nil { db = db.Where("type = ?", n) }
	}
	if v := c.Query("user_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil { db = db.Where("user_id = ?", n) }
	}
	if v := c.Query("channel_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil { db = db.Where("channel_id = ?", n) }
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
	if v := c.Query("min_quota"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil { db = db.Where("quota >= ?", n) }
	}
	if v := c.Query("max_quota"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil { db = db.Where("quota <= ?", n) }
	}
	return db
}

func DeleteLogs(c *gin.Context) {
	var req struct{ BeforeTimestamp int64 `json:"before_timestamp"` }
	if err := c.ShouldBindJSON(&req); err != nil || req.BeforeTimestamp <= 0 {
		common.Fail(c, common.CodeParamError, "请提供有效的时间戳")
		return
	}
	result := common.DB.Where("created_at < ?", req.BeforeTimestamp).Delete(&model.Log{})
	if result.Error != nil {
		common.Fail(c, common.CodeServerError, "删除日志失败")
		return
	}
	common.OKMsg(c, "日志清理成功", gin.H{"deleted": result.RowsAffected})
}
