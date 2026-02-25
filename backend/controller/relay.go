package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"STfreApi/adapter"
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
	"STfreApi/service"

	"github.com/gin-gonic/gin"
)

// Relay 处理核心转发逻辑
func Relay(c *gin.Context) {
	// 1. 获取 Authorization Token
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "You didn't provide an API key. You need to provide your API key in an Authorization header using Bearer auth (i.e. Authorization: Bearer YOUR_KEY).",
			Type:    "invalid_request_error",
			Code:    "invalid_api_key",
		}})
		return
	}
	tokenKey := strings.TrimPrefix(authHeader, "Bearer ")

	// 2. 验证 Token
	var token model.Token
	if err := common.DB.Where("key = ?", tokenKey).First(&token).Error; err != nil {
		c.JSON(http.StatusUnauthorized, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "Incorrect API key provided.",
			Type:    "invalid_request_error",
			Code:    "invalid_api_key",
		}})
		return
	}

	if token.Status != model.TokenStatusActive {
		c.JSON(http.StatusUnauthorized, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "Your API key is disabled.",
			Type:    "invalid_request_error",
		}})
		return
	}

	// 3. 解析请求体以获取 Model
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "Invalid request body.",
			Type:    "invalid_request_error",
		}})
		return
	}

	var openAIReq dto.OpenAIRequest
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &openAIReq); err != nil {
			c.JSON(http.StatusBadRequest, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
				Message: "Invalid JSON body.",
				Type:    "invalid_request_error",
			}})
			return
		}
	} else {
		// Handle empty body (CLI support)
		if c.Request.Method == "GET" {
			c.JSON(http.StatusOK, gin.H{
				"message": "Akasha is running! Please use POST to /v1/chat/completions",
			})
			return
		}
	}

	// Default model if not provided (CLI support)
	if openAIReq.Model == "" {
		openAIReq.Model = common.DefaultModel
	}

	var user model.User
	if err := common.DB.Where("id = ?", token.UserId).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "Internal server error.",
			Type:    "server_error",
		}})
		return
	}

	// 4. 选择渠道 (Get list of channels for retry)
	channels, mappedModels, err := SelectChannel(openAIReq.Model, user.Group)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: fmt.Sprintf("No available channel for model: %s", openAIReq.Model),
			Type:    "server_error",
		}})
		return
	}

	// Retry Loop
	var lastError error
	for i, channel := range channels {
		mappedModel := mappedModels[i]

		// Update request model for this attempt
		openAIReq.Model = mappedModel

		// Get Adaptor
		adaptor := adapter.GetAdaptor(channel.Type)

		// Convert Request
		convertedReq, err := adaptor.ConvertRequest(c, &openAIReq)
		if err != nil {
			lastError = err
			continue
		}

		// Do Request
		resp, err := adaptor.DoRequest(c, channel, convertedReq)
		if err != nil {
			lastError = err
			continue
		}

		// Error Handling & Auto-Ban
		if resp.StatusCode >= 400 {
			// Read body to check error details
			responseBody, err := io.ReadAll(resp.Body)
			resp.Body.Close()

			if err != nil {
				lastError = fmt.Errorf("channel %s returned status %d and failed to read body", channel.Name, resp.StatusCode)
				continue
			}

			var errorResponse dto.OpenAIErrorResponse
			// Try to unmarshal, but if it fails, we still have the status code
			json.Unmarshal(responseBody, &errorResponse)
			
			// Check if we should disable this channel
			if service.ShouldDisableChannel(&errorResponse.Error, resp.StatusCode) {
				// Use goroutine to avoid blocking
				go service.DisableChannel(channel.Id, fmt.Sprintf("%d - %s", resp.StatusCode, errorResponse.Error.Message))
			}

			// Determine if we should retry
			// Retry on:
			// 1. 429 (Too Many Requests)
			// 2. 5xx (Server Errors)
			// 3. 401 (Unauthorized - Invalid Key) -> We disabled it, so retry next
			// 4. 403 (Forbidden - Account issue) -> We disabled it, so retry next
			shouldRetry := resp.StatusCode == 429 || resp.StatusCode >= 500 || service.ShouldDisableChannel(&errorResponse.Error, resp.StatusCode)

			if shouldRetry {
				lastError = fmt.Errorf("channel %s returned error: %d %s", channel.Name, resp.StatusCode, errorResponse.Error.Message)
				continue
			}

			// If not retrying (e.g. 400 Bad Request), return error to client
			c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), responseBody)
			return
		}

		// Success (2xx)
		defer resp.Body.Close()

		// Do Response & Calculate Usage
		usage, err := adaptor.DoResponse(c, resp, &token)
		if err != nil {
			// Response handling error (e.g. JSON parse error from provider)
			// Should we retry here? Maybe if it's a provider issue.
			// But content is already written partially?
			// DoResponse handles writing to c.Writer.
			// If stream, it writes chunks.
			// If error happens during stream, client gets partial data.
			// We can't really retry stream easily once started.
			// So we just log/return.
			// But for non-stream, DoResponse writes at the end?
			// Adaptor implementation specific.
			// Assuming DoResponse handles response fully.
			// But we need to record log.
		}

		// Record Log
		promptTokens := 0
		completionTokens := 0

		// Handle Image Generation Quota
		if strings.HasPrefix(openAIReq.Model, "dall-e") {
			// For images, we can set a fixed "token" cost or handle it in RecordConsumeLog
			// DALL-E 3: ~20000 quota ($0.04)
			// DALL-E 2: ~10000 quota ($0.02)
			// Let's use a convention: PromptTokens = Quota Cost
			if strings.HasPrefix(openAIReq.Model, "dall-e-3") {
				promptTokens = common.QuotaDalle3
			} else {
				promptTokens = common.QuotaDalle2
			}
			completionTokens = 0
		} else if usage != nil {
			promptTokens = usage.PromptTokens
			completionTokens = usage.CompletionTokens
		} else {
			// Fallback: estimate from request/response if adaptor didn't return usage
			// Simple estimation for now
			if openAIReq.Input != nil {
				promptTokens = common.CountToken(fmt.Sprint(openAIReq.Input))
			} else {
				promptTokens = common.CountToken(fmt.Sprint(openAIReq.Messages))
			}
			// Completion tokens? Unknown for stream if not tracked.
		}

		go RecordConsumeLog(c, &token, openAIReq.Model, promptTokens, completionTokens)

		return // Exit on success
	}

	// All attempts failed
	c.JSON(http.StatusServiceUnavailable, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
		Message: fmt.Sprintf("All channels failed. Last error: %v", lastError),
		Type:    "server_error",
	}})
}
