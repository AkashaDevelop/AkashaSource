package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RecordConsumeLog(c *gin.Context, token *model.Token, modelName string, promptTokens int, completionTokens int) {
	// 计算配额
	ratio := common.GetModelRatio(modelName)
	completionRatio := common.GetCompletionRatio(modelName)
	
	// Quota Calculation
	// Base: 500000 quota = $1
	// 1 token = 1 quota (if ratio is 1)
	
	quota := float64(promptTokens) * ratio
	quota += float64(completionTokens) * ratio * completionRatio

	// Special handling for DALL-E (Image Generation)
	// If it's an image model, promptTokens already contains the estimated quota cost.
	// We should probably just use it directly or ensure Ratio is 1.0 for DALL-E in database.
	// Assuming Ratio is 1.0 for now.
	
	finalQuota := int64(quota)
	
	// Transaction for atomic update
	common.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Update User Quota & UsedQuota
		if err := tx.Model(&model.User{}).Where("id = ?", token.UserId).
			Updates(map[string]interface{}{
				"used_quota": gorm.Expr("used_quota + ?", finalQuota),
				"quota":      gorm.Expr("quota - ?", finalQuota),
			}).Error; err != nil {
			return err
		}

		// 2. Update Token Quota & AccessedTime
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

		// 3. Create Log
		log := model.Log{
			UserId:           token.UserId,
			CreatedAt:        time.Now().Unix(),
			Type:             model.LogTypeConsume,
			Content:          fmt.Sprintf("使用了模型 %s", modelName),
			TokenName:        token.Name,
			ModelName:        modelName,
			Quota:            finalQuota,
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
		}
		return tx.Create(&log).Error
	})
}
