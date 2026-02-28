package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Admin: list all MJ tasks
func AdminGetMJTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	var tasks []model.MidjourneyTask
	var total int64
	query := common.DB.Model(&model.MidjourneyTask{})

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)
	query.Order("id desc").Offset(offset).Limit(size).Find(&tasks)

	common.OK(c, gin.H{"tasks": tasks, "total": total})
}

// Admin: list all Suno tasks
func AdminGetSunoTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	var tasks []model.SunoTask
	var total int64
	query := common.DB.Model(&model.SunoTask{})

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)
	query.Order("id desc").Offset(offset).Limit(size).Find(&tasks)

	common.OK(c, gin.H{"tasks": tasks, "total": total})
}

// User: list own MJ tasks
func UserGetMJTasks(c *gin.Context) {
	userId := c.GetInt("id")
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	var tasks []model.MidjourneyTask
	var total int64
	common.DB.Model(&model.MidjourneyTask{}).Where("user_id = ?", userId).Count(&total)
	common.DB.Where("user_id = ?", userId).Order("id desc").Offset(offset).Limit(size).Find(&tasks)

	common.OK(c, gin.H{"tasks": tasks, "total": total})
}

// User: list own Suno tasks
func UserGetSunoTasks(c *gin.Context) {
	userId := c.GetInt("id")
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	var tasks []model.SunoTask
	var total int64
	common.DB.Model(&model.SunoTask{}).Where("user_id = ?", userId).Count(&total)
	common.DB.Where("user_id = ?", userId).Order("id desc").Offset(offset).Limit(size).Find(&tasks)

	common.OK(c, gin.H{"tasks": tasks, "total": total})
}
