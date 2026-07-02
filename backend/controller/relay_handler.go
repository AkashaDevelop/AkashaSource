package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RecordConsumeLog(c *gin.Context, token *model.Token, modelName string, promptTokens int, completionTokens int, cachedTokens ...int) {
	ratio := common.GetModelRatio(modelName)
	completionRatio := common.GetCompletionRatio(modelName)
	cacheRatio := common.GetCacheRatio()

	cached := 0
	if len(cachedTokens) > 0 {
		cached = cachedTokens[0]
	}

	// Cache billing: non-cached prompt tokens at full price, cached at discount
	nonCachedPrompt := promptTokens - cached
	if nonCachedPrompt < 0 {
		nonCachedPrompt = 0
	}

	quota := float64(nonCachedPrompt) * ratio
	quota += float64(cached) * ratio * cacheRatio
	quota += float64(completionTokens) * ratio * completionRatio

	finalQuota := int64(quota)

	common.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).Where("id = ?", token.UserId).
			Updates(map[string]interface{}{
				"used_quota": gorm.Expr("used_quota + ?", finalQuota),
				"quota":      gorm.Expr("quota - ?", finalQuota),
			}).Error; err != nil {
			return err
		}

		updates := map[string]interface{}{
			"used_quota":    gorm.Expr("used_quota + ?", finalQuota),
			"accessed_time": time.Now().Unix(),
		}
		if !token.UnlimitedQuota {
			updates["remain_quota"] = gorm.Expr("remain_quota - ?", finalQuota)
		}

		if err := tx.Model(token).Updates(updates).Error; err != nil {
			return err
		}

		var username string
		tx.Model(&model.User{}).Select("username").Where("id = ?", token.UserId).Scan(&username)

		log := model.Log{
			UserId:           token.UserId,
			Username:         username,
			CreatedAt:        time.Now().Unix(),
			Type:             model.LogTypeConsume,
			Content:          fmt.Sprintf("使用了模型 %s", modelName),
			TokenName:        token.Name,
			ModelName:        modelName,
			Quota:            finalQuota,
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			CachedTokens:     cached,
			IP:               c.ClientIP(),
			RequestId:        c.GetString("request_id"),
		}
		service.EnqueueLog(log)
		return nil
	})
}

func RecordFailLog(c *gin.Context, token *model.Token, modelName string, errContent string) {
	log := model.Log{
		UserId:    token.UserId,
		CreatedAt: time.Now().Unix(),
		Type:      model.LogTypeFail,
		Content:   errContent,
		TokenName: token.Name,
		ModelName: modelName,
		Quota:     0,
		IP:        c.ClientIP(),
		RequestId: c.GetString("request_id"),
	}
	service.EnqueueLog(log)
}
