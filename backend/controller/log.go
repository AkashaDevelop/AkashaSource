package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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

	db := common.DB.Model(&model.Log{})

	// 简单的筛选
	if username := c.Query("username"); username != "" {
		db = db.Where("username LIKE ?", "%"+username+"%")
	}
	if tokenName := c.Query("token_name"); tokenName != "" {
		db = db.Where("token_name LIKE ?", "%"+tokenName+"%")
	}
	if modelName := c.Query("model_name"); modelName != "" {
		db = db.Where("model_name LIKE ?", "%"+modelName+"%")
	}

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

	db := common.DB.Model(&model.Log{}).Where("user_id = ?", userId)

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
