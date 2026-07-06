package controller

import (
	"STfreApi/common"
	"STfreApi/model"

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
	stats.TotalQuotaUsed = float64(totalQuota) / 500000.0

	dateSql := getDateFormatSql()
	type DailyCount struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}
	var dailyRequests, dailyUsers, dailyErrors []DailyCount
	common.DB.Raw("SELECT "+dateSql+" as date, count(*) as count FROM logs WHERE type IN (?, ?) GROUP BY date ORDER BY date DESC LIMIT 7", model.LogTypeConsume, model.LogTypeFail).Scan(&dailyRequests)
	common.DB.Raw("SELECT "+dateSql+" as date, count(distinct user_id) as count FROM logs WHERE type IN (?, ?) GROUP BY date ORDER BY date DESC LIMIT 7", model.LogTypeConsume, model.LogTypeFail).Scan(&dailyUsers)
	common.DB.Raw("SELECT "+dateSql+" as date, count(*) as count FROM logs WHERE type = ? GROUP BY date ORDER BY date DESC LIMIT 7", model.LogTypeFail).Scan(&dailyErrors)
	for _, s := range []*[]DailyCount{&dailyRequests, &dailyUsers, &dailyErrors} {
		for i, j := 0, len(*s)-1; i < j; i, j = i+1, j-1 {
			(*s)[i], (*s)[j] = (*s)[j], (*s)[i]
		}
	}
	common.OK(c, gin.H{
		"stats": stats, "chart": dailyRequests,
		"chart_dau": dailyUsers, "chart_error": dailyErrors,
	})
}

func GetUserDashboard(c *gin.Context) {
	userId, _ := c.Get("id")
	var user model.User
	if err := common.DB.First(&user, userId).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}
	var tokenCount, requestCount int64
	common.DB.Model(&model.Token{}).Where("user_id = ?", userId).Count(&tokenCount)
	common.DB.Model(&model.Log{}).Where("user_id = ?", userId).Count(&requestCount)
	var logs []model.Log
	common.DB.Where("user_id = ?", userId).Order("id desc").Limit(5).Find(&logs)
	type DailyUsage struct {
		Date  string  `json:"date"`
		Usage float64 `json:"usage"`
	}
	var dailyUsage []DailyUsage
	dateSql := getDateFormatSql()
	common.DB.Raw("SELECT "+dateSql+" as date, sum(quota)/500000.0 as usage FROM logs WHERE user_id = ? AND type = ? GROUP BY date ORDER BY date DESC LIMIT 7", userId, model.LogTypeConsume).Scan(&dailyUsage)
	for i, j := 0, len(dailyUsage)-1; i < j; i, j = i+1, j-1 {
		dailyUsage[i], dailyUsage[j] = dailyUsage[j], dailyUsage[i]
	}
	common.OK(c, gin.H{
		"user": gin.H{
			"username": user.Username, "quota": float64(user.Quota) / 500000.0,
			"used_quota": float64(user.UsedQuota) / 500000.0, "role": user.Role,
		},
		"stats": gin.H{"token_count": tokenCount, "request_count": requestCount},
		"logs":  logs, "chart": dailyUsage,
	})
}
