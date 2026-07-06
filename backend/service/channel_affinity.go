package service

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"STfreApi/common"
	"STfreApi/pkg/cachex"
	"github.com/gin-gonic/gin"
)

const (
	channelAffinityCacheNamespace = "stfreapi:channel_affinity:v1"
	defaultMaxEntries             = 100000
	defaultTTLSeconds             = 3600
)

var (
	affinityCache     *cachex.HybridCache[int]
	affinityCacheOnce sync.Once
	regexCache        sync.Map
)

type AffinityRule struct {
	Name              string
	ModelRegex        string
	PathRegex         string
	UserAgentInclude  []string
	KeySourceType     string
	KeySourceKey      string
	KeySourcePath     string
	ValueRegex        string
	TTLSeconds        int
	IncludeRuleName   bool
	IncludeUsingGroup bool
	SkipRetryOnFail   bool
	ParamOverride     map[string]interface{}
}

type AffinityMeta struct {
	CacheKey      string
	TTLSeconds    int
	RuleName      string
	SkipRetry     bool
	ParamTemplate map[string]interface{}
}

func getAffinityCache() *cachex.HybridCache[int] {
	affinityCacheOnce.Do(func() {
		affinityCache = cachex.NewHybridCache[int](
			channelAffinityCacheNamespace,
			common.RedisClient,
			func() bool { return common.RedisClient != nil },
			defaultMaxEntries,
			time.Duration(defaultTTLSeconds)*time.Second,
		)
	})
	return affinityCache
}

func matchRegex(pattern, s string) bool {
	if pattern == "" || s == "" {
		return false
	}
	re, ok := regexCache.Load(pattern)
	if !ok {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return false
		}
		re = compiled
		regexCache.Store(pattern, re)
	}
	return re.(*regexp.Regexp).MatchString(s)
}

func matchAnyInclude(patterns []string, s string) bool {
	if len(patterns) == 0 || s == "" {
		return false
	}
	sLower := strings.ToLower(s)
	for _, p := range patterns {
		if p == "" {
			continue
		}
		if strings.Contains(sLower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

func extractAffinityValue(c *gin.Context, rule AffinityRule) string {
	switch rule.KeySourceType {
	case "context_int":
		if rule.KeySourceKey == "" {
			return ""
		}
		v := c.GetInt(rule.KeySourceKey)
		if v <= 0 {
			return ""
		}
		return strconv.Itoa(v)
	case "context_string":
		if rule.KeySourceKey == "" {
			return ""
		}
		return strings.TrimSpace(c.GetString(rule.KeySourceKey))
	case "header":
		if rule.KeySourceKey == "" {
			return ""
		}
		return strings.TrimSpace(c.GetHeader(rule.KeySourceKey))
	case "query":
		if rule.KeySourceKey == "" {
			return ""
		}
		return strings.TrimSpace(c.Query(rule.KeySourceKey))
	default:
		return ""
	}
}

func buildCacheKey(rule AffinityRule, affinityValue, usingGroup string) string {
	parts := []string{}
	if rule.IncludeRuleName {
		parts = append(parts, rule.Name)
	}
	if rule.IncludeUsingGroup {
		parts = append(parts, usingGroup)
	}
	parts = append(parts, affinityValue)
	return strings.Join(parts, ":")
}

func GetPreferredChannelByAffinity(c *gin.Context, rules []AffinityRule, modelName, usingGroup string) (int, *AffinityMeta) {
	if len(rules) == 0 {
		return 0, nil
	}

	requestPath := c.Request.URL.Path
	userAgent := c.GetHeader("User-Agent")

	for _, rule := range rules {
		if rule.ModelRegex != "" && !matchRegex(rule.ModelRegex, modelName) {
			continue
		}

		if rule.PathRegex != "" && !matchRegex(rule.PathRegex, requestPath) {
			continue
		}

		if len(rule.UserAgentInclude) > 0 && !matchAnyInclude(rule.UserAgentInclude, userAgent) {
			continue
		}

		affinityValue := extractAffinityValue(c, rule)
		if affinityValue == "" {
			continue
		}

		if rule.ValueRegex != "" && !matchRegex(rule.ValueRegex, affinityValue) {
			continue
		}

		cacheKey := buildCacheKey(rule, affinityValue, usingGroup)
		cache := getAffinityCache()
		channelID, found := cache.Get(cacheKey)
		if found && channelID > 0 {
			meta := &AffinityMeta{
				CacheKey:      cacheKey,
				TTLSeconds:    rule.TTLSeconds,
				RuleName:      rule.Name,
				SkipRetry:     rule.SkipRetryOnFail,
				ParamTemplate: rule.ParamOverride,
			}
			return channelID, meta
		}
	}

	return 0, nil
}

func RecordChannelAffinity(c *gin.Context, channelID int, meta *AffinityMeta) {
	if meta == nil || meta.CacheKey == "" || channelID <= 0 {
		return
	}

	ttl := time.Duration(meta.TTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = time.Duration(defaultTTLSeconds) * time.Second
	}

	cache := getAffinityCache()
	cache.Set(meta.CacheKey, channelID, ttl)
}

func ApplyParamOverride(c *gin.Context, paramOverride map[string]interface{}) bool {
	if len(paramOverride) == 0 {
		return false
	}

	for key, value := range paramOverride {
		c.Set(key, value)
	}
	return true
}

func ClearAffinityCacheAll() int {
	cache := getAffinityCache()
	keys := cache.Keys()
	if len(keys) > 0 {
		cache.DeleteMany(keys)
	}
	return len(keys)
}

func GenerateAffinityKey(userID int, model string) string {
	data := fmt.Sprintf("user:%d:model:%s", userID, model)
	hash := sha1.Sum([]byte(data))
	return hex.EncodeToString(hash[:8])
}

