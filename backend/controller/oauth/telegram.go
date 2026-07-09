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
	"strconv"
	"strings"
	"time"

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

	// ～防重放攻击：验证 auth_date 时间戳，超过 24 小时的登录链接拒绝喵～
	authDateStr := c.Query("auth_date")
	if authDateStr != "" {
		authDate, err := strconv.ParseInt(authDateStr, 10, 64)
		if err == nil {
			age := time.Now().Unix() - authDate
			if age > 86400 || age < -300 { // 24小时过期，允许 5 分钟时钟偏移
				c.JSON(http.StatusOK, gin.H{"success": false, "message": "Login link expired or invalid timestamp"})
				return
			}
		}
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
