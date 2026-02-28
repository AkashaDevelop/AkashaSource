package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"fmt"
	"strconv"
	"strings"
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

// ─── Admin ───────────────────────────────────────────────────────────────────

func AdminGetAllInvitations(c *gin.Context) {
	p, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	status, _ := strconv.Atoi(c.DefaultQuery("status", "0"))
	if p < 1 { p = 1 }
	if size < 1 || size > 100 { size = 20 }

	query := common.DB.Model(&model.Invitation{})
	if status > 0 {
		query = query.Where("status = ?", status)
	}
	var total int64
	query.Count(&total)
	var invitations []model.Invitation
	query.Order("created_at desc").Offset((p - 1) * size).Limit(size).Find(&invitations)

	// Collect user IDs
	idSet := map[int]struct{}{}
	for _, inv := range invitations {
		if inv.InviterId > 0 { idSet[inv.InviterId] = struct{}{} }
		if inv.InviteeId > 0 { idSet[inv.InviteeId] = struct{}{} }
	}
	userMap := map[int]string{}
	if len(idSet) > 0 {
		ids := make([]int, 0, len(idSet))
		for id := range idSet { ids = append(ids, id) }
		var users []model.User
		common.DB.Select("id, username").Where("id IN ?", ids).Find(&users)
		for _, u := range users { userMap[u.Id] = u.Username }
	}

	type Row struct {
		model.Invitation
		InviterName string `json:"inviter_name"`
		InviteeName string `json:"invitee_name"`
	}
	rows := make([]Row, len(invitations))
	for i, inv := range invitations {
		rows[i] = Row{Invitation: inv, InviterName: userMap[inv.InviterId], InviteeName: userMap[inv.InviteeId]}
	}
	common.OK(c, gin.H{"data": rows, "total": total})
}

func AdminGenerateInvitations(c *gin.Context) {
	var req struct {
		Count   int      `json:"count"`
		MaxUses int      `json:"max_uses"`
		Codes   []string `json:"codes"` // optional custom codes
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if len(req.Codes) == 0 && req.Count <= 0 {
		common.Fail(c, common.CodeParamError, "请指定数量或自定义码")
		return
	}
	if req.Count > 500 { req.Count = 500 }
	maxUses := req.MaxUses
	if maxUses < 0 { maxUses = 0 }

	var items []model.Invitation
	if len(req.Codes) > 0 {
		for _, code := range req.Codes {
			code = strings.TrimSpace(code)
			if code == "" { continue }
			items = append(items, model.Invitation{Code: code, MaxUses: maxUses, Status: model.InvitationStatusUnused})
		}
	} else {
		for i := 0; i < req.Count; i++ {
			items = append(items, model.Invitation{Code: uuid.New().String(), MaxUses: maxUses, Status: model.InvitationStatusUnused})
		}
	}
	if len(items) == 0 {
		common.Fail(c, common.CodeParamError, "无有效邀请码")
		return
	}
	if err := common.DB.Create(&items).Error; err != nil {
		common.Fail(c, common.CodeServerError, "生成失败")
		return
	}
	common.OK(c, items)
}

func AdminDeleteInvitation(c *gin.Context) {
	id := c.Param("id")
	if err := common.DB.Delete(&model.Invitation{}, id).Error; err != nil {
		common.Fail(c, common.CodeServerError, "删除失败")
		return
	}
	common.OKMsg(c, "删除成功", nil)
}
