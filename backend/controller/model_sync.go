package controller

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"sort"
	"time"

	"STfreApi/common"
	"STfreApi/model"

	"github.com/gin-gonic/gin"
)

const modelsDevURL = "https://models.dev/api.json"

type modelsDevProvider struct {
	Models map[string]modelsDevModel `json:"models"`
}

type modelsDevModel struct {
	Cost modelsDevCost `json:"cost"`
}

type modelsDevCost struct {
	Input  *float64 `json:"input"`
	Output *float64 `json:"output"`
}

func SyncUpstreamPricing(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", modelsDevURL, nil)
	if err != nil {
		common.Fail(c, common.CodeServerError, "创建请求失败")
		return
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		common.Fail(c, common.CodeServerError, "获取上游数据失败: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		common.Fail(c, common.CodeServerError, "上游返回错误状态")
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		common.Fail(c, common.CodeServerError, "读取响应失败")
		return
	}

	var upstreamData map[string]modelsDevProvider
	if err := json.Unmarshal(body, &upstreamData); err != nil {
		common.Fail(c, common.CodeServerError, "解析数据失败")
		return
	}

	if len(upstreamData) == 0 {
		common.Fail(c, common.CodeServerError, "上游数据为空")
		return
	}

	// 选择每个模型的最低价格
	modelPrices := make(map[string]struct {
		input  float64
		output float64
	})

	providers := make([]string, 0, len(upstreamData))
	for provider := range upstreamData {
		providers = append(providers, provider)
	}
	sort.Strings(providers)

	for _, provider := range providers {
		providerData := upstreamData[provider]
		for modelName, modelData := range providerData.Models {
			if modelData.Cost.Input == nil {
				continue
			}
			input := *modelData.Cost.Input
			if math.IsNaN(input) || math.IsInf(input, 0) || input < 0 {
				continue
			}

			var output float64
			if modelData.Cost.Output != nil {
				output = *modelData.Cost.Output
				if math.IsNaN(output) || math.IsInf(output, 0) || output < 0 {
					continue
				}
			}

			existing, exists := modelPrices[modelName]
			if !exists || (input > 0 && input < existing.input) {
				modelPrices[modelName] = struct {
					input  float64
					output float64
				}{input: input, output: output}
			}
		}
	}

	created := 0
	updated := 0

	for modelName, prices := range modelPrices {
		var existing model.ModelConfig
		err := common.DB.Where("model_name = ?", modelName).First(&existing).Error

		if err != nil {
			cfg := model.ModelConfig{
				ModelName:           modelName,
				DisplayName:         modelName,
				Category:            "chat",
				InputRatio:          1,
				OutputRatio:         1,
				UpstreamInputPrice:  prices.input,
				UpstreamOutputPrice: prices.output,
				MaxContext:          4096,
				Enabled:             true,
				CreatedAt:           time.Now().Unix(),
			}
			if err := common.DB.Create(&cfg).Error; err == nil {
				created++
			}
		} else {
			existing.UpstreamInputPrice = prices.input
			existing.UpstreamOutputPrice = prices.output
			if err := common.DB.Save(&existing).Error; err == nil {
				updated++
			}
		}
	}

	common.OK(c, gin.H{
		"created": created,
		"updated": updated,
		"total":   len(modelPrices),
	})
}
