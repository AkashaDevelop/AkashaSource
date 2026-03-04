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

	// 2. Extract model and action from URL path (*path = "/gemini-pro:generateContent")
	rawPath := strings.TrimPrefix(c.Param("path"), "/")
	parts := strings.SplitN(rawPath, ":", 2)
	modelName := parts[0]
	if modelName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model required"})
		return
	}
	action := "generateContent"
	if len(parts) == 2 && parts[1] != "" {
		action = parts[1]
	}

	// 3. Get user
	var user model.User
	if err := common.DB.Where("id = ?", token.UserId).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	// 4. Select Gemini channel
	channels, mappedModels, err := SelectChannelWithAffinity(modelName, user.Group, apiKey, defaultChannelAffinityRule)
	if err != nil || len(channels) == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": fmt.Sprintf("no channel for model: %s", modelName),
		})
		return
	}

	// Find a Gemini-type channel
	selectedIndex := 0
	for i := range channels {
		if channels[i].Type == model.ChannelTypeGemini {
			selectedIndex = i
			break
		}
	}
	channel := channels[selectedIndex]
	mappedModel := modelName
	if selectedIndex < len(mappedModels) && strings.TrimSpace(mappedModels[selectedIndex]) != "" {
		mappedModel = mappedModels[selectedIndex]
	}
	channel.Key = service.GetNextKey(channel.Key)

	// 5. Read body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// 6. Forward to upstream Gemini
	baseURL := channel.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	targetURL := fmt.Sprintf("%s/v1beta/models/%s:%s?key=%s",
		baseURL, mappedModel, action, channel.Key)

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
		go RecordConsumeLog(c, token, mappedModel,
			promptTokens, completionTokens)
		go upsertChannelAffinity(defaultChannelAffinityRule, user.Group, getChannelAffinityKeyFP(apiKey), channel.Id, mappedModel, promptTokens, completionTokens, 0)
	}
}
