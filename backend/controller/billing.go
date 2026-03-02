package controller

import (
	"net/http"
	"strings"

	"STfreApi/common"
	"STfreApi/model"

	"github.com/gin-gonic/gin"
)

type OpenAISubscriptionResponse struct {
	Object             string  `json:"object"`
	HasPaymentMethod   bool    `json:"has_payment_method"`
	SoftLimitUSD       float64 `json:"soft_limit_usd"`
	HardLimitUSD       float64 `json:"hard_limit_usd"`
	SystemHardLimitUSD float64 `json:"system_hard_limit_usd"`
	AccessUntil        int64   `json:"access_until"`
}

type OpenAIUsageResponse struct {
	Object     string  `json:"object"`
	TotalUsage float64 `json:"total_usage"`
}

func resolveBillingToken(c *gin.Context) (*model.Token, error) {
	tokenKey := c.GetHeader("x-api-key")
	if tokenKey == "" {
		authHeader := c.GetHeader("Authorization")
		tokenKey = strings.TrimPrefix(authHeader, "Bearer ")
	}
	if tokenKey == "" {
		tokenKey = c.Query("api_key")
	}
	if tokenKey == "" {
		tokenKey = c.Query("key")
	}
	if tokenKey == "" {
		return nil, http.ErrNoCookie
	}

	token, err := GetTokenByKey(tokenKey)
	if err != nil {
		return nil, err
	}
	if err := ValidateToken(token); err != nil {
		return nil, err
	}
	return token, nil
}

// GetSubscription implements OpenAI-compatible billing subscription endpoint.
func GetSubscription(c *gin.Context) {
	token, err := resolveBillingToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "invalid api key",
				"type":    "invalid_request_error",
				"code":    "invalid_api_key",
			},
		})
		return
	}

	var user model.User
	if err := common.DB.Where("id = ?", token.UserId).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"message": "internal error", "type": "server_error"},
		})
		return
	}

	remainQuota := token.RemainQuota
	if token.UnlimitedQuota {
		remainQuota = user.Quota
	}
	quota := remainQuota + token.UsedQuota
	amount := float64(quota) / common.QuotaPerUnit
	if token.UnlimitedQuota {
		amount = 100000000
	}

	expiredTime := token.ExpiredTime
	if expiredTime <= 0 {
		expiredTime = 0
	}

	subscription := OpenAISubscriptionResponse{
		Object:             "billing_subscription",
		HasPaymentMethod:   true,
		SoftLimitUSD:       amount,
		HardLimitUSD:       amount,
		SystemHardLimitUSD: amount,
		AccessUntil:        expiredTime,
	}
	c.JSON(http.StatusOK, subscription)
}

// GetUsage implements OpenAI-compatible billing usage endpoint.
func GetUsage(c *gin.Context) {
	token, err := resolveBillingToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "invalid api key",
				"type":    "invalid_request_error",
				"code":    "invalid_api_key",
			},
		})
		return
	}

	amount := float64(token.UsedQuota) / common.QuotaPerUnit
	usage := OpenAIUsageResponse{
		Object:     "list",
		TotalUsage: amount * 100,
	}
	c.JSON(http.StatusOK, usage)
}
