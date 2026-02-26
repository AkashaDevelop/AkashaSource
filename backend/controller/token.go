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
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数解析失败"})
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

	// Default status
	if token.Status == 0 {
		token.Status = model.TokenStatusActive
	}

	// Default expiration
	if token.ExpiredTime == 0 {
		token.ExpiredTime = -1 // Never expire by default
	}

	if err := common.DB.Create(&token).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建令牌失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "令牌创建成功", "data": token})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除令牌失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "令牌删除成功"})
}

func UpdateToken(c *gin.Context) {
	var token model.Token
	if err := c.ShouldBindJSON(&token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数解析失败"})
		return
	}
	if token.Id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少ID"})
		return
	}

	userId, _ := c.Get("id")
	role, _ := c.Get("role")

	var existingToken model.Token
	db := common.DB.Where("id = ?", token.Id)
	if role.(int) < model.RoleAdmin {
		db = db.Where("user_id = ?", userId)
	}

	if err := db.First(&existingToken).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "令牌不存在"})
		return
	}

	// Allow updating specific fields
	existingToken.Name = token.Name
	existingToken.Status = token.Status
	existingToken.ExpiredTime = token.ExpiredTime
	existingToken.RemainQuota = token.RemainQuota
	existingToken.UnlimitedQuota = token.UnlimitedQuota
	existingToken.AllowedIPs = token.AllowedIPs
	existingToken.AllowedModels = token.AllowedModels

	if err := common.DB.Save(&existingToken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新令牌失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "令牌更新成功"})
}

// GetKeyInfo returns quota info for the given API key (neko-api-key-tool compatible)
func GetKeyInfo(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	key := strings.TrimPrefix(authHeader, "Bearer ")
	if key == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing api key"})
		return
	}

	token, err := GetTokenByKey(key)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
		return
	}

	var user model.User
	if err := common.DB.Where("id = ?", token.UserId).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"key":             token.Key,
		"name":            token.Name,
		"status":          token.Status,
		"quota":           user.Quota,
		"used_quota":      user.UsedQuota,
		"remain_quota":    token.RemainQuota,
		"unlimited_quota": token.UnlimitedQuota,
	})
}

func GetAllTokens(c *gin.Context) {
	var tokens []model.Token
	userId, _ := c.Get("id")
	role, _ := c.Get("role")

	db := common.DB.Order("id desc")

	// If not admin, only show own tokens
	if role.(int) < model.RoleAdmin {
		db = db.Where("user_id = ?", userId)
	}

	if err := db.Find(&tokens).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取令牌失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tokens})
}
