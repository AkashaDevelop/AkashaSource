package midjourney

import (
	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service/xuanjian"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// MJRequest represents the request body for Midjourney proxy
type MJRequest struct {
	Action     string `json:"action"` // IMAGINE, UPSCALE, VARIATION, etc.
	Prompt     string `json:"prompt"`
	TaskId     string `json:"taskId"`     // For upscale/variation
	Index      int    `json:"index"`      // For upscale/variation
	NotifyHook string `json:"notifyHook"` // Callback URL
}

// MJResponse represents the response from Midjourney proxy
type MJResponse struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	Result      string `json:"result"` // Task ID
}

func RelayMidjourney(c *gin.Context) {
	// 0. Get User
	userId := c.GetInt("id")
	var user model.User
	if err := common.DB.First(&user, userId).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// 1. Parse Request
	var mjReq MJRequest
	if err := c.ShouldBindJSON(&mjReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1.5 宸汐玄鉴 AI 审核
	if xuanjian.IsEnabled() && mjReq.Prompt != "" {
		reviewMessages := []interface{}{
			map[string]interface{}{"role": "user", "content": mjReq.Prompt},
		}
		allowed, blockMsg := xuanjian.AIReviewCheck(0, userId, reviewMessages, "")
		if !allowed {
			c.JSON(http.StatusBadRequest, gin.H{"error": blockMsg})
			return
		}
	}

	// Override notifyHook to our server
	// We need to know the public URL of our server to set notifyHook
	// For now, assume mj-proxy is configured to callback to us, or we pass it explicitly
	// If mj-proxy supports notifyHook in request, we set it to our /mj/notify endpoint
	// But we need to pass a unique ID to identify the user/task

	// Calculate Quota
	var quota int64
	switch strings.ToUpper(mjReq.Action) {
	case "IMAGINE":
		quota = int64(common.QuotaMJImagine)
	case "UPSCALE":
		quota = int64(common.QuotaMJUpscale)
	case "VARIATION":
		quota = int64(common.QuotaMJVariation)
	default:
		quota = int64(common.QuotaMJImagine)
	}

	if user.Quota < quota {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient quota"})
		return
	}

	// 2. Select Channel (mj-proxy supported)
	channel, err := model.GetRandomChannel("midjourney")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "No available Midjourney channel"})
		return
	}

	// 3. Construct Request to Proxy
	// mj-proxy usually has endpoints like /mj/submit/imagine, /mj/submit/change
	// We need to map our action to the correct endpoint
	var endpoint string
	switch strings.ToUpper(mjReq.Action) {
	case "IMAGINE":
		endpoint = "/mj/submit/imagine"
	case "UPSCALE", "VARIATION":
		endpoint = "/mj/submit/change" // Change includes both U and V
	case "DESCRIBE":
		endpoint = "/mj/submit/describe"
	case "BLEND":
		endpoint = "/mj/submit/blend"
	default:
		endpoint = "/mj/submit/imagine" // Default
	}

	targetURL := fmt.Sprintf("%s%s", channel.BaseURL, endpoint)

	reqBody, _ := json.Marshal(mjReq)
	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(reqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("mj-api-secret", channel.Key) // Common auth header for mj-proxy

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to request upstream"})
		return
	}
	defer resp.Body.Close()

	// 4. Handle Response
	body, _ := io.ReadAll(resp.Body)

	// Check if upstream returned success
	var mjResp MJResponse
	if err := json.Unmarshal(body, &mjResp); err == nil && mjResp.Code == 1 {
		// Task submitted successfully
		// Deduct quota immediately or wait for callback?
		// Usually deduct first, refund on failure

		user.Quota -= quota
		user.UsedQuota += quota
		user.RequestCount += 1
		common.DB.Save(&user)

		// Log
		log := model.Log{
			UserId:    userId,
			CreatedAt: common.GetTimestamp(),
			Type:      model.LogTypeConsume,
			Content:   fmt.Sprintf("Midjourney %s Task: %s", mjReq.Action, mjResp.Result),
			TokenName: "user_token", // Or actual token name
			ModelName: "midjourney",
			Quota:     quota,
			ChannelId: channel.Id,
		}
		common.DB.Create(&log)

		// Save Task for Callback
		go func() {
			task := model.MidjourneyTask{
				UserId:    userId,
				Code:      1,
				Progress:  "0%",
				Prompt:    mjReq.Prompt,
				PromptEn:  mjReq.Prompt,
				Action:    mjReq.Action,
				MjId:      mjResp.Result,
				Status:    "SUBMITTED",
				CreatedAt: common.GetTimestamp(),
				Quota:     quota,
			}
			common.DB.Create(&task)
		}()
	}

	c.Data(resp.StatusCode, "application/json", body)
}

// RelayMidjourneyFetch proxies task query endpoints like:
// - GET /mj/task/:id/fetch
// - GET /mj/task/:id/image-seed
func RelayMidjourneyFetch(c *gin.Context) {
	userId := c.GetInt("id")
	if userId <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	taskID := strings.TrimSpace(c.Param("id"))
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing task id"})
		return
	}

	channel, err := model.GetRandomChannel("midjourney")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "No available Midjourney channel"})
		return
	}

	endpoint := fmt.Sprintf("/mj/task/%s/fetch", taskID)
	if strings.Contains(c.FullPath(), "image-seed") {
		endpoint = fmt.Sprintf("/mj/task/%s/image-seed", taskID)
	}

	targetURL := fmt.Sprintf("%s%s", strings.TrimSuffix(channel.BaseURL, "/"), endpoint)
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	req.Header.Set("mj-api-secret", channel.Key)

	client := common.NewHTTPClient(channel.Proxy)
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to request upstream"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(resp.StatusCode, contentType, body)
}

// Notify callback from mj-proxy
func MidjourneyNotify(c *gin.Context) {
	var cb map[string]interface{}
	if err := c.ShouldBindJSON(&cb); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mjId, ok := cb["id"].(string)
	if !ok {
		// Try other fields depending on mj-proxy version
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	status, _ := cb["status"].(string)
	// progress, _ := cb["progress"].(string)
	// imageUrl, _ := cb["imageUrl"].(string)

	if status == "FAILURE" {
		// Refund
		var task model.MidjourneyTask
		if err := common.DB.Where("mj_id = ?", mjId).First(&task).Error; err == nil {
			if task.Status != "FAILURE" {
				task.Status = "FAILURE"
				task.FinishTime = common.GetTimestamp()
				common.DB.Save(&task)

				// Refund User
				var user model.User
				if err := common.DB.First(&user, task.UserId).Error; err == nil {
					user.Quota += task.Quota
					user.UsedQuota -= task.Quota
					common.DB.Save(&user)

					// Refund Log
					log := model.Log{
						UserId:    user.Id,
						CreatedAt: common.GetTimestamp(),
						Type:      model.LogTypeSystem,
						Content:   fmt.Sprintf("Refund MJ Task: %s", mjId),
						Quota:     task.Quota,
					}
					common.DB.Create(&log)
				}
			}
		}
	} else if status == "SUCCESS" {
		common.DB.Model(&model.MidjourneyTask{}).Where("mj_id = ?", mjId).Updates(map[string]interface{}{
			"status":      "SUCCESS",
			"finish_time": common.GetTimestamp(),
			"image_url":   cb["imageUrl"],
		})
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
