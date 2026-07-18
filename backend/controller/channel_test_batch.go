package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ✨ 批量测试渠道模型的萌萌控制器～
// 支持对单个渠道的多个模型进行并发测试，快速验证可用性！

// ModelTestResult 单个模型的测试结果
type ModelTestResult struct {
	Model        string `json:"model"`         // 模型名称
	Status       string `json:"status"`        // 测试状态: "未测试" | "测试中" | "成功" | "失败"
	ResponseTime int    `json:"response_time"` // 响应时间(ms)
	Error        string `json:"error"`         // 错误信息
}

// BatchTestChannelModels 批量测试渠道的多个模型
// POST /api/channel/:id/test-batch
// 请求体:
//
//	{
//	  "models": ["gpt-4", "gpt-3.5-turbo"],  // 可选，不传则测试渠道配置的所有模型
//	  "prompt": "你好"                        // 可选，测试问题
//	}
func BatchTestChannelModels(c *gin.Context) {
	id := c.Param("id")
	var channel model.Channel
	if err := common.DB.First(&channel, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}

	var req struct {
		Models []string `json:"models"` // 要测试的模型列表
		Prompt string   `json:"prompt"` // 测试问题
	}
	_ = c.ShouldBindJSON(&req)

	// 如果没有指定模型，使用渠道配置的所有模型
	testModels := req.Models
	if len(testModels) == 0 {
		modelStr := strings.TrimSpace(channel.Models)
		if modelStr == "" {
			common.Fail(c, common.CodeParamError, "渠道未配置模型")
			return
		}
		for _, m := range strings.Split(modelStr, ",") {
			m = strings.TrimSpace(m)
			if m != "" {
				testModels = append(testModels, m)
			}
		}
	}

	if len(testModels) == 0 {
		common.Fail(c, common.CodeParamError, "没有可测试的模型")
		return
	}

	// 并发测试所有模型
	results := make([]ModelTestResult, len(testModels))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, modelName := range testModels {
		wg.Add(1)
		go func(idx int, model string) {
			defer wg.Done()

			result := ModelTestResult{
				Model:  model,
				Status: "测试中",
			}

			// 执行测试
			startTime := time.Now()
			responseTime, err := service.CheckChannelWithPrompt(&channel, model, req.Prompt)
			elapsed := int(time.Since(startTime).Milliseconds())

			if err != nil {
				result.Status = "失败"
				result.Error = err.Error()
				result.ResponseTime = elapsed
			} else {
				result.Status = "成功"
				result.ResponseTime = responseTime
			}

			mu.Lock()
			results[idx] = result
			mu.Unlock()
		}(i, modelName)
	}

	wg.Wait()

	// 统计结果
	successCount := 0
	failedCount := 0
	for _, r := range results {
		if r.Status == "成功" {
			successCount++
		} else if r.Status == "失败" {
			failedCount++
		}
	}

	common.OK(c, gin.H{
		"results":       results,
		"total":         len(results),
		"success_count": successCount,
		"failed_count":  failedCount,
	})
}

// TestChannelWithEndpointType 测试渠道连接（支持端点类型检测）
// POST /api/channel/:id/test-advanced
// 请求体:
//
//	{
//	  "endpoint_type": "auto",  // 端点类型: "auto" | "openai" | "azure" | "claude"
//	  "test_mode": false        // 是否启用测试模式
//	}
func TestChannelWithEndpointType(c *gin.Context) {
	id := c.Param("id")
	var channel model.Channel
	if err := common.DB.First(&channel, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}

	var req struct {
		EndpointType string `json:"endpoint_type"` // 端点类型
		TestMode     bool   `json:"test_mode"`     // 测试模式
	}
	_ = c.ShouldBindJSON(&req)

	// 目前先简单实现，后续可以根据endpoint_type做不同的检测逻辑
	responseTime, err := service.CheckChannel(&channel)
	if err != nil {
		common.OK(c, gin.H{
			"success":       false,
			"msg":           err.Error(),
			"time":          0,
			"endpoint_type": req.EndpointType,
		})
		return
	}

	// 更新渠道响应时间
	common.DB.Model(&channel).Updates(map[string]interface{}{
		"response_time": responseTime,
		"test_time":     common.GetTimestamp(),
	})

	common.OK(c, gin.H{
		"success":       true,
		"time":          responseTime,
		"msg":           "测试成功",
		"endpoint_type": req.EndpointType,
	})
}
