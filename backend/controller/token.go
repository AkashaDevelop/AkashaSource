package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func AddToken(c *gin.Context) {
	var token model.Token
	if err := c.ShouldBindJSON(&token); err != nil {
		common.Fail(c, common.CodeParamError, "参数解析失败")
		return
	}
	userId, _ := c.Get("id")
	token.UserId = userId.(int)
	if token.Name == "" {
		token.Name = "新令牌"
	}
	token.CreatedTime = time.Now().Unix()
	token.AccessedTime = 0
	token.Key = common.GenerateKey()
	if token.Status == 0 {
		token.Status = model.TokenStatusActive
	}
	if token.ExpiredTime == 0 {
		token.ExpiredTime = -1
	}
	if err := common.DB.Create(&token).Error; err != nil {
		common.Fail(c, common.CodeServerError, "创建令牌失败")
		return
	}
	common.OKMsg(c, "令牌创建成功", token)
}

func DeleteToken(c *gin.Context) {
	id := c.Param("id")
	userId, _ := c.Get("id")
	role, _ := c.Get("role")
	db := common.DB.Where("id = ?", id)
	if role.(int) < model.RoleAdmin {
		db = db.Where("user_id = ?", userId)
	}
	if err := db.Delete(&model.Token{}).Error; err != nil {
		common.Fail(c, common.CodeServerError, "删除令牌失败")
		return
	}
	common.OKMsg(c, "令牌删除成功", nil)
}

func UpdateToken(c *gin.Context) {
	var token model.Token
	if err := c.ShouldBindJSON(&token); err != nil {
		common.Fail(c, common.CodeParamError, "参数解析失败")
		return
	}
	if token.Id == 0 {
		common.Fail(c, common.CodeParamError, "缺少令牌 ID")
		return
	}
	userId, _ := c.Get("id")
	role, _ := c.Get("role")
	db := common.DB.Where("id = ?", token.Id)
	if role.(int) < model.RoleAdmin {
		db = db.Where("user_id = ?", userId)
	}
	var existing model.Token
	if err := db.First(&existing).Error; err != nil {
		c.JSON(http.StatusNotFound, common.R{Code: common.CodeNotFound, Msg: "令牌不存在"})
		return
	}
	existing.Name = token.Name
	existing.Status = token.Status
	existing.ExpiredTime = token.ExpiredTime
	existing.RemainQuota = token.RemainQuota
	existing.UnlimitedQuota = token.UnlimitedQuota
	existing.AllowedIPs = token.AllowedIPs
	existing.AllowedModels = token.AllowedModels
	if err := common.DB.Save(&existing).Error; err != nil {
		common.Fail(c, common.CodeServerError, "更新令牌失败")
		return
	}
	common.OKMsg(c, "令牌更新成功", nil)
}

func GetKeyInfo(c *gin.Context) {
	key := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if key == "" {
		common.Fail(c, common.CodeUnauthorized, "缺少 API Key")
		return
	}
	token, err := GetTokenByKey(key)
	if err != nil {
		common.Fail(c, common.CodeUnauthorized, "无效的 API Key")
		return
	}
	var user model.User
	if err := common.DB.Where("id = ?", token.UserId).First(&user).Error; err != nil {
		common.Fail(c, common.CodeServerError, "内部错误")
		return
	}
	common.OK(c, gin.H{
		"key": token.Key, "name": token.Name, "status": token.Status,
		"quota": user.Quota, "used_quota": user.UsedQuota,
		"remain_quota": token.RemainQuota, "unlimited_quota": token.UnlimitedQuota,
	})
}

func GetAllTokens(c *gin.Context) {
	userId, _ := c.Get("id")
	role, _ := c.Get("role")
	db := common.DB.Order("id desc")
	if role.(int) < model.RoleAdmin {
		db = db.Where("user_id = ?", userId)
	}
	var tokens []model.Token
	if err := db.Find(&tokens).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取令牌失败")
		return
	}
	common.OK(c, tokens)
}
