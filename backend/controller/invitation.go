package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GenerateInvitationCode(c *gin.Context) {
	userId := c.GetInt("id")
	username := c.GetString("username")
	costStr := common.OptionMap[model.OptionKeyInvitationCost]
	cost, _ := strconv.ParseFloat(costStr, 64)
	costInt := int64(cost)
	var user model.User
	if err := common.DB.First(&user, userId).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取用户信息失败")
		return
	}
	code := uuid.New().String()
	invitation := model.Invitation{Code: code, InviterId: userId, Status: model.InvitationStatusUnused, Cost: cost}
	err := common.DB.Transaction(func(tx *gorm.DB) error {
		if costInt > 0 {
			if user.Quota < costInt {
				return fmt.Errorf("额度不足")
			}
			if err := tx.Model(&user).Update("quota", gorm.Expr("quota - ?", costInt)).Error; err != nil {
				return err
			}
			tx.Create(&model.Log{
				UserId: userId, Username: username, CreatedAt: time.Now().Unix(),
				Type: model.LogTypeConsume, Content: "生成邀请码", Quota: costInt, ModelName: "system",
			})
		}
		return tx.Create(&invitation).Error
	})
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	common.OKMsg(c, "邀请码生成成功", invitation)
}

func GetUserInvitationCodes(c *gin.Context) {
	userId := c.GetInt("id")
	var invitations []model.Invitation
	if err := common.DB.Where("inviter_id = ?", userId).Order("created_at desc").Find(&invitations).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取邀请码失败")
		return
	}
	common.OK(c, invitations)
}
