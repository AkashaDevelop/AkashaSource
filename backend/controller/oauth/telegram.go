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

	// Verify HMAC-SHA256 hash
	params := []string{}
	for key, values := range c.Request.URL.Query() {
		if key == "hash" {
			continue
		}
		params = append(params, fmt.Sprintf("%s=%s", key, values[0]))
	}
	sort.Strings(params)
	dataCheckString := strings.Join(params, "\n")

	secretKey := sha256.Sum256([]byte(common.TelegramBotToken))
	mac := hmac.New(sha256.New, secretKey[:])
	mac.Write([]byte(dataCheckString))
	if hex.EncodeToString(mac.Sum(nil)) != hash {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Invalid hash"})
		return
	}

	telegramId := c.Query("id")
	if telegramId == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Missing telegram id"})
		return
	}

	firstName := c.Query("first_name")
	lastName := c.Query("last_name")
	username := c.Query("username")
	displayName := strings.TrimSpace(firstName + " " + lastName)
	if displayName == "" {
		displayName = username
	}

	user, pendingSessionID, err := createOAuthUser("telegram_id", telegramId, func() model.User {
		return model.User{
			Username:    fmt.Sprintf("tg_%s", telegramId),
			DisplayName: displayName,
			TelegramId:  telegramId,
			Role:        model.RoleUser,
			Status:      model.UserStatusActive,
		}
	}, "telegram")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if pendingSessionID != "" {
		c.Redirect(http.StatusFound, fmt.Sprintf("/oauth/pending?oauth_pending=%s", pendingSessionID))
		return
	}
	oauthRedirect(c, user)
}
