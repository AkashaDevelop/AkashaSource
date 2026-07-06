package controller

import (
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
	"STfreApi/service"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 本文件为中继流程的计费/结算辅助逻辑（预扣、结算、退款、日志记录），
// 并非独立的 relay 请求处理器；主中继入口见 relay.go。

// usageDetail 由调用者通过 c.Set("usage_detail", usage) 设置，用于传递额外计费维度
type usageDetail struct {
	ImageTokens           int
	AudioTokens           int
	AudioCompletionTokens int
	WebSearchCount        int
	FileSearchCount       int
}

// hasActiveSubscriptionQuota 检查用户是否有可用订阅额度
// 返回 (sub, true) 表示可从订阅扣减，否则返回 (零值, false)
func hasActiveSubscriptionQuota(userId int, finalQuota int64) (model.UserSubscription, bool) {
	var sub model.UserSubscription
	err := common.DB.Where("user_id = ? AND status = ? AND (expired_at = 0 OR expired_at > ?) AND quota > 0",
		userId, model.SubStatusActive, time.Now().Unix()).
		Order("id desc").
		First(&sub).Error
	if err != nil {
		return model.UserSubscription{}, false
	}
	if sub.UsedQuota+finalQuota <= sub.Quota {
		return sub, true
	}
	return model.UserSubscription{}, false
}

// getUserBillingPreference 读取用户个人的资金来源优先级
// 返回 "subscription_first"（默认）、"wallet_first"、"subscription_only" 或 "wallet_only"
func getUserBillingPreference(userId int) string {
	var user model.User
	if err := common.DB.Select("billing_preference").First(&user, userId).Error; err != nil {
		return BillingPreferenceSubscriptionFirst
	}
	switch user.BillingPreference {
	case BillingPreferenceSubscriptionFirst,
		BillingPreferenceWalletFirst,
		BillingPreferenceSubscriptionOnly,
		BillingPreferenceWalletOnly:
		return user.BillingPreference
	default:
		return BillingPreferenceSubscriptionFirst
	}
}

// deductFromWalletTx 在事务中从钱包扣减额度
func deductFromWalletTx(tx *gorm.DB, token *model.Token, finalQuota int64) {
	tx.Model(&model.User{}).Where("id = ?", token.UserId).Updates(map[string]interface{}{
		"used_quota": gorm.Expr("used_quota + ?", finalQuota),
		"quota":      gorm.Expr("quota - ?", finalQuota),
	})
	if !token.UnlimitedQuota {
		tx.Model(token).Updates(map[string]interface{}{
			"used_quota":    gorm.Expr("used_quota + ?", finalQuota),
			"remain_quota":  gorm.Expr("remain_quota - ?", finalQuota),
			"accessed_time": time.Now().Unix(),
		})
	} else {
		tx.Model(token).Updates(map[string]interface{}{
			"used_quota":    gorm.Expr("used_quota + ?", finalQuota),
			"accessed_time": time.Now().Unix(),
		})
	}
}

// deductFromPrioritySource 根据用户偏好从对应资金来源扣减额度
func deductFromPrioritySource(tx *gorm.DB, token *model.Token, finalQuota int64) error {
	priority := getUserBillingPreference(token.UserId)

	switch priority {
	case BillingPreferenceWalletOnly:
		var user model.User
		tx.First(&user, token.UserId)
		if user.Quota < finalQuota {
			return fmt.Errorf("钱包余额不足")
		}
		deductFromWalletTx(tx, token, finalQuota)
		return nil

	case BillingPreferenceSubscriptionOnly:
		if sub, ok := hasActiveSubscriptionQuota(token.UserId, finalQuota); ok {
			tx.Model(&sub).Update("used_quota", gorm.Expr("used_quota + ?", finalQuota))
			return nil
		}
		return fmt.Errorf("订阅额度不足")

	case BillingPreferenceWalletFirst:
		var user model.User
		tx.First(&user, token.UserId)
		if user.Quota >= finalQuota {
			deductFromWalletTx(tx, token, finalQuota)
			return nil
		}
		if sub, ok := hasActiveSubscriptionQuota(token.UserId, finalQuota); ok {
			tx.Model(&sub).Update("used_quota", gorm.Expr("used_quota + ?", finalQuota))
			return nil
		}
		return fmt.Errorf("余额和订阅额度均不足")

	default: // subscription_first
		if sub, ok := hasActiveSubscriptionQuota(token.UserId, finalQuota); ok {
			tx.Model(&sub).Update("used_quota", gorm.Expr("used_quota + ?", finalQuota))
			return nil
		}
		deductFromWalletTx(tx, token, finalQuota)
		return nil
	}
}

func RecordConsumeLog(c *gin.Context, token *model.Token, modelName string, promptTokens int, completionTokens int, cachedTokens ...int) {
	cached := 0
	if len(cachedTokens) > 0 {
		cached = cachedTokens[0]
	}

	// 从 context 获取额外计费信息（由调用者通过 c.Set("usage_detail", usage) 设置）
	detail, _ := c.Get("usage_detail")
	d, _ := detail.(usageDetail)

	// 查询用户分组，令牌自己有独立分组的话优先按令牌的算账～
	var userGroup string
	common.DB.Model(&model.User{}).Where("id = ?", token.UserId).Select("group").Scan(&userGroup)
	// auto 分组下具体命中了哪个候选分组，selector.go 已经通过 c.Set("billing_group", ...) 逐次记录，
	// 这里直接读那份"实际命中"的记录来算账，不能笼统按 auto 或用户自己的分组打马虎眼～
	billingGroup := c.GetString("billing_group")
	if billingGroup == "" {
		billingGroup = service.ResolveBillingGroup(userGroup, token.Group)
	}

	// 计算实际消耗
	finalQuota, otherInfo := service.CalculateQuota(
		modelName, promptTokens, completionTokens, cached,
		d.ImageTokens, d.AudioTokens, d.AudioCompletionTokens,
		d.WebSearchCount, d.FileSearchCount, userGroup, billingGroup,
	)

	// 从 context 获取额外信息（由调用者设置）
	useTime := c.GetInt("use_time")
	isStream := c.GetBool("is_stream")
	channelId := c.GetInt("channel_id")

	// 检查是否使用订阅额度（多资金来源：优先订阅）
	var username string
	common.DB.Model(&model.User{}).Select("username").Where("id = ?", token.UserId).Scan(&username)

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
		ChannelId:        channelId,
		UseTime:          useTime,
		IsStream:         isStream,
		Group:            billingGroup,
		Other:            otherInfo,
		IP:               c.ClientIP(),
		RequestId:        c.GetString("request_id"),
	}

	// 根据优先级从资金来源扣减
	common.DB.Transaction(func(tx *gorm.DB) error {
		if err := deductFromPrioritySource(tx, token, finalQuota); err != nil {
			return err
		}
		service.EnqueueLog(log)
		return nil
	})
}

// RecordConsumeWithBilling 使用预扣/退款模式结算
// preConsumedQuota 为请求前预扣的额度，usage 为实际用量
func RecordConsumeWithBilling(c *gin.Context, token *model.Token, modelName string,
	preConsumedQuota int64, usage *dto.Usage) {

	cached := usage.CachedTokens

	// 从 context 获取额外信息
	useTime := c.GetInt("use_time")
	isStream := c.GetBool("is_stream")
	channelId := c.GetInt("channel_id")

	// 查询用户分组，令牌自己有独立分组的话优先按令牌的算账～
	var userGroup string
	common.DB.Model(&model.User{}).Where("id = ?", token.UserId).Select("group").Scan(&userGroup)
	billingGroup := c.GetString("billing_group")
	if billingGroup == "" {
		billingGroup = service.ResolveBillingGroup(userGroup, token.Group)
	}

	// 计算实际消耗
	finalQuota, otherInfo := service.CalculateQuota(
		modelName, usage.PromptTokens, usage.CompletionTokens, cached,
		usage.ImageTokens, usage.AudioTokens, usage.AudioCompletionTokens,
		usage.WebSearchCount, usage.FileSearchCount, userGroup, billingGroup,
	)

	var username string
	common.DB.Model(&model.User{}).Select("username").Where("id = ?", token.UserId).Scan(&username)

	log := model.Log{
		UserId:           token.UserId,
		Username:         username,
		CreatedAt:        time.Now().Unix(),
		Type:             model.LogTypeConsume,
		Content:          fmt.Sprintf("使用了模型 %s", modelName),
		TokenName:        token.Name,
		ModelName:        modelName,
		Quota:            finalQuota,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		CachedTokens:     cached,
		ChannelId:        channelId,
		UseTime:          useTime,
		IsStream:         isStream,
		Group:            billingGroup,
		Other:            otherInfo,
		IP:               c.ClientIP(),
		RequestId:        c.GetString("request_id"),
	}

	// 根据用户偏好决定结算来源
	priority := getUserBillingPreference(token.UserId)

	switch priority {
	case BillingPreferenceWalletOnly:
		service.PostConsumeQuota(token, preConsumedQuota, finalQuota)
		service.EnqueueLog(log)
		return

	case BillingPreferenceSubscriptionOnly:
		if sub, ok := hasActiveSubscriptionQuota(token.UserId, finalQuota); ok {
			service.RefundQuota(token, preConsumedQuota)
			common.DB.Model(&sub).Update("used_quota", gorm.Expr("used_quota + ?", finalQuota))
			service.EnqueueLog(log)
			return
		}
		service.PostConsumeQuota(token, preConsumedQuota, finalQuota)
		service.EnqueueLog(log)
		return

	case BillingPreferenceWalletFirst:
		var user model.User
		common.DB.First(&user, token.UserId)
		if user.Quota+preConsumedQuota >= finalQuota {
			service.PostConsumeQuota(token, preConsumedQuota, finalQuota)
			service.EnqueueLog(log)
			return
		}
		if sub, ok := hasActiveSubscriptionQuota(token.UserId, finalQuota); ok {
			service.RefundQuota(token, preConsumedQuota)
			common.DB.Model(&sub).Update("used_quota", gorm.Expr("used_quota + ?", finalQuota))
			service.EnqueueLog(log)
			return
		}
		service.PostConsumeQuota(token, preConsumedQuota, finalQuota)
		service.EnqueueLog(log)
		return

	default: // subscription_first
		if sub, ok := hasActiveSubscriptionQuota(token.UserId, finalQuota); ok {
			service.RefundQuota(token, preConsumedQuota)
			common.DB.Model(&sub).Update("used_quota", gorm.Expr("used_quota + ?", finalQuota))
			service.EnqueueLog(log)
			return
		}
		service.PostConsumeQuota(token, preConsumedQuota, finalQuota)
		service.EnqueueLog(log)
	}
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
