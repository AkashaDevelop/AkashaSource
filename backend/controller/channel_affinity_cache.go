package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const (
	channelAffinityNamespace      = "channel_affinity"
	defaultChannelAffinityRule    = "default"
	defaultChannelAffinityTTL     = 24 * time.Hour
	channelAffinityScanBatchCount = int64(200)
)

type channelAffinityEntry struct {
	RuleName         string `json:"rule_name"`
	UsingGroup       string `json:"using_group"`
	KeyFP            string `json:"key_fp"`
	ChannelID        int    `json:"channel_id"`
	ModelName        string `json:"model_name"`
	Hit              int64  `json:"hit"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	CachedTokens     int64  `json:"cached_tokens"`
	LastSeenAt       int64  `json:"last_seen_at"`
}

func sanitizeAffinityPart(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "_"
	}
	v = strings.ReplaceAll(v, ":", "_")
	return v
}

func buildChannelAffinityCacheKey(ruleName, usingGroup, keyFP string) string {
	ruleName = sanitizeAffinityPart(ruleName)
	usingGroup = sanitizeAffinityPart(usingGroup)
	keyFP = sanitizeAffinityPart(keyFP)
	return fmt.Sprintf("%s:%s:%s:%s", channelAffinityNamespace, ruleName, usingGroup, keyFP)
}

func parseChannelAffinityKey(key string) (ruleName, usingGroup, keyFP string, ok bool) {
	parts := strings.Split(key, ":")
	if len(parts) != 4 {
		return "", "", "", false
	}
	if parts[0] != channelAffinityNamespace {
		return "", "", "", false
	}
	return parts[1], parts[2], parts[3], true
}

func getChannelAffinityKeyFP(rawTokenKey string) string {
	rawTokenKey = strings.TrimSpace(rawTokenKey)
	if rawTokenKey == "" {
		return ""
	}
	h := sha256.Sum256([]byte(rawTokenKey))
	hexVal := hex.EncodeToString(h[:])
	if len(hexVal) > 16 {
		return hexVal[:16]
	}
	return hexVal
}

func getChannelAffinityChannelID(ruleName, usingGroup, keyFP string) (int, bool) {
	if common.RedisClient == nil {
		return 0, false
	}
	key := buildChannelAffinityCacheKey(ruleName, usingGroup, keyFP)
	var entry channelAffinityEntry
	if !common.GetCache(key, &entry) {
		return 0, false
	}
	if entry.ChannelID <= 0 {
		return 0, false
	}
	return entry.ChannelID, true
}

func upsertChannelAffinity(ruleName, usingGroup, keyFP string, channelID int, modelName string, promptTokens, completionTokens, cachedTokens int) {
	if common.RedisClient == nil || channelID <= 0 {
		return
	}
	key := buildChannelAffinityCacheKey(ruleName, usingGroup, keyFP)

	var entry channelAffinityEntry
	if !common.GetCache(key, &entry) {
		entry = channelAffinityEntry{
			RuleName:   sanitizeAffinityPart(ruleName),
			UsingGroup: sanitizeAffinityPart(usingGroup),
			KeyFP:      sanitizeAffinityPart(keyFP),
		}
	}

	entry.RuleName = sanitizeAffinityPart(ruleName)
	entry.UsingGroup = sanitizeAffinityPart(usingGroup)
	entry.KeyFP = sanitizeAffinityPart(keyFP)
	entry.ChannelID = channelID
	entry.ModelName = strings.TrimSpace(modelName)
	entry.Hit++
	entry.PromptTokens += int64(promptTokens)
	entry.CompletionTokens += int64(completionTokens)
	entry.CachedTokens += int64(cachedTokens)
	entry.LastSeenAt = time.Now().Unix()

	common.SetCache(key, entry, defaultChannelAffinityTTL)
}

func prioritizeAffinityChannel(channels []*model.Channel, mappedModels []string, channelID int) ([]*model.Channel, []string) {
	if channelID <= 0 || len(channels) <= 1 || len(channels) != len(mappedModels) {
		return channels, mappedModels
	}
	idx := -1
	for i, ch := range channels {
		if ch != nil && ch.Id == channelID {
			idx = i
			break
		}
	}
	if idx <= 0 {
		return channels, mappedModels
	}

	newChannels := make([]*model.Channel, 0, len(channels))
	newModels := make([]string, 0, len(mappedModels))

	newChannels = append(newChannels, channels[idx])
	newModels = append(newModels, mappedModels[idx])
	for i := range channels {
		if i == idx {
			continue
		}
		newChannels = append(newChannels, channels[i])
		newModels = append(newModels, mappedModels[i])
	}
	return newChannels, newModels
}

func clearChannelAffinityCacheAll() (int, error) {
	if common.RedisClient == nil {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cursor uint64
	deleted := 0
	for {
		keys, next, err := common.RedisClient.Scan(ctx, cursor, channelAffinityNamespace+":*", channelAffinityScanBatchCount).Result()
		if err != nil {
			return deleted, err
		}
		if len(keys) > 0 {
			n, err := common.RedisClient.Del(ctx, keys...).Result()
			if err != nil {
				return deleted, err
			}
			deleted += int(n)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return deleted, nil
}

func clearChannelAffinityCacheByRuleName(ruleName string) (int, error) {
	ruleName = sanitizeAffinityPart(ruleName)
	if ruleName == "_" {
		return 0, fmt.Errorf("rule_name 不能为空")
	}
	if common.RedisClient == nil {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pattern := fmt.Sprintf("%s:%s:*", channelAffinityNamespace, ruleName)
	var cursor uint64
	deleted := 0
	for {
		keys, next, err := common.RedisClient.Scan(ctx, cursor, pattern, channelAffinityScanBatchCount).Result()
		if err != nil {
			return deleted, err
		}
		if len(keys) > 0 {
			n, err := common.RedisClient.Del(ctx, keys...).Result()
			if err != nil {
				return deleted, err
			}
			deleted += int(n)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return deleted, nil
}

func getChannelAffinityCacheStats() (map[string]any, error) {
	if common.RedisClient == nil {
		return map[string]any{
			"enabled":   false,
			"mode":      "redis",
			"supported": false,
			"size":      0,
			"by_rule":   map[string]int{},
			"items":     []any{},
		}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	var cursor uint64
	byRule := make(map[string]int)
	items := make([]map[string]any, 0, 200)
	maxLastSeen := int64(0)
	total := 0

	for {
		keys, next, err := common.RedisClient.Scan(ctx, cursor, channelAffinityNamespace+":*", channelAffinityScanBatchCount).Result()
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			total++
			ruleName, usingGroup, keyFP, ok := parseChannelAffinityKey(key)
			if !ok {
				continue
			}
			byRule[ruleName]++
			if len(items) >= 200 {
				continue
			}
			var entry channelAffinityEntry
			if common.GetCache(key, &entry) {
				if entry.LastSeenAt > maxLastSeen {
					maxLastSeen = entry.LastSeenAt
				}
				items = append(items, map[string]any{
					"rule_name":         ruleName,
					"using_group":       usingGroup,
					"key_fp":            keyFP,
					"channel_id":        entry.ChannelID,
					"model_name":        entry.ModelName,
					"hit":               entry.Hit,
					"prompt_tokens":     entry.PromptTokens,
					"completion_tokens": entry.CompletionTokens,
					"cached_tokens":     entry.CachedTokens,
					"last_seen_at":      entry.LastSeenAt,
				})
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}

	return map[string]any{
		"enabled":      true,
		"mode":         "redis",
		"supported":    true,
		"size":         total,
		"last_seen_at": maxLastSeen,
		"by_rule":      byRule,
		"items":        items,
	}, nil
}

func getChannelAffinityUsageCacheStats(ruleName, usingGroup, keyFP string) (map[string]any, error) {
	ruleName = sanitizeAffinityPart(ruleName)
	keyFP = sanitizeAffinityPart(keyFP)
	usingGroup = sanitizeAffinityPart(usingGroup)

	if ruleName == "_" || keyFP == "_" {
		return nil, fmt.Errorf("缺少必要参数 rule_name 或 key_fp")
	}
	if common.RedisClient == nil {
		return map[string]any{
			"enabled":     false,
			"mode":        "redis",
			"supported":   false,
			"rule_name":   ruleName,
			"using_group": usingGroup,
			"key_fp":      keyFP,
			"size":        0,
			"items":       []any{},
		}, nil
	}

	keys := make([]string, 0, 16)
	if usingGroup != "_" {
		keys = append(keys, buildChannelAffinityCacheKey(ruleName, usingGroup, keyFP))
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var cursor uint64
		pattern := fmt.Sprintf("%s:%s:*:%s", channelAffinityNamespace, ruleName, keyFP)
		for {
			matched, next, err := common.RedisClient.Scan(ctx, cursor, pattern, channelAffinityScanBatchCount).Result()
			if err != nil {
				return nil, err
			}
			keys = append(keys, matched...)
			cursor = next
			if cursor == 0 {
				break
			}
		}
	}

	items := make([]map[string]any, 0, len(keys))
	var totalHit int64
	var totalPrompt int64
	var totalCompletion int64
	var totalCached int64
	var lastSeenAt int64

	for _, key := range keys {
		var entry channelAffinityEntry
		if !common.GetCache(key, &entry) {
			continue
		}
		totalTokens := entry.PromptTokens + entry.CompletionTokens + entry.CachedTokens
		promptCacheHit := entry.CachedTokens
		if promptCacheHit > entry.PromptTokens {
			promptCacheHit = entry.PromptTokens
		}
		items = append(items, map[string]any{
			"rule_name":               entry.RuleName,
			"using_group":             entry.UsingGroup,
			"key_fp":                  entry.KeyFP,
			"channel_id":              entry.ChannelID,
			"model_name":              entry.ModelName,
			"hit":                     entry.Hit,
			"total":                   entry.Hit,
			"prompt_tokens":           entry.PromptTokens,
			"completion_tokens":       entry.CompletionTokens,
			"total_tokens":            totalTokens,
			"cached_tokens":           entry.CachedTokens,
			"prompt_cache_hit_tokens": promptCacheHit,
			"last_seen_at":            entry.LastSeenAt,
		})
		totalHit += entry.Hit
		totalPrompt += entry.PromptTokens
		totalCompletion += entry.CompletionTokens
		totalCached += entry.CachedTokens
		if entry.LastSeenAt > lastSeenAt {
			lastSeenAt = entry.LastSeenAt
		}
	}

	return map[string]any{
		"enabled":                 true,
		"mode":                    "redis",
		"supported":               true,
		"rule_name":               ruleName,
		"using_group":             usingGroup,
		"key_fp":                  keyFP,
		"size":                    len(items),
		"hit":                     totalHit,
		"total":                   totalHit,
		"prompt_tokens":           totalPrompt,
		"completion_tokens":       totalCompletion,
		"total_tokens":            totalPrompt + totalCompletion + totalCached,
		"cached_tokens":           totalCached,
		"prompt_cache_hit_tokens": minInt64(totalCached, totalPrompt),
		"last_seen_at":            lastSeenAt,
		"items":                   items,
	}, nil
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
