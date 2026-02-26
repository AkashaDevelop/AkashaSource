package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetAllLogs 获取所有日志 (管理员)
func GetAllLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize
	var logs []model.Log
	var total int64

	db := buildLogQuery(c, common.DB.Model(&model.Log{}))

	db.Count(&total)
	if err := db.Order("id desc").Limit(pageSize).Offset(offset).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取日志失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  logs,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// GetUserLogs 获取当前用户日志
func GetUserLogs(c *gin.Context) {
	userId, _ := c.Get("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * pageSize
	var logs []model.Log
	var total int64

	db := buildLogQuery(c, common.DB.Model(&model.Log{}).Where("user_id = ?", userId))

	db.Count(&total)
	if err := db.Order("id desc").Limit(pageSize).Offset(offset).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取日志失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  logs,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

func buildLogQuery(c *gin.Context, db *gorm.DB) *gorm.DB {
	if username := c.Query("username"); username != "" {
		db = db.Where("username LIKE ?", "%"+username+"%")
	}
	if tokenName := c.Query("token_name"); tokenName != "" {
		db = db.Where("token_name LIKE ?", "%"+tokenName+"%")
	}
	if modelName := c.Query("model_name"); modelName != "" {
		db = db.Where("model_name LIKE ?", "%"+modelName+"%")
	}
	if content := c.Query("content"); content != "" {
		db = db.Where("content LIKE ?", "%"+content+"%")
	}
	if logType := c.Query("type"); logType != "" {
		if v, err := strconv.Atoi(logType); err == nil {
			db = db.Where("type = ?", v)
		}
	}
	if userId := c.Query("user_id"); userId != "" {
		if v, err := strconv.Atoi(userId); err == nil {
			db = db.Where("user_id = ?", v)
		}
	}
	if channelId := c.Query("channel_id"); channelId != "" {
		if v, err := strconv.Atoi(channelId); err == nil {
			db = db.Where("channel_id = ?", v)
		}
	}
	if start := c.Query("start_time"); start != "" {
		if v, err := strconv.ParseInt(start, 10, 64); err == nil {
			db = db.Where("created_at >= ?", v)
		} else if t, err := time.Parse("2006-01-02", start); err == nil {
			db = db.Where("created_at >= ?", t.Unix())
		}
	}
	if end := c.Query("end_time"); end != "" {
		if v, err := strconv.ParseInt(end, 10, 64); err == nil {
			db = db.Where("created_at <= ?", v)
		} else if t, err := time.Parse("2006-01-02", end); err == nil {
			db = db.Where("created_at <= ?", t.Add(24*time.Hour).Unix())
		}
	}
	if minQuota := c.Query("min_quota"); minQuota != "" {
		if v, err := strconv.ParseInt(minQuota, 10, 64); err == nil {
			db = db.Where("quota >= ?", v)
		}
	}
	if maxQuota := c.Query("max_quota"); maxQuota != "" {
		if v, err := strconv.ParseInt(maxQuota, 10, 64); err == nil {
			db = db.Where("quota <= ?", v)
		}
	}
	return db
}

// DeleteLogs deletes logs before a given timestamp (admin)
func DeleteLogs(c *gin.Context) {
	var req struct {
		BeforeTimestamp int64 `json:"before_timestamp"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.BeforeTimestamp <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供有效的时间戳"})
		return
	}

	result := common.DB.Where("created_at < ?", req.BeforeTimestamp).Delete(&model.Log{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除日志失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "日志清理成功",
		"deleted": result.RowsAffected,
	})
}
