package controller

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"STfreApi/common"
	"STfreApi/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CheckInRequest struct {
	Turnstile string                          `json:"turnstile"`
	GeeTest   *common.GeeTestValidateRequest  `json:"geetest"`
}

func CheckIn(c *gin.Context) {
	userId, _ := c.Get("id")

	// Check if enabled
	common.OptionLock.RLock()
	enabled := common.OptionMap["checkin_enabled"] == "true"
	minRewardStr := common.OptionMap["checkin_min_reward"]
	maxRewardStr := common.OptionMap["checkin_max_reward"]
	common.OptionLock.RUnlock()

	if !enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "签到功能未开启"})
		return
	}

	// Captcha verification if configured
	if common.CheckinCaptcha {
		var req CheckInRequest
		if c.Request.Body != nil {
			json.NewDecoder(c.Request.Body).Decode(&req)
		}
		if !common.VerifyCaptcha(req.Turnstile, req.GeeTest) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "人机验证失败"})
			return
		}
	}

	minReward, _ := strconv.ParseInt(minRewardStr, 10, 64)
	maxReward, _ := strconv.ParseInt(maxRewardStr, 10, 64)
	if minReward <= 0 {
		minReward = 1000
	}
	if maxReward <= minReward {
		maxReward = minReward + 1000
	}

	today := time.Now().Format("2006-01-02")
	uid := userId.(int)

	// Check if already checked in
	var count int64
	common.DB.Model(&model.CheckIn{}).
		Where("user_id = ? AND check_date = ?", uid, today).
		Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "今日已签到"})
		return
	}

	reward := minReward + rand.Int63n(maxReward-minReward+1)

	err := common.DB.Transaction(func(tx *gorm.DB) error {
		checkin := model.CheckIn{
			UserId:    uid,
			CheckDate: today,
			Reward:    reward,
			CreatedAt: time.Now().Unix(),
		}
		if err := tx.Create(&checkin).Error; err != nil {
			return err
		}
		return tx.Model(&model.User{}).Where("id = ?", uid).
			Update("quota", gorm.Expr("quota + ?", reward)).Error
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "签到失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "签到成功",
		"reward":  reward,
	})
}

func GetCheckInStatus(c *gin.Context) {
	userId, _ := c.Get("id")
	today := time.Now().Format("2006-01-02")

	var count int64
	common.DB.Model(&model.CheckIn{}).
		Where("user_id = ? AND check_date = ?", userId.(int), today).
		Count(&count)

	c.JSON(http.StatusOK, gin.H{
		"checked_in": count > 0,
	})
}
