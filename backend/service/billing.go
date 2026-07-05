package service

import (
	"STfreApi/common"
	"STfreApi/model"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// PreConsumeQuota 预扣额度
// 在请求转发前调用，检查余额并预扣
// imageTokens, audioTokens 可选传入（无则传 0），用于预估图像/音频输入消耗
func PreConsumeQuota(token *model.Token, promptTokens int, maxTokens int, modelName string, imageTokens int, audioTokens int) (int64, error) {
	ratio := common.GetModelRatio(modelName)
	completionRatio := common.GetCompletionRatio(modelName)
	imageRatio := common.GetImageRatio(modelName)
	audioRatio := common.GetAudioRatio(modelName)

	// 查询用户分组，令牌自己有独立分组的话优先用令牌的～
	var userGroup string
	common.DB.Model(&model.User{}).Where("id = ?", token.UserId).Select("group").Scan(&userGroup)
	billingGroup := ResolveBillingGroup(userGroup, token.Group)
	groupRatio := GetUserGroupRatio(userGroup, billingGroup)
	if override, ok := GetGroupModelRatioOverride(billingGroup, modelName); ok {
		ratio = override
	}

	// 预估预扣额度
	// 预扣额度 = (max(promptTokens, 预估最小值) + imageTokens × imageRatio + audioTokens × audioRatio + maxTokens × completionRatio) × ratio × groupRatio
	const minEstimateTokens = 500
	promptEstimate := promptTokens
	if promptEstimate < minEstimateTokens {
		promptEstimate = minEstimateTokens
	}
	preConsumedQuota := int64((float64(promptEstimate)+float64(imageTokens)*imageRatio+float64(audioTokens)*audioRatio+float64(maxTokens)*completionRatio) * ratio * groupRatio)
	if preConsumedQuota < 1 {
		preConsumedQuota = 1
	}

	// 检查额度
	if !token.UnlimitedQuota && token.RemainQuota < preConsumedQuota {
		return 0, fmt.Errorf("token 额度不足，需要 %d，剩余 %d", preConsumedQuota, token.RemainQuota)
	}

	// 查询用户余额
	var user model.User
	common.DB.First(&user, token.UserId)
	if user.Quota < preConsumedQuota {
		return 0, fmt.Errorf("用户额度不足，需要 %d，剩余 %d", preConsumedQuota, user.Quota)
	}

	// 执行预扣
	err := common.DB.Transaction(func(tx *gorm.DB) error {
		// 扣减用户余额
		if err := tx.Model(&model.User{}).Where("id = ? AND quota >= ?", token.UserId, preConsumedQuota).
			Updates(map[string]interface{}{
				"quota": gorm.Expr("quota - ?", preConsumedQuota),
			}).Error; err != nil {
			return err
		}

		// 扣减 token 额度（非无限额度时）
		if !token.UnlimitedQuota {
			if err := tx.Model(token).Where("remain_quota >= ?", preConsumedQuota).
				Updates(map[string]interface{}{
					"remain_quota": gorm.Expr("remain_quota - ?", preConsumedQuota),
				}).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("预扣失败: %w", err)
	}

	log.Printf("[billing] 预扣 %d quota (token=%d, model=%s)", preConsumedQuota, token.Id, modelName)
	return preConsumedQuota, nil
}

// PostConsumeQuota 结算额度（多退少补）
// actualQuota 是实际消耗，preConsumedQuota 是预扣额度
func PostConsumeQuota(token *model.Token, preConsumedQuota int64, actualQuota int64) (int64, error) {
	delta := actualQuota - preConsumedQuota

	err := common.DB.Transaction(func(tx *gorm.DB) error {
		if delta > 0 {
			// 实际消耗 > 预扣，补扣差额
			if err := tx.Model(&model.User{}).Where("id = ?", token.UserId).
				Updates(map[string]interface{}{
					"used_quota": gorm.Expr("used_quota + ?", actualQuota),
					"quota":      gorm.Expr("quota - ?", delta),
				}).Error; err != nil {
				return err
			}
			if !token.UnlimitedQuota {
				tx.Model(token).Updates(map[string]interface{}{
					"used_quota":    gorm.Expr("used_quota + ?", actualQuota),
					"remain_quota":  gorm.Expr("remain_quota - ?", delta),
					"accessed_time": time.Now().Unix(),
				})
			} else {
				tx.Model(token).Updates(map[string]interface{}{
					"used_quota":    gorm.Expr("used_quota + ?", actualQuota),
					"accessed_time": time.Now().Unix(),
				})
			}
		} else if delta < 0 {
			// 实际消耗 < 预扣，退还差额
			refundAmount := -delta
			if err := tx.Model(&model.User{}).Where("id = ?", token.UserId).
				Updates(map[string]interface{}{
					"used_quota": gorm.Expr("used_quota + ?", actualQuota),
					"quota":      gorm.Expr("quota + ?", refundAmount),
				}).Error; err != nil {
				return err
			}
			if !token.UnlimitedQuota {
				tx.Model(token).Updates(map[string]interface{}{
					"used_quota":    gorm.Expr("used_quota + ?", actualQuota),
					"remain_quota":  gorm.Expr("remain_quota + ?", refundAmount),
					"accessed_time": time.Now().Unix(),
				})
			} else {
				tx.Model(token).Updates(map[string]interface{}{
					"used_quota":    gorm.Expr("used_quota + ?", actualQuota),
					"accessed_time": time.Now().Unix(),
				})
			}
		} else {
			// 实际消耗 = 预扣，只记录 used_quota
			tx.Model(&model.User{}).Where("id = ?", token.UserId).
				Update("used_quota", gorm.Expr("used_quota + ?", actualQuota))
			tx.Model(token).Updates(map[string]interface{}{
				"used_quota":    gorm.Expr("used_quota + ?", actualQuota),
				"accessed_time": time.Now().Unix(),
			})
		}
		return nil
	})

	return delta, err
}

// RefundQuota 退还预扣额度（请求失败时调用）
func RefundQuota(token *model.Token, preConsumedQuota int64) error {
	err := common.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).Where("id = ?", token.UserId).
			Update("quota", gorm.Expr("quota + ?", preConsumedQuota)).Error; err != nil {
			return err
		}
		if !token.UnlimitedQuota {
			tx.Model(token).Update("remain_quota", gorm.Expr("remain_quota + ?", preConsumedQuota))
		}
		return nil
	})

	if err != nil {
		log.Printf("[billing] 退款失败 token=%d quota=%d: %v", token.Id, preConsumedQuota, err)
		return err
	}

	log.Printf("[billing] 退还 %d quota (token=%d)", preConsumedQuota, token.Id)
	return nil
}

// CalculateQuota 计算实际消耗额度（不扣款）
// 返回 (quota, otherInfo JSON)
// 使用 shopspring/decimal 进行高精度计算，避免 float64 累加误差
// ownerGroup 是用户自己的分组（用来查"用户分组×使用分组"的专属折扣），
// billingGroup 是这次请求实际按哪个分组算账（令牌分组覆盖后的结果，auto 已经在调用方折算回用户分组）～
func CalculateQuota(modelName string, promptTokens, completionTokens, cachedTokens, imageTokens, audioTokens, audioCompletionTokens, webSearchCount, fileSearchCount int, ownerGroup string, billingGroup string) (int64, string) {
	groupRatio := GetUserGroupRatio(ownerGroup, billingGroup)
	modelRatioBase := common.GetModelRatio(modelName)
	if override, ok := GetGroupModelRatioOverride(billingGroup, modelName); ok {
		modelRatioBase = override
	}

	// 检查是否按次计费
	if price, ok := common.GetModelPrice(modelName); ok {
		// 按次计费：quota = price × QuotaPerUnit × groupRatio + 工具附加费
		dPrice := decimal.NewFromFloat(price)
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		dGroupRatio := decimal.NewFromFloat(groupRatio)
		dQuota := dPrice.Mul(dQuotaPerUnit).Mul(dGroupRatio)

		// 工具附加费
		toolSurcharge := int64(0)
		if webSearchCount > 0 {
			toolSurcharge += int64(float64(webSearchCount) * 1000)
		}
		if fileSearchCount > 0 {
			toolSurcharge += int64(float64(fileSearchCount) * 100)
		}

		finalQuota := dQuota.IntPart() + toolSurcharge
		otherInfo, _ := json.Marshal(map[string]interface{}{
			"model_price": price,
			"group_ratio": groupRatio,
			"use_price":   true,
		})
		return finalQuota, string(otherInfo)
	}

	// 否则走按量计费逻辑
	ratio := modelRatioBase
	completionRatio := common.GetCompletionRatio(modelName)
	cacheRatio := common.GetCacheRatio()
	imageRatio := common.GetImageRatio(modelName)
	audioRatio := common.GetAudioRatio(modelName)
	audioCompletionRatio := common.GetAudioCompletionRatio(modelName)

	// 分离图像/音频 token 后的纯文本 prompt
	basePrompt := promptTokens - cachedTokens - imageTokens - audioTokens
	if basePrompt < 0 {
		basePrompt = 0
	}

	dRatio := decimal.NewFromFloat(ratio)
	dCompletionRatio := decimal.NewFromFloat(completionRatio)
	dCacheRatio := decimal.NewFromFloat(cacheRatio)
	dGroupRatio := decimal.NewFromFloat(groupRatio)
	dImageRatio := decimal.NewFromFloat(imageRatio)
	dAudioRatio := decimal.NewFromFloat(audioRatio)
	dAudioCompRatio := decimal.NewFromFloat(audioCompletionRatio)

	dBasePrompt := decimal.NewFromInt(int64(basePrompt))
	dCached := decimal.NewFromInt(int64(cachedTokens))
	dImage := decimal.NewFromInt(int64(imageTokens))
	dAudio := decimal.NewFromInt(int64(audioTokens))
	dCompletion := decimal.NewFromInt(int64(completionTokens))
	dAudioCompletion := decimal.NewFromInt(int64(audioCompletionTokens))

	dPromptQuota := dBasePrompt.Add(dCached.Mul(dCacheRatio)).Add(dImage.Mul(dImageRatio)).Add(dAudio.Mul(dAudioRatio))
	dCompletionQuota := dCompletion.Mul(dCompletionRatio).Add(dAudioCompletion.Mul(dAudioCompRatio))
	dQuota := dPromptQuota.Add(dCompletionQuota).Mul(dRatio).Mul(dGroupRatio)

	// 工具调用附加费（以 Quota 为单位，不走倍率）
	toolSurcharge := int64(0)
	if webSearchCount > 0 {
		toolSurcharge += int64(float64(webSearchCount) * 1000) // 每次 Web Search = 1000 Quota ($0.002)
	}
	if fileSearchCount > 0 {
		toolSurcharge += int64(float64(fileSearchCount) * 100) // 每次 File Search = 100 Quota
	}
	finalQuota := dQuota.IntPart() + toolSurcharge

	otherInfo, _ := json.Marshal(map[string]interface{}{
		"model_ratio":            ratio,
		"completion_ratio":       completionRatio,
		"cache_ratio":            cacheRatio,
		"group_ratio":            groupRatio,
		"image_ratio":            imageRatio,
		"audio_ratio":            audioRatio,
		"audio_completion_ratio": audioCompletionRatio,
	})

	return finalQuota, string(otherInfo)
}
