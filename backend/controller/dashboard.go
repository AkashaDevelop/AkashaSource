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

func getDateFormatSql() string {
	switch common.DB.Dialector.Name() {
	case "sqlite":
		return "strftime('%Y-%m-%d', datetime(created_at, 'unixepoch'))"
	case "mysql":
		return "FROM_UNIXTIME(created_at, '%Y-%m-%d')"
	case "postgres":
		return "TO_CHAR(TO_TIMESTAMP(created_at), 'YYYY-MM-DD')"
	default:
		return "strftime('%Y-%m-%d', datetime(created_at, 'unixepoch'))"
	}
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

	dateSql := getDateFormatSql()

	// 1. Daily Requests (Success + Fail)
	type DailyRequest struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}
	var dailyRequests []DailyRequest
	common.DB.Raw("SELECT "+dateSql+" as date, count(*) as count FROM logs WHERE type IN (?, ?) GROUP BY date ORDER BY date DESC LIMIT 7", model.LogTypeConsume, model.LogTypeFail).Scan(&dailyRequests)

	// Reverse
	for i, j := 0, len(dailyRequests)-1; i < j; i, j = i+1, j-1 {
		dailyRequests[i], dailyRequests[j] = dailyRequests[j], dailyRequests[i]
	}

	// 2. Daily Active Users (DAU)
	type DailyUser struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}
	var dailyUsers []DailyUser
	common.DB.Raw("SELECT "+dateSql+" as date, count(distinct user_id) as count FROM logs WHERE type IN (?, ?) GROUP BY date ORDER BY date DESC LIMIT 7", model.LogTypeConsume, model.LogTypeFail).Scan(&dailyUsers)
	// Reverse
	for i, j := 0, len(dailyUsers)-1; i < j; i, j = i+1, j-1 {
		dailyUsers[i], dailyUsers[j] = dailyUsers[j], dailyUsers[i]
	}

	// 3. Daily Errors
	type DailyError struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}
	var dailyErrors []DailyError
	common.DB.Raw("SELECT "+dateSql+" as date, count(*) as count FROM logs WHERE type = ? GROUP BY date ORDER BY date DESC LIMIT 7", model.LogTypeFail).Scan(&dailyErrors)
	// Reverse
	for i, j := 0, len(dailyErrors)-1; i < j; i, j = i+1, j-1 {
		dailyErrors[i], dailyErrors[j] = dailyErrors[j], dailyErrors[i]
	}

	c.JSON(http.StatusOK, gin.H{
		"stats":       stats,
		"chart":       dailyRequests,
		"chart_dau":   dailyUsers,
		"chart_error": dailyErrors,
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
		Date  string  `json:"date"`
		Usage float64 `json:"usage"`
	}
	var dailyUsage []DailyUsage
	dateSql := getDateFormatSql()
	common.DB.Raw("SELECT "+dateSql+" as date, sum(quota)/500000.0 as usage FROM logs WHERE user_id = ? AND type = ? GROUP BY date ORDER BY date DESC LIMIT 7", userId, model.LogTypeConsume).Scan(&dailyUsage)

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
