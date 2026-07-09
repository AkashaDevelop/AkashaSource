package suno

import (
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
	"STfreApi/service/realname"
	"STfreApi/service/xuanjian"
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

	// 实名认证检查（模型调用场景）
	if err := realname.CheckRealnameRequirement(token.UserId, model.RealnameScenarioModelCall); err != nil {
		c.JSON(http.StatusForbidden, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: err.Error(),
			Type:    "invalid_request_error",
			Code:    "realname_required",
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

	// 宸汐玄鉴 AI 审核：提取 prompt/title/tags 等文本字段送审
	if xuanjian.IsEnabled() {
		var sunoReq struct {
			Prompt string `json:"prompt"`
			Title  string `json:"title"`
			Tags   string `json:"tags"`
		}
		if json.Unmarshal(bodyBytes, &sunoReq) == nil {
			var reviewText string
			if sunoReq.Prompt != "" {
				reviewText += sunoReq.Prompt + " "
			}
			if sunoReq.Title != "" {
				reviewText += sunoReq.Title + " "
			}
			if sunoReq.Tags != "" {
				reviewText += sunoReq.Tags
			}
			if reviewText != "" {
				reviewMessages := []interface{}{
					map[string]interface{}{"role": "user", "content": reviewText},
				}
				allowed, blockMsg := xuanjian.AIReviewCheck(token.Id, token.UserId, reviewMessages, "")
				if !allowed {
					c.JSON(http.StatusBadRequest, gin.H{"error": blockMsg})
					return
				}
			}
		}
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

	// 宸汐玄鉴：把 Suno 生成请求也纳入行为画像，补上此前的画像盲区～
	// Suno 是 token-key 认证，有真实 token.Id，和文本 relay 一致处理。
	if xuanjian.IsEnabled() {
		var sunoReq struct {
			Prompt string `json:"prompt"`
			Title  string `json:"title"`
			Tags   string `json:"tags"`
		}
		_ = json.Unmarshal(bodyBytes, &sunoReq)
		snippet := strings.TrimSpace(strings.Join([]string{sunoReq.Prompt, sunoReq.Title, sunoReq.Tags}, " "))
		go xuanjian.RecordRequest(xuanjian.RequestRecord{
			TokenID:        token.Id,
			TokenName:      token.Name,
			UserID:         token.UserId,
			TokenCreatedAt: time.Unix(token.CreatedTime, 0),
			IP:             c.ClientIP(),
			UserAgent:      c.GetHeader("User-Agent"),
			Model:          "suno_" + path,
			PromptSnippet:  xuanjian.TruncateSnippet(snippet, 2000),
			StatusCode:     resp.StatusCode,
		})
	}

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

// RelaySunoFetch proxies Suno task fetch endpoints:
// - POST /suno/fetch
// - GET  /suno/fetch/:id
func RelaySunoFetch(c *gin.Context) {
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

	var channel model.Channel
	if err := common.DB.Where("type = ? AND status = ?",
		model.ChannelTypeSuno, model.ChannelStatusActive).
		Order("priority DESC, RANDOM()").First(&channel).Error; err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no suno channel available"})
		return
	}

	baseURL := strings.TrimSuffix(channel.BaseURL, "/")
	targetURL := fmt.Sprintf("%s/suno/fetch", baseURL)
	method := c.Request.Method

	if method == http.MethodGet {
		taskID := strings.TrimSpace(c.Param("id"))
		if taskID != "" {
			targetURL = fmt.Sprintf("%s/suno/fetch/%s", baseURL, taskID)
		}
		if raw := c.Request.URL.RawQuery; raw != "" {
			targetURL += "?" + raw
		}

		req, err := http.NewRequest(http.MethodGet, targetURL, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
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
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}
	req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewBuffer(bodyBytes))
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
