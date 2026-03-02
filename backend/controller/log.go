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
	if page < 1 {
		page = 1
	}
	if pageSize > 100 {
		pageSize = 100
	}
	var logs []model.Log
	var total int64
	db := buildLogQuery(c, common.DB.Model(&model.Log{}))
	db.Count(&total)
	if err := db.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&logs).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取日志失败")
		return
	}
	common.OK(c, gin.H{"data": logs, "total": total, "page": page, "size": pageSize})
}

func GetUserLogs(c *gin.Context) {
	userId, _ := c.Get("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	var logs []model.Log
	var total int64
	db := buildLogQuery(c, common.DB.Model(&model.Log{}).Where("user_id = ?", userId))
	db.Count(&total)
	if err := db.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&logs).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取日志失败")
		return
	}
	common.OK(c, gin.H{"data": logs, "total": total, "page": page, "size": pageSize})
}

func SearchAllLogs(c *gin.Context) {
	GetAllLogs(c)
}

func SearchUserLogs(c *gin.Context) {
	GetUserLogs(c)
}

func buildLogQuery(c *gin.Context, db *gorm.DB) *gorm.DB {
	if v := c.Query("username"); v != "" {
		db = db.Where("username LIKE ?", "%"+v+"%")
	}
	if v := c.Query("token_name"); v != "" {
		db = db.Where("token_name LIKE ?", "%"+v+"%")
	}
	if v := c.Query("model_name"); v != "" {
		db = db.Where("model_name LIKE ?", "%"+v+"%")
	}
	if v := c.Query("content"); v != "" {
		db = db.Where("content LIKE ?", "%"+v+"%")
	}
	if v := c.Query("type"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			db = db.Where("type = ?", n)
		}
	}
	if v := c.Query("user_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			db = db.Where("user_id = ?", n)
		}
	}
	channelVal := c.Query("channel_id")
	if channelVal == "" {
		channelVal = c.Query("channel")
	}
	if channelVal != "" {
		if n, err := strconv.Atoi(channelVal); err == nil {
			db = db.Where("channel_id = ?", n)
		}
	}
	startVal := c.Query("start_time")
	if startVal == "" {
		startVal = c.Query("start_timestamp")
	}
	if startVal != "" {
		if n, err := strconv.ParseInt(startVal, 10, 64); err == nil {
			db = db.Where("created_at >= ?", n)
		} else if t, err := time.Parse("2006-01-02", startVal); err == nil {
			db = db.Where("created_at >= ?", t.Unix())
		}
	}
	endVal := c.Query("end_time")
	if endVal == "" {
		endVal = c.Query("end_timestamp")
	}
	if endVal != "" {
		if n, err := strconv.ParseInt(endVal, 10, 64); err == nil {
			db = db.Where("created_at <= ?", n)
		} else if t, err := time.Parse("2006-01-02", endVal); err == nil {
			db = db.Where("created_at <= ?", t.Add(24*time.Hour).Unix())
		}
	}
	if v := c.Query("min_quota"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			db = db.Where("quota >= ?", n)
		}
	}
	if v := c.Query("max_quota"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			db = db.Where("quota <= ?", n)
		}
	}
	return db
}

type LogStatItem struct {
	ModelName        string `json:"model_name"`
	TotalQuota       int64  `json:"total_quota"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	RequestCount     int64  `json:"request_count"`
}

func GetLogStat(c *gin.Context) {
	db := buildLogQuery(c, common.DB.Model(&model.Log{}).Where("type = ?", model.LogTypeConsume))
	var items []LogStatItem
	if err := db.Select("model_name, SUM(quota) as total_quota, SUM(prompt_tokens) as prompt_tokens, SUM(completion_tokens) as completion_tokens, COUNT(*) as request_count").
		Group("model_name").Order("total_quota desc").Scan(&items).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取统计失败")
		return
	}
	var totalQuota int64
	for _, it := range items {
		totalQuota += it.TotalQuota
	}
	common.OK(c, gin.H{"items": items, "total_quota": totalQuota})
}

func GetUserLogStat(c *gin.Context) {
	userId, _ := c.Get("id")
	db := buildLogQuery(c, common.DB.Model(&model.Log{}).Where("type = ? AND user_id = ?", model.LogTypeConsume, userId))
	var items []LogStatItem
	if err := db.Select("model_name, SUM(quota) as total_quota, SUM(prompt_tokens) as prompt_tokens, SUM(completion_tokens) as completion_tokens, COUNT(*) as request_count").
		Group("model_name").Order("total_quota desc").Scan(&items).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取统计失败")
		return
	}
	var totalQuota int64
	for _, it := range items {
		totalQuota += it.TotalQuota
	}
	common.OK(c, gin.H{"items": items, "total_quota": totalQuota})
}

func GetLogsStat(c *gin.Context) {
	GetLogStat(c)
}

func GetLogsSelfStat(c *gin.Context) {
	GetUserLogStat(c)
}

func DeleteLogs(c *gin.Context) {
	before := int64(0)
	if v := c.Query("target_timestamp"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			before = n
		}
	}
	if before <= 0 {
		var req struct {
			BeforeTimestamp int64 `json:"before_timestamp"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			before = req.BeforeTimestamp
		}
	}
	if before <= 0 {
		common.Fail(c, common.CodeParamError, "请提供有效的时间戳")
		return
	}
	result := common.DB.Where("created_at < ?", before).Delete(&model.Log{})
	if result.Error != nil {
		common.Fail(c, common.CodeServerError, "删除日志失败")
		return
	}
	common.OKMsg(c, "日志清理成功", gin.H{"deleted": result.RowsAffected})
}

func DeleteHistoryLogs(c *gin.Context) {
	DeleteLogs(c)
}
