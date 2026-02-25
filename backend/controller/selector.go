package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"encoding/json"
	"errors"
	"math/rand"
	"strings"
	"time"
)

// SelectChannel 根据模型选择合适的渠道
// 支持负载均衡（权重）和优先级
// 返回选中的渠道和重定向后的模型名称
func SelectChannel(modelName string, userGroup string) ([]*model.Channel, []string, error) {
	var channels []model.Channel

	// 1. 查找所有启用且包含该模型的渠道
	db := common.DB.Where("status = ?", model.ChannelStatusActive)
	if userGroup != "" {
		db = db.Where("`group` IN ?", []string{userGroup, "default", ""})
	} else {
		db = db.Where("`group` IN ?", []string{"default", ""})
	}

	if err := db.Order("priority desc").Find(&channels).Error; err != nil {
		return nil, nil, err
	}

	// 2. 过滤支持该模型的渠道，并处理模型映射
	var candidates []Candidate

	for _, channel := range channels {
		// 解析模型列表
		models := strings.Split(channel.Models, ",")
		support := false
		mappedModel := modelName

		// 检查是否在支持列表中
		for _, m := range models {
			if m == modelName {
				support = true
				break
			}
		}

		// 检查模型映射 (ModelMapping)
		if channel.ModelMapping != "" {
			var mapping map[string]string
			if err := json.Unmarshal([]byte(channel.ModelMapping), &mapping); err == nil {
				if target, ok := mapping[modelName]; ok {
					mappedModel = target
					support = true // 如果有映射，也视为支持
				}
			}
		}

		if support {
			candidates = append(candidates, Candidate{
				Channel:     channel,
				MappedModel: mappedModel,
			})
		}
	}

	if len(candidates) == 0 {
		return nil, nil, errors.New("no available channel for this model")
	}

	// 3. 按照优先级分组
	var finalChannels []*model.Channel
	var finalModels []string

	// Since candidates are already sorted by priority (desc) from DB query
	// We just need to shuffle/balance within same priority groups

	currentPriority := -999999
	var currentGroup []Candidate = make([]Candidate, 0)

	for _, c := range candidates {
		if c.Channel.Priority != currentPriority {
			// Process previous group
			if len(currentGroup) > 0 {
				shuffled := weightedShuffle(currentGroup)
				for _, sc := range shuffled {
					ch := sc.Channel
					finalChannels = append(finalChannels, &ch)
					finalModels = append(finalModels, sc.MappedModel)
				}
				currentGroup = make([]Candidate, 0)
			}
			currentPriority = c.Channel.Priority
		}
		currentGroup = append(currentGroup, c)
	}

	// Process last group
	if len(currentGroup) > 0 {
		shuffled := weightedShuffle(currentGroup)
		for _, sc := range shuffled {
			ch := sc.Channel
			finalChannels = append(finalChannels, &ch)
			finalModels = append(finalModels, sc.MappedModel)
		}
	}

	return finalChannels, finalModels, nil
}

type Candidate struct {
	Channel     model.Channel
	MappedModel string
}

func weightedShuffle(candidates []Candidate) []Candidate {
	if len(candidates) <= 1 {
		return candidates
	}

	var result []Candidate
	temp := make([]Candidate, len(candidates))
	copy(temp, candidates)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for len(temp) > 0 {
		totalWeight := 0
		for _, c := range temp {
			totalWeight += c.Channel.Weight
		}

		if totalWeight <= 0 {
			// If all weights are 0, just append rest
			result = append(result, temp...)
			break
		}

		target := r.Intn(totalWeight)
		current := 0
		for i, c := range temp {
			current += c.Channel.Weight
			if target < current {
				result = append(result, c)
				// Remove i
				temp = append(temp[:i], temp[i+1:]...)
				break
			}
		}
	}

	return result
}
