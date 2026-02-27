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
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	db := common.DB.Model(&model.Redemption{})
	if status := c.Query("status"); status != "" {
		if v, err := strconv.Atoi(status); err == nil {
			db = db.Where("status = ?", v)
		}
	}
	if keyword := c.Query("keyword"); keyword != "" {
		db = db.Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	db.Count(&total)
	var redemptions []model.Redemption
	if err := db.Limit(size).Offset((page-1)*size).Order("id desc").Find(&redemptions).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取兑换码失败")
		return
	}
	common.OK(c, gin.H{"data": redemptions, "total": total})
}

type GenerateRedemptionRequest struct {
	Name    string `json:"name"`
	Quota   int64  `json:"quota"`
	Count   int    `json:"count"`
	MaxUses int    `json:"max_uses"`
}

func GenerateRedemptionCodes(c *gin.Context) {
	var req GenerateRedemptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if req.MaxUses < 0 {
		req.MaxUses = 1
	}
	userId, _ := c.Get("id")
	adminId := userId.(int)
	var codes []string
	for i := 0; i < req.Count; i++ {
		code := common.GenerateKey()
		r := model.Redemption{
			Name: req.Name, Code: code, Quota: req.Quota,
			MaxUses: req.MaxUses, CreatedBy: adminId, Status: model.RedemptionStatusUnused,
		}
		if err := common.DB.Create(&r).Error; err != nil {
			continue
		}
		codes = append(codes, code)
	}
	common.OKMsg(c, "兑换码生成成功", codes)
}

func UpdateRedemptionStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "无效的 ID")
		return
	}
	var body struct{ Status int `json:"status"` }
	if err := c.ShouldBindJSON(&body); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	var r model.Redemption
	if err := common.DB.First(&r, id).Error; err != nil {
		c.JSON(http.StatusNotFound, common.R{Code: common.CodeNotFound, Msg: "兑换码不存在"})
		return
	}
	if r.Status == model.RedemptionStatusUsed && body.Status != model.RedemptionStatusUsed {
		common.Fail(c, common.CodeParamError, "已用完的兑换码不能修改状态")
		return
	}
	if err := common.DB.Model(&r).Update("status", body.Status).Error; err != nil {
		common.Fail(c, common.CodeServerError, "更新失败")
		return
	}
	common.OKMsg(c, "状态已更新", nil)
}

func BatchRedemptionAction(c *gin.Context) {
	var body struct {
		IDs    []int  `json:"ids"`
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.IDs) == 0 {
		common.Fail(c, common.CodeParamError, "参数无效")
		return
	}
	switch body.Action {
	case "enable":
		common.DB.Model(&model.Redemption{}).Where("id IN ? AND status = ?", body.IDs, model.RedemptionStatusDisabled).Update("status", model.RedemptionStatusUnused)
	case "disable":
		common.DB.Model(&model.Redemption{}).Where("id IN ? AND status = ?", body.IDs, model.RedemptionStatusUnused).Update("status", model.RedemptionStatusDisabled)
	case "delete":
		common.DB.Delete(&model.Redemption{}, body.IDs)
	default:
		common.Fail(c, common.CodeParamError, "未知操作")
		return
	}
	common.OKMsg(c, "操作成功", nil)
}

type UseRedemptionRequest struct {
	Code string `json:"code" binding:"required"`
}

func UseRedemptionCode(c *gin.Context) {
	var req UseRedemptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	userId, _ := c.Get("id")
	uid := userId.(int)
	var redemption model.Redemption
	if err := common.DB.Where("code = ?", req.Code).First(&redemption).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "兑换码无效")
		return
	}
	if redemption.Status == model.RedemptionStatusDisabled {
		common.Fail(c, common.CodeForbidden, "兑换码已禁用")
		return
	}
	if redemption.Status == model.RedemptionStatusUsed {
		common.Fail(c, common.CodeConflict, "兑换码已用完")
		return
	}
	tx := common.DB.Begin()
	var user model.User
	if err := tx.First(&user, uid).Error; err != nil {
		tx.Rollback()
		common.Fail(c, common.CodeServerError, "用户不存在")
		return
	}
	user.Quota += redemption.Quota
	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		common.Fail(c, common.CodeServerError, "更新额度失败")
		return
	}
	redemption.UsedCount++
	redemption.UsedBy = uid
	redemption.UsedAt = time.Now().Unix()
	if redemption.MaxUses > 0 && redemption.UsedCount >= redemption.MaxUses {
		redemption.Status = model.RedemptionStatusUsed
	}
	if err := tx.Save(&redemption).Error; err != nil {
		tx.Rollback()
		common.Fail(c, common.CodeServerError, "更新兑换码状态失败")
		return
	}
	tx.Create(&model.Log{
		UserId: uid, CreatedAt: time.Now().Unix(), Type: model.LogTypeTopup,
		Quota: redemption.Quota, Content: fmt.Sprintf("兑换码: %s", req.Code),
		TokenName: "System", ModelName: "Topup", Username: user.Username,
	})
	tx.Commit()
	common.OKMsg(c, "兑换成功", gin.H{"quota": redemption.Quota, "new_balance": user.Quota})
}
