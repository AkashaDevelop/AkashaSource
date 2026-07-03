package controller

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
	"STfreApi/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CheckInRequest struct {
	Turnstile string                        `json:"turnstile"`
	GeeTest   *common.GeeTestValidateRequest `json:"geetest"`
}

func CheckIn(c *gin.Context) {
	userId, _ := c.Get("id")
	common.OptionLock.RLock()
	enabled := common.OptionMap["checkin_enabled"] == "true"
	minRewardStr := common.OptionMap["checkin_min_reward"]
	maxRewardStr := common.OptionMap["checkin_max_reward"]
	common.OptionLock.RUnlock()
	if !enabled {
		common.Fail(c, common.CodeForbidden, "签到功能未开启")
		return
	}
	if common.CheckinCaptcha {
		var req CheckInRequest
		if c.Request.Body != nil {
			json.NewDecoder(c.Request.Body).Decode(&req)
		}
		if !common.VerifyCaptcha(req.Turnstile, req.GeeTest) {
			common.Fail(c, common.CodeParamError, "人机验证失败")
			return
		}
	}
	minReward, _ := strconv.ParseInt(minRewardStr, 10, 64)
	maxReward, _ := strconv.ParseInt(maxRewardStr, 10, 64)
	if minReward <= 0 { minReward = 1000 }
	if maxReward <= minReward { maxReward = minReward + 1000 }
	today := time.Now().Format("2006-01-02")
	uid := userId.(int)
	var count int64
	common.DB.Model(&model.CheckIn{}).Where("user_id = ? AND check_date = ?", uid, today).Count(&count)
	if count > 0 {
		common.Fail(c, common.CodeConflict, "今日已签到")
		return
	}
	reward := minReward + rand.Int63n(maxReward-minReward+1)
	err := common.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model.CheckIn{UserId: uid, CheckDate: today, Reward: reward, CreatedAt: time.Now().Unix()}).Error; err != nil {
			return err
		}
		return tx.Model(&model.User{}).Where("id = ?", uid).Update("quota", gorm.Expr("quota + ?", reward)).Error
	})
	if err != nil {
		common.Fail(c, common.CodeServerError, "签到失败")
		return
	}
	common.OKMsg(c, "签到成功", gin.H{"reward": reward})

	// ～签到成功了，给用户发个小通知哦～
	go func() {
		var user model.User
		if err := common.DB.Select("id,email,setting").First(&user, uid).Error; err != nil {
			return
		}
		setting := user.GetSetting()
		usd := float64(reward) / common.QuotaPerUnit
		notify := dto.NewNotify(
			dto.NotifyTypeSystem,
			"签到成功",
			fmt.Sprintf("今日签到奖励 $%.4f（quota: %d），再接再厉哦！", usd, reward),
			nil,
		)
		service.NotifyUser(user.Id, user.Email, setting, notify) //nolint:errcheck
	}()
}

func GetCheckInStatus(c *gin.Context) {
	userId, _ := c.Get("id")
	today := time.Now().Format("2006-01-02")
	var count int64
	common.DB.Model(&model.CheckIn{}).Where("user_id = ? AND check_date = ?", userId.(int), today).Count(&count)
	common.OK(c, gin.H{"checked_in": count > 0})
}
