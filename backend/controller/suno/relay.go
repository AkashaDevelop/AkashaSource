package suno

import (
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// RelaySuno proxies requests to an external suno-api service
func RelaySuno(c *gin.Context) {
	// Auth
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "missing api key",
			Type:    "invalid_request_error",
		}})
		return
	}
	tokenKey := strings.TrimPrefix(authHeader, "Bearer ")

	var token model.Token
	if err := common.DB.Where("key = ?", tokenKey).First(&token).Error; err != nil {
		c.JSON(http.StatusUnauthorized, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "invalid api key",
			Type:    "invalid_request_error",
		}})
		return
	}

	// Find a Suno channel
	var channel model.Channel
	if err := common.DB.Where("type = ? AND status = ?",
		model.ChannelTypeSuno, model.ChannelStatusActive).
		Order("priority DESC, RANDOM()").First(&channel).Error; err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no suno channel available"})
		return
	}

	// Read request body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// Forward to suno-api service
	baseURL := strings.TrimSuffix(channel.BaseURL, "/")
	path := c.Param("action")
	targetURL := fmt.Sprintf("%s/suno/submit/%s", baseURL, path)

	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if channel.Key != "" {
		req.Header.Set("Authorization", "Bearer "+channel.Key)
	}

	client := common.NewHTTPClient(channel.Proxy)
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// Record task
	var result map[string]interface{}
	if json.Unmarshal(respBody, &result) == nil {
		if taskId, ok := result["task_id"].(string); ok && taskId != "" {
			task := model.SunoTask{
				TaskId:    taskId,
				UserId:    token.UserId,
				Action:    path,
				Status:    "pending",
				Input:     string(bodyBytes),
				CreatedAt: time.Now().Unix(),
			}
			common.DB.Create(&task)
		}
	}

	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// SunoNotify handles callback notifications from suno-api
func SunoNotify(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	var notification struct {
		TaskId string `json:"task_id"`
		Status string `json:"status"`
		Result string `json:"result"`
	}
	if err := json.Unmarshal(bodyBytes, &notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	if notification.TaskId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_id required"})
		return
	}

	// Update task
	updates := map[string]interface{}{
		"status":     notification.Status,
		"result":     notification.Result,
		"updated_at": time.Now().Unix(),
	}
	common.DB.Model(&model.SunoTask{}).
		Where("task_id = ?", notification.TaskId).
		Updates(updates)

	// Deduct quota on completion
	if notification.Status == "completed" {
		var task model.SunoTask
		if common.DB.Where("task_id = ?", notification.TaskId).First(&task).Error == nil {
			quota := int64(common.QuotaSunoGenerate)
			common.DB.Model(&model.User{}).Where("id = ?", task.UserId).
				Update("quota", common.DB.Raw("quota - ?", quota))
			common.DB.Model(&model.SunoTask{}).
				Where("task_id = ?", notification.TaskId).
				Update("quota", quota)
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
