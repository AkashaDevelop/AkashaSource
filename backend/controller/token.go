package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func AddToken(c *gin.Context) {
	var token model.Token
	if err := c.ShouldBindJSON(&token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId, _ := c.Get("id")
	token.UserId = userId.(int)

	if token.Name == "" {
		token.Name = "New Token"
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token created successfully", "data": token})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Token deleted successfully"})
}

func UpdateToken(c *gin.Context) {
	var token model.Token
	if err := c.ShouldBindJSON(&token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if token.Id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		return
	}

	// Allow updating specific fields
	existingToken.Name = token.Name
	existingToken.Status = token.Status
	existingToken.ExpiredTime = token.ExpiredTime
	existingToken.RemainQuota = token.RemainQuota
	existingToken.UnlimitedQuota = token.UnlimitedQuota

	if err := common.DB.Save(&existingToken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Token updated successfully"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tokens"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tokens})
}
