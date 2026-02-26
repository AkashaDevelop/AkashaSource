package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func ExportLogsCSV(c *gin.Context) {
	var logs []model.Log
	db := buildLogQuery(c, common.DB.Model(&model.Log{}))
	if err := db.Order("id desc").Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导出失败"})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=logs.csv")
	c.Header("Content-Type", "text/csv; charset=utf-8")
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"id", "user_id", "username", "type", "content", "model", "quota", "created_at"})
	for _, item := range logs {
		_ = writer.Write([]string{
			strconv.Itoa(item.Id),
			strconv.Itoa(item.UserId),
			item.Username,
			strconv.Itoa(item.Type),
			item.Content,
			item.ModelName,
			strconv.FormatInt(item.Quota, 10),
			time.Unix(item.CreatedAt, 0).Format("2006-01-02 15:04:05"),
		})
	}
	writer.Flush()
}

func ExportRedemptionsCSV(c *gin.Context) {
	var items []model.Redemption
	db := common.DB.Model(&model.Redemption{})
	if status := c.Query("status"); status != "" {
		if v, err := strconv.Atoi(status); err == nil {
			db = db.Where("status = ?", v)
		}
	}
	if createdBy := c.Query("created_by"); createdBy != "" {
		if v, err := strconv.Atoi(createdBy); err == nil {
			db = db.Where("created_by = ?", v)
		}
	}
	if usedBy := c.Query("used_by"); usedBy != "" {
		if v, err := strconv.Atoi(usedBy); err == nil {
			db = db.Where("used_by = ?", v)
		}
	}
	if err := db.Order("id desc").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导出失败"})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=redemptions.csv")
	c.Header("Content-Type", "text/csv; charset=utf-8")
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"id", "code", "quota", "status", "created_at", "used_at", "created_by", "used_by"})
	for _, item := range items {
		_ = writer.Write([]string{
			strconv.Itoa(item.Id),
			item.Code,
			strconv.FormatInt(item.Quota, 10),
			strconv.Itoa(item.Status),
			time.Unix(item.CreatedAt, 0).Format("2006-01-02 15:04:05"),
			time.Unix(item.UsedAt, 0).Format("2006-01-02 15:04:05"),
			strconv.Itoa(item.CreatedBy),
			strconv.Itoa(item.UsedBy),
		})
	}
	writer.Flush()
}
