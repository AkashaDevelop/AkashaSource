package controller

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
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
	skipped := 0

	// 🌸 以「模型广场(ModelMeta)」为准～只给广场里收录的模型更新上游参考价。
	// 广场里没有的模型即使 models.dev 有价、ModelConfig 里也有记录，也一律跳过，
	// 免得把没在用的模型也顺手改了喵(๑•̀ㅂ•́)و
	metaNameSet := loadModelMetaNameSet()

	// 只更新「既在模型广场、又在 ModelConfig」的模型，且只写两个参考价字段～
	for modelName, prices := range modelPrices {
		if !metaNameSet[modelName] {
			// 不在模型广场 → 跳过
			skipped++
			continue
		}
		var existing model.ModelConfig
		err := common.DB.Where("model_name = ?", modelName).First(&existing).Error
		if err != nil {
			// 广场里有、但还没建定价记录 → 也跳过，交给"未匹配定价"筛选让超管手动配
			skipped++
			continue
		}
		if err := common.DB.Model(&model.ModelConfig{}).
			Where("id = ?", existing.Id).
			Updates(map[string]interface{}{
				"upstream_input_price":  prices.input,
				"upstream_output_price": prices.output,
			}).Error; err == nil {
			updated++
		}
	}

	common.OK(c, gin.H{
		"created": created, // 恒为 0，保留字段兼容前端提示～
		"updated": updated,
		"skipped": skipped,
		"total":   len(modelPrices),
	})
}

// 🌸 loadModelMetaNameSet ～把模型广场(ModelMeta)里所有模型名捞成一个集合，
// 给"以广场为准"的同步逻辑当白名单用～查库失败就返回空集(宁可少同步也不乱同步)喵
func loadModelMetaNameSet() map[string]bool {
	set := make(map[string]bool)
	var names []string
	if err := common.DB.Model(&model.ModelMeta{}).Pluck("model_name", &names).Error; err != nil {
		return set
	}
	for _, n := range names {
		trimmed := strings.TrimSpace(n)
		if trimmed != "" {
			set[trimmed] = true
		}
	}
	return set
}
