package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DashboardStats struct {
	UserCount      int64   `json:"user_count"`
	ChannelCount   int64   `json:"channel_count"`
	RequestCount   int64   `json:"request_count"`
	TotalQuotaUsed float64 `json:"total_quota_used"`
	ActiveChannels int64   `json:"active_channels"`
}

func GetAdminDashboard(c *gin.Context) {
	stats := DashboardStats{}

	common.DB.Model(&model.User{}).Count(&stats.UserCount)
	common.DB.Model(&model.Channel{}).Count(&stats.ChannelCount)
	common.DB.Model(&model.Channel{}).Where("status = ?", model.ChannelStatusActive).Count(&stats.ActiveChannels)
	common.DB.Model(&model.Log{}).Count(&stats.RequestCount)

	var totalQuota int64
	common.DB.Model(&model.Log{}).Select("sum(quota)").Scan(&totalQuota)
	stats.TotalQuotaUsed = float64(totalQuota) / 500000.0 // Convert to USD

	// Get chart data (last 7 days requests)
	type DailyRequest struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}
	var dailyRequests []DailyRequest
	// SQLite syntax for date grouping (strftime)
	// For MySQL it would be DATE_FORMAT
	// Let's assume SQLite for now based on previous context, or use a generic approach if possible.
	// Since we are using GORM and likely SQLite (default in main.go), we use strftime.
	common.DB.Raw("SELECT strftime('%Y-%m-%d', datetime(created_at, 'unixepoch')) as date, count(*) as count FROM logs WHERE type = 1 GROUP BY date ORDER BY date DESC LIMIT 7").Scan(&dailyRequests)

	// Reverse to chronological order
	for i, j := 0, len(dailyRequests)-1; i < j; i, j = i+1, j-1 {
		dailyRequests[i], dailyRequests[j] = dailyRequests[j], dailyRequests[i]
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
		"chart": dailyRequests,
	})
}

func GetUserDashboard(c *gin.Context) {
	userId, _ := c.Get("id")
	
	var user model.User
	if err := common.DB.First(&user, userId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var tokenCount int64
	common.DB.Model(&model.Token{}).Where("user_id = ?", userId).Count(&tokenCount)

	var requestCount int64
	common.DB.Model(&model.Log{}).Where("user_id = ?", userId).Count(&requestCount)
	
	// Recent logs
	var logs []model.Log
	common.DB.Where("user_id = ?", userId).Order("id desc").Limit(5).Find(&logs)

	// Chart data for user
	type DailyUsage struct {
		Date  string `json:"date"`
		Usage float64 `json:"usage"`
	}
	var dailyUsage []DailyUsage
	common.DB.Raw("SELECT strftime('%Y-%m-%d', datetime(created_at, 'unixepoch')) as date, sum(quota)/500000.0 as usage FROM logs WHERE user_id = ? AND type = 1 GROUP BY date ORDER BY date DESC LIMIT 7", userId).Scan(&dailyUsage)
	
	// Reverse
	for i, j := 0, len(dailyUsage)-1; i < j; i, j = i+1, j-1 {
		dailyUsage[i], dailyUsage[j] = dailyUsage[j], dailyUsage[i]
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"username":   user.Username,
			"quota":      float64(user.Quota) / 500000.0,
			"used_quota": float64(user.UsedQuota) / 500000.0,
			"role":       user.Role,
		},
		"stats": gin.H{
			"token_count":   tokenCount,
			"request_count": requestCount,
		},
		"logs":  logs,
		"chart": dailyUsage,
	})
}
