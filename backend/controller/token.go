package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"net/http"
	"strconv"
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

func GetTokenUsage(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "No Authorization header"})
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid Bearer token"})
		return
	}

	tokenKey := strings.TrimSpace(parts[1])
	if tokenKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid Bearer token"})
		return
	}
	if !strings.HasPrefix(tokenKey, "sk-") {
		tokenKey = "sk-" + tokenKey
	}

	var token model.Token
	if err := common.DB.Where("key = ?", tokenKey).First(&token).Error; err != nil {
		common.Fail(c, common.CodeUnauthorized, "无效的 API Key")
		return
	}
	if err := ValidateToken(&token); err != nil {
		common.Fail(c, common.CodeUnauthorized, err.Error())
		return
	}

	expiresAt := token.ExpiredTime
	if expiresAt == -1 {
		expiresAt = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    true,
		"message": "ok",
		"data": gin.H{
			"object":          "token_usage",
			"name":            token.Name,
			"total_granted":   token.RemainQuota + token.UsedQuota,
			"total_used":      token.UsedQuota,
			"total_available": token.RemainQuota,
			"unlimited_quota": token.UnlimitedQuota,
			"expires_at":      expiresAt,
		},
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

func SearchTokens(c *gin.Context) {
	userId, _ := c.Get("id")
	role, _ := c.Get("role")

	page, _ := strconv.Atoi(c.DefaultQuery("p", c.DefaultQuery("page", "1")))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	if size > 100 {
		size = 100
	}

	db := common.DB.Model(&model.Token{})
	if role.(int) < model.RoleAdmin {
		db = db.Where("user_id = ?", userId)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		db = db.Where("name LIKE ?", "%"+keyword+"%")
	}
	if token := strings.TrimSpace(c.Query("token")); token != "" {
		db = db.Where("`key` LIKE ?", "%"+token+"%")
	}

	var total int64
	db.Count(&total)

	var tokens []model.Token
	if err := db.Order("id desc").Limit(size).Offset((page - 1) * size).Find(&tokens).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取令牌失败")
		return
	}
	common.OK(c, gin.H{"data": tokens, "total": total, "page": page, "size": size})
}

func GetToken(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.Fail(c, common.CodeParamError, "无效的令牌 ID")
		return
	}

	userId, _ := c.Get("id")
	role, _ := c.Get("role")

	db := common.DB.Where("id = ?", id)
	if role.(int) < model.RoleAdmin {
		db = db.Where("user_id = ?", userId)
	}

	var token model.Token
	if err := db.First(&token).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "令牌不存在")
		return
	}
	common.OK(c, token)
}

type TokenBatchRequest struct {
	Ids []int `json:"ids"`
}

func DeleteTokenBatch(c *gin.Context) {
	var req TokenBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Ids) == 0 {
		common.Fail(c, common.CodeParamError, "参数无效")
		return
	}

	userId, _ := c.Get("id")
	role, _ := c.Get("role")

	db := common.DB.Model(&model.Token{}).Where("id IN ?", req.Ids)
	if role.(int) < model.RoleAdmin {
		db = db.Where("user_id = ?", userId)
	}
	result := db.Delete(&model.Token{})
	if result.Error != nil {
		common.Fail(c, common.CodeServerError, "批量删除令牌失败")
		return
	}
	common.OK(c, gin.H{"deleted": result.RowsAffected})
}
