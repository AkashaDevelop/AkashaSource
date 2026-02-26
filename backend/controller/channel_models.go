package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service"

	"github.com/gin-gonic/gin"
)

// FetchChannelModels fetches models from an OpenAI-compatible channel
func FetchChannelModels(c *gin.Context) {
	id := c.Param("id")
	var channel model.Channel
	if err := common.DB.First(&channel, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "渠道不存在"})
		return
	}

	baseUrl := channel.BaseURL
	if baseUrl == "" {
		baseUrl = "https://api.openai.com"
	}
	baseUrl = strings.TrimSuffix(baseUrl, "/")
	targetURL := fmt.Sprintf("%s/v1/models", baseUrl)

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建请求失败"})
		return
	}

	key := service.GetNextKey(channel.Key)
	if key != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
	}

	client := common.NewHTTPClient(channel.Proxy)
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "请求上游失败: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			Id string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "解析响应失败"})
		return
	}

	var models []string
	for _, m := range result.Data {
		models = append(models, m.Id)
	}

	c.JSON(http.StatusOK, gin.H{"data": models})
}

// TestAllChannels tests all active channels concurrently
func TestAllChannels(c *gin.Context) {
	var channels []model.Channel
	common.DB.Where("status = ?", model.ChannelStatusActive).Find(&channels)

	type TestResult struct {
		Id           int    `json:"id"`
		Name         string `json:"name"`
		Success      bool   `json:"success"`
		ResponseTime int    `json:"response_time"`
		Error        string `json:"error,omitempty"`
	}

	results := make([]TestResult, len(channels))
	var wg sync.WaitGroup

	for i, ch := range channels {
		wg.Add(1)
		go func(idx int, channel model.Channel) {
			defer wg.Done()
			rt, err := service.CheckChannel(&channel)
			r := TestResult{
				Id:   channel.Id,
				Name: channel.Name,
			}
			if err != nil {
				r.Success = false
				r.Error = err.Error()
			} else {
				r.Success = true
				r.ResponseTime = rt
				// Update channel test info
				common.DB.Model(&channel).Updates(map[string]interface{}{
					"response_time": rt,
					"test_time":     common.GetTimestamp(),
				})
			}
			results[idx] = r
		}(i, ch)
	}

	wg.Wait()
	c.JSON(http.StatusOK, gin.H{"data": results})
}
