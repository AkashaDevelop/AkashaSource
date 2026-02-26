package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"errors"
	"strings"
	"time"
)

// RecordLog 记录日志
func RecordLog(userId int, logType int, content string) {
	log := model.Log{
		UserId:    userId,
		Type:      logType,
		Content:   content,
		CreatedAt: common.GetTimestamp(),
	}
	common.DB.Create(&log)
}

type TokenCache struct {
	Id             int    `json:"id"`
	UserId         int    `json:"user_id"`
	Status         int    `json:"status"`
	ExpiredTime    int64  `json:"expired_time"`
	AllowedIPs     string `json:"allowed_ips"`
	AllowedModels  string `json:"allowed_models"`
	Name           string `json:"name"`
	UnlimitedQuota bool   `json:"unlimited_quota"`
	RemainQuota    int64  `json:"remain_quota"`
}

func GetTokenByKey(tokenKey string) (*model.Token, error) {
	cacheKey := "token:" + tokenKey
	var cache TokenCache
	if common.GetCache(cacheKey, &cache) {
		return &model.Token{
			Id:             cache.Id,
			UserId:         cache.UserId,
			Status:         cache.Status,
			ExpiredTime:    cache.ExpiredTime,
			AllowedIPs:     cache.AllowedIPs,
			AllowedModels:  cache.AllowedModels,
			Name:           cache.Name,
			Key:            tokenKey,
			UnlimitedQuota: cache.UnlimitedQuota,
			RemainQuota:    cache.RemainQuota,
		}, nil
	}
	var token model.Token
	if err := common.DB.Where("key = ?", tokenKey).First(&token).Error; err != nil {
		return nil, err
	}
	common.SetCache(cacheKey, TokenCache{
		Id:             token.Id,
		UserId:         token.UserId,
		Status:         token.Status,
		ExpiredTime:    token.ExpiredTime,
		AllowedIPs:     token.AllowedIPs,
		AllowedModels:  token.AllowedModels,
		Name:           token.Name,
		UnlimitedQuota: token.UnlimitedQuota,
		RemainQuota:    token.RemainQuota,
	}, 5*time.Minute)
	return &token, nil
}

func CheckTokenExpired(token *model.Token) bool {
	if token.ExpiredTime == -1 {
		return false
	}
	return time.Now().Unix() > token.ExpiredTime
}

func CheckTokenIP(token *model.Token, ip string) bool {
	if token.AllowedIPs == "" {
		return true
	}
	items := strings.Split(token.AllowedIPs, ",")
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if item == ip {
			return true
		}
	}
	return false
}

func CheckTokenModel(token *model.Token, modelName string) bool {
	if token.AllowedModels == "" {
		return true
	}
	items := strings.Split(token.AllowedModels, ",")
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if item == modelName {
			return true
		}
	}
	return false
}

func ValidateToken(token *model.Token) error {
	if token.Status != model.TokenStatusActive {
		return errors.New("令牌已禁用")
	}
	if CheckTokenExpired(token) {
		return errors.New("令牌已过期")
	}
	return nil
}
