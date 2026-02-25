package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func GetAllRedemptions(c *gin.Context) {
	var redemptions []model.Redemption
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	offset := (page - 1) * size

	var total int64
	common.DB.Model(&model.Redemption{}).Count(&total)

	if err := common.DB.Limit(size).Offset(offset).Order("id desc").Find(&redemptions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch redemptions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  redemptions,
		"total": total,
	})
}

type GenerateRedemptionRequest struct {
	Name  string `json:"name"`
	Quota int64  `json:"quota"`
	Count int    `json:"count"`
}

func GenerateRedemptionCodes(c *gin.Context) {
	var req GenerateRedemptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId, _ := c.Get("id")
	adminId := userId.(int)

	var codes []string
	for i := 0; i < req.Count; i++ {
		code := common.GenerateKey()
		redemption := model.Redemption{
			Name:      req.Name,
			Code:      code,
			Quota:     req.Quota,
			CreatedBy: adminId,
			Status:    model.RedemptionStatusUnused,
		}
		if err := common.DB.Create(&redemption).Error; err != nil {
			continue
		}
		codes = append(codes, code)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Codes generated successfully",
		"data":    codes,
	})
}

type UseRedemptionRequest struct {
	Code string `json:"code" binding:"required"`
}

func UseRedemptionCode(c *gin.Context) {
	var req UseRedemptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId, _ := c.Get("id")
	uid := userId.(int)

	var redemption model.Redemption
	if err := common.DB.Where("code = ?", req.Code).First(&redemption).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid code"})
		return
	}

	if redemption.Status != model.RedemptionStatusUnused {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code already used or disabled"})
		return
	}

	tx := common.DB.Begin()

	var user model.User
	if err := tx.First(&user, uid).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
		return
	}

	user.Quota += redemption.Quota
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update quota"})
		return
	}

	redemption.Status = model.RedemptionStatusUsed
	redemption.UsedBy = uid
	redemption.UsedAt = time.Now().Unix()
	if err := tx.Save(&redemption).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update code status"})
		return
	}

	log := model.Log{
		UserId:    uid,
		CreatedAt: time.Now().Unix(),
		Type:      model.LogTypeTopup,
		Quota:     redemption.Quota,
		Content:   fmt.Sprintf("Used redemption code: %s", req.Code),
		TokenName: "System",
		ModelName: "Topup",
		Username:  user.Username,
	}
	
	if err := tx.Create(&log).Error; err != nil {
		// Log error but proceed?
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{
		"message": "Redemption successful", 
		"quota": redemption.Quota,
		"new_balance": user.Quota,
	})
}
