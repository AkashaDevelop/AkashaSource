package controller

import (
	"encoding/json"
	"errors"
	"math/rand"
	"sort"
	"strings"
	"time"

	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service"
)

// SelectChannel 根据模型选择合适的渠道（基于 Ability 表：group×model×channel 索引，不再整表扫描 Channel）～
// userGroup：这个请求所属用户自己的分组，用来算 auto 轮询时"我到底能碰哪些分组"～
// usingGroup：这次请求实际要用的分组（可能被令牌的独立分组覆盖，也可能是特殊值 "auto"）～
// crossGroupRetry：仅在 usingGroup=="auto" 时生效，决定候选分组用完重试要不要接着换下一个分组～
// 返回值第三项 channelGroups 与 channels 一一对应，记着每个候选渠道究竟是从哪个分组挑出来的～
// auto 分组下不同候选分组的渠道混在一起返回，事后计费/限流必须按这份"实际命中分组"算账，
// 不能笼统按 usingGroup（字面上就是"auto"）或 userGroup 打马虎眼，否则分组倍率不一致时会算错账～
func SelectChannel(modelName string, userGroup string, usingGroup string, crossGroupRetry bool) ([]*model.Channel, []string, []string, error) {
	groupsToTry := []string{usingGroup}
	if usingGroup == "auto" {
		autoGroups := service.GetUserAutoGroup(userGroup)
		if len(autoGroups) == 0 {
			return nil, nil, nil, errors.New("auto 分组未启用或没有可用的候选分组")
		}
		groupsToTry = autoGroups
	}

	var finalChannels []*model.Channel
	var finalModels []string
	var finalGroups []string

	for _, g := range groupsToTry {
		channels, models, err := selectChannelForSingleGroup(modelName, g)
		if err != nil {
			return nil, nil, nil, err
		}
		if len(channels) == 0 {
			continue
		}
		finalChannels = append(finalChannels, channels...)
		finalModels = append(finalModels, models...)
		for range channels {
			finalGroups = append(finalGroups, g)
		}
		if usingGroup != "auto" || !crossGroupRetry {
			// 非 auto 分组本来就只有一组；auto 分组但不许跨组重试时，凑够第一个有货的组就收手～
			break
		}
	}

	if len(finalChannels) == 0 {
		return nil, nil, nil, errors.New("no available channel for this model")
	}

	return finalChannels, finalModels, finalGroups, nil
}

// selectChannelForSingleGroup 在单个分组内，按 Ability 表选出全部候选渠道，
// 按优先级降序分桶、同优先级内加权洗牌，返回一条已经排好队的完整渠道列表～
func selectChannelForSingleGroup(modelName string, group string) ([]*model.Channel, []string, error) {
	abilities, err := model.GetChannelCandidates(group, modelName)
	if err != nil {
		return nil, nil, err
	}
	if len(abilities) == 0 {
		return nil, nil, nil
	}

	allowedChannels := service.GetGroupAllowedChannels(group)

	channelIds := make([]int, 0, len(abilities))
	abilityByChannel := make(map[int]model.Ability, len(abilities))
	for _, a := range abilities {
		if allowedChannels != nil {
			if _, ok := allowedChannels[a.ChannelId]; !ok {
				continue
			}
		}
		channelIds = append(channelIds, a.ChannelId)
		abilityByChannel[a.ChannelId] = a
	}
	if len(channelIds) == 0 {
		return nil, nil, nil
	}

	var channels []model.Channel
	if err := common.DB.Where("id IN ? AND status = ?", channelIds, model.ChannelStatusActive).Find(&channels).Error; err != nil {
		return nil, nil, err
	}
	if len(channels) == 0 {
		return nil, nil, nil
	}

	candidates := make([]Candidate, 0, len(channels))
	for _, ch := range channels {
		ability := abilityByChannel[ch.Id]
		mappedModel := modelName
		if strings.TrimSpace(ch.ModelMapping) != "" {
			var mapping map[string]string
			if err := json.Unmarshal([]byte(ch.ModelMapping), &mapping); err == nil {
				if target, ok := mapping[modelName]; ok {
					mappedModel = target
				}
			}
		}
		// Ability 里存的 priority/weight 才是分组维度生效的那份快照，渠道本体字段只是原始值～
		chCopy := ch
		chCopy.Priority = ability.Priority
		chCopy.Weight = ability.Weight
		candidates = append(candidates, Candidate{Channel: chCopy, MappedModel: mappedModel})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Channel.Priority > candidates[j].Channel.Priority
	})

	var finalChannels []*model.Channel
	var finalModels []string
	bucket := make([]Candidate, 0, len(candidates))
	flush := func() {
		if len(bucket) == 0 {
			return
		}
		shuffled := weightedShuffle(bucket)
		for _, sc := range shuffled {
			ch := sc.Channel
			finalChannels = append(finalChannels, &ch)
			finalModels = append(finalModels, sc.MappedModel)
		}
		bucket = bucket[:0]
	}

	currentPriority := candidates[0].Channel.Priority
	for _, c := range candidates {
		if c.Channel.Priority != currentPriority {
			flush()
			currentPriority = c.Channel.Priority
		}
		bucket = append(bucket, c)
	}
	flush()

	return finalChannels, finalModels, nil
}

// SelectChannelWithAffinity 在常规选路结果上，优先命中渠道亲和缓存。
func SelectChannelWithAffinity(modelName string, userGroup string, usingGroup string, crossGroupRetry bool, tokenKey string, ruleName string) ([]*model.Channel, []string, []string, error) {
	channels, mappedModels, channelGroups, err := SelectChannel(modelName, userGroup, usingGroup, crossGroupRetry)
	if err != nil {
		return nil, nil, nil, err
	}

	ruleName = strings.TrimSpace(ruleName)
	if ruleName == "" {
		ruleName = defaultChannelAffinityRule
	}
	keyFP := getChannelAffinityKeyFP(tokenKey)
	if keyFP == "" {
		return channels, mappedModels, channelGroups, nil
	}

	channelID, ok := getChannelAffinityChannelID(ruleName, usingGroup, keyFP)
	if !ok {
		return channels, mappedModels, channelGroups, nil
	}
	newChannels, newModels, newGroups := prioritizeAffinityChannel(channels, mappedModels, channelGroups, channelID)
	return newChannels, newModels, newGroups, nil
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
