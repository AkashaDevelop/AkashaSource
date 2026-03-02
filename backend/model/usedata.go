package model

import "STfreApi/common"

type QuotaData struct {
	Id        int    `json:"id"`
	UserID    int    `json:"user_id"`
	Username  string `json:"username"`
	ModelName string `json:"model_name"`
	CreatedAt int64  `json:"created_at"`
	TokenUsed int    `json:"token_used"`
	Count     int    `json:"count"`
	Quota     int    `json:"quota"`
}

func GetQuotaDataByUserId(userId int, startTime int64, endTime int64) ([]*QuotaData, error) {
	var rows []*QuotaData
	dateExpr := "created_at - (created_at % 3600)"
	err := common.DB.Model(&Log{}).
		Select("0 as id, user_id, username, model_name, "+dateExpr+" as created_at, SUM(prompt_tokens + completion_tokens + cached_tokens) as token_used, COUNT(*) as count, SUM(quota) as quota").
		Where("user_id = ? AND created_at >= ? AND created_at <= ? AND type = ?", userId, startTime, endTime, LogTypeConsume).
		Group("user_id, username, model_name, " + dateExpr).
		Order("created_at DESC").
		Scan(&rows).Error
	return rows, err
}

func GetAllQuotaDates(startTime int64, endTime int64, username string) ([]*QuotaData, error) {
	if username != "" {
		var rows []*QuotaData
		dateExpr := "created_at - (created_at % 3600)"
		err := common.DB.Model(&Log{}).
			Select("0 as id, 0 as user_id, username, model_name, "+dateExpr+" as created_at, SUM(prompt_tokens + completion_tokens + cached_tokens) as token_used, COUNT(*) as count, SUM(quota) as quota").
			Where("username = ? AND created_at >= ? AND created_at <= ? AND type = ?", username, startTime, endTime, LogTypeConsume).
			Group("username, model_name, " + dateExpr).
			Order("created_at DESC").
			Scan(&rows).Error
		return rows, err
	}

	var rows []*QuotaData
	dateExpr := "created_at - (created_at % 3600)"
	err := common.DB.Model(&Log{}).
		Select("0 as id, 0 as user_id, '' as username, model_name, "+dateExpr+" as created_at, SUM(prompt_tokens + completion_tokens + cached_tokens) as token_used, COUNT(*) as count, SUM(quota) as quota").
		Where("created_at >= ? AND created_at <= ? AND type = ?", startTime, endTime, LogTypeConsume).
		Group("model_name, " + dateExpr).
		Order("created_at DESC").
		Scan(&rows).Error
	return rows, err
}
