package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GenerateInvitationCode 生成邀请码
func GenerateInvitationCode(c *gin.Context) {
	userId := c.GetInt("id")
	username := c.GetString("username")

	// 1. Check if invitation is enabled (optional, maybe user wants to generate even if not strictly required?)
	// Let's assume always enabled for now, or check option.

	// 2. Check cost
	costStr := common.OptionMap[model.OptionKeyInvitationCost]
	cost, _ := strconv.ParseFloat(costStr, 64)
	costInt := int64(cost)

	var user model.User
	if err := common.DB.First(&user, userId).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}

	// 3. Transaction: Deduct balance & Create code
	code := uuid.New().String()
	invitation := model.Invitation{
		Code:      code,
		InviterId: userId,
		Status:    model.InvitationStatusUnused,
		Cost:      cost,
	}

	err := common.DB.Transaction(func(tx *gorm.DB) error {
		// Deduct quota if cost > 0
		if costInt > 0 {
			if user.Quota < costInt {
				return fmt.Errorf("insufficient quota")
			}
			// Update user quota
			if err := tx.Model(&user).Update("quota", gorm.Expr("quota - ?", costInt)).Error; err != nil {
				return err
			}
			// Log consumption
			log := model.Log{
				UserId:    userId,
				Username:  username,
				CreatedAt: time.Now().Unix(),
				Type:      model.LogTypeConsume,
				Content:   "生成邀请码",
				Quota:     costInt,
				ModelName: "system",
			}
			if err := tx.Create(&log).Error; err != nil {
				return err
			}
		}

		// Create invitation
		return tx.Create(&invitation).Error
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Invitation code generated successfully",
		"data":    invitation,
	})
}

// GetUserInvitationCodes 获取用户的邀请码列表
func GetUserInvitationCodes(c *gin.Context) {
	userId := c.GetInt("id")
	var invitations []model.Invitation
	
	// Pagination could be added here
	if err := common.DB.Where("inviter_id = ?", userId).Order("created_at desc").Find(&invitations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch invitations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": invitations})
}
