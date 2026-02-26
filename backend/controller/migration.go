package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetSQLMigrations(c *gin.Context) {
	var applied []model.SQLMigration
	common.DB.Order("version asc").Find(&applied)
	c.JSON(http.StatusOK, gin.H{"data": applied})
}

func ApplySQLMigrations(c *gin.Context) {
	if err := model.ApplySQLFileMigrations(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "迁移执行失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "迁移执行完成"})
}

func RollbackSQLMigrations(c *gin.Context) {
	steps, _ := strconv.Atoi(c.DefaultQuery("steps", "1"))
	if steps <= 0 {
		steps = 1
	}
	if err := model.RollbackSQLFileMigrations(steps); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "回滚执行失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "回滚执行完成"})
}
