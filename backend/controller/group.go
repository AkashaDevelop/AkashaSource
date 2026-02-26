package controller

import (
	"net/http"
	"time"

	"STfreApi/common"
	"STfreApi/model"

	"github.com/gin-gonic/gin"
)

func GetAllGroups(c *gin.Context) {
	var groups []model.Group
	if err := common.DB.Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取分组失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": groups})
}

func AddGroup(c *gin.Context) {
	var group model.Group
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	group.CreatedAt = time.Now().Unix()
	if err := common.DB.Create(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建分组失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "分组创建成功", "data": group})
}

func UpdateGroup(c *gin.Context) {
	var group model.Group
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if group.Id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID 必填"})
		return
	}
	if err := common.DB.Model(&group).Updates(group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新分组失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "分组更新成功"})
}

func DeleteGroup(c *gin.Context) {
	id := c.Param("id")
	if err := common.DB.Delete(&model.Group{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除分组失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "分组删除成功"})
}
