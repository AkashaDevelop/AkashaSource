package oauth

import (
	"STfreApi/common"
	"STfreApi/model"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

// TelegramCallback handles Telegram Login Widget callback.
// Telegram uses hash verification instead of standard OAuth code flow.
func TelegramCallback(c *gin.Context) {
	if common.TelegramBotToken == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Telegram login not configured"})
		return
	}

	hash := c.Query("hash")
	if hash == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Invalid request"})
		return
	}

	// Collect all query params except hash
	params := []string{}
	for key, values := range c.Request.URL.Query() {
		if key == "hash" {
			continue
		}
		params = append(params, fmt.Sprintf("%s=%s", key, values[0]))
	}
	sort.Strings(params)
	dataCheckString := strings.Join(params, "\n")

	// Verify hash: SHA256(bot_token) as secret key, HMAC-SHA256 of data
	secretKey := sha256.Sum256([]byte(common.TelegramBotToken))
	mac := hmac.New(sha256.New, secretKey[:])
	mac.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(mac.Sum(nil))

	if expectedHash != hash {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Invalid hash"})
		return
	}

	telegramId := c.Query("id")
	firstName := c.Query("first_name")
	lastName := c.Query("last_name")
	username := c.Query("username")

	if telegramId == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Missing telegram id"})
		return
	}

	displayName := firstName
	if lastName != "" {
		displayName += " " + lastName
	}
	if displayName == "" {
		displayName = username
	}

	// Find or create user
	var user model.User
	if err := common.DB.Where("telegram_id = ?", telegramId).First(&user).Error; err != nil {
		user = model.User{
			Username:    fmt.Sprintf("tg_%s", telegramId),
			DisplayName: displayName,
			TelegramId:  telegramId,
			Role:        model.RoleUser,
			Status:      model.UserStatusActive,
			Quota:       0,
		}
		if err := common.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to create user: " + err.Error()})
			return
		}
	} else {
		if user.Status == model.UserStatusBanned {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "User is banned"})
			return
		}
	}

	token, err := common.GenerateToken(user.Id, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to generate token"})
		return
	}
	c.Redirect(http.StatusFound, fmt.Sprintf("/?token=%s", token))
}
