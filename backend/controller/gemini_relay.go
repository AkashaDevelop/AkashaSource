package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RelayGeminiNative proxies requests in Gemini's native REST format
func RelayGeminiNative(c *gin.Context) {
	// 1. Auth via query param or header
	apiKey := c.Query("key")
	if apiKey == "" {
		authHeader := c.GetHeader("Authorization")
		apiKey = strings.TrimPrefix(authHeader, "Bearer ")
	}
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing api key"})
		return
	}

	token, err := GetTokenByKey(apiKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
		return
	}
	if err := ValidateToken(token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 2. Extract model from URL path
	modelName := c.Param("model")
	if modelName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model required"})
		return
	}

	// 3. Get user
	var user model.User
	if err := common.DB.Where("id = ?", token.UserId).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	// 4. Select Gemini channel
	channels, _, err := SelectChannel(modelName, user.Group)
	if err != nil || len(channels) == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": fmt.Sprintf("no channel for model: %s", modelName),
		})
		return
	}

	// Find a Gemini-type channel
	var channel *model.Channel
	for i := range channels {
		if channels[i].Type == model.ChannelTypeGemini {
			channel = channels[i]
			break
		}
	}
	if channel == nil {
		channel = channels[0]
	}
	channel.Key = service.GetNextKey(channel.Key)

	// 5. Read body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// 6. Determine action from URL
	action := c.Param("action")
	if action == "" {
		action = "generateContent"
	}

	// 7. Forward to upstream Gemini
	baseURL := channel.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	targetURL := fmt.Sprintf("%s/v1beta/models/%s:%s?key=%s",
		baseURL, modelName, action, channel.Key)

	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	common.ApplyHeaders(req, channel.Headers)

	client := common.NewHTTPClient(channel.Proxy)
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	// 8. Stream response back
	respBody, _ := io.ReadAll(resp.Body)

	for k, v := range resp.Header {
		c.Writer.Header().Set(k, v[0])
	}
	c.Writer.WriteHeader(resp.StatusCode)
	c.Writer.Write(respBody)

	// 9. Estimate usage for billing
	if resp.StatusCode == http.StatusOK {
		promptTokens := common.CountToken(string(bodyBytes))
		completionTokens := common.CountToken(string(respBody))
		go RecordConsumeLog(c, token, modelName,
			promptTokens, completionTokens)
	}
}
