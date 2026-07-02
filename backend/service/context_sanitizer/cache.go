package context_sanitizer

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"STfreApi/common"
	"STfreApi/model"
)

type policyCache struct {
	mu              sync.RWMutex
	loaded          bool
	globalDefault   *model.ContextSanitizationPolicy
	modelDefaults   map[string]*model.ContextSanitizationPolicy
	channelDefaults map[int]*model.ContextSanitizationPolicy
	channelModels   map[string]*model.ContextSanitizationPolicy
	lastLoad        time.Time
}

var cache = &policyCache{
	modelDefaults:   map[string]*model.ContextSanitizationPolicy{},
	channelDefaults: map[int]*model.ContextSanitizationPolicy{},
	channelModels:   map[string]*model.ContextSanitizationPolicy{},
}

func ReloadPolicyCache() error {
	var policies []model.ContextSanitizationPolicy
	if err := common.DB.Where("enabled = ?", true).Find(&policies).Error; err != nil {
		return err
	}

	next := &policyCache{
		loaded:          true,
		modelDefaults:   map[string]*model.ContextSanitizationPolicy{},
		channelDefaults: map[int]*model.ContextSanitizationPolicy{},
		channelModels:   map[string]*model.ContextSanitizationPolicy{},
		lastLoad:        time.Now(),
	}

	for i := range policies {
		p := policies[i]
		pp := &p
		switch p.Scope {
		case model.ContextSanitizationScopeGlobal:
			if p.ModelName == "" {
				next.globalDefault = pp
			} else {
				next.modelDefaults[strings.ToLower(p.ModelName)] = pp
			}
		case model.ContextSanitizationScopeModel:
			next.modelDefaults[strings.ToLower(p.ModelName)] = pp
		case model.ContextSanitizationScopeChannel:
			if p.ChannelId != nil {
				next.channelDefaults[*p.ChannelId] = pp
			}
		case model.ContextSanitizationScopeChannelModel:
			if p.ChannelId != nil {
				next.channelModels[channelModelKey(*p.ChannelId, p.ModelName)] = pp
			}
		}
	}

	cache.mu.Lock()
	cache.loaded = true
	cache.globalDefault = next.globalDefault
	cache.modelDefaults = next.modelDefaults
	cache.channelDefaults = next.channelDefaults
	cache.channelModels = next.channelModels
	cache.lastLoad = next.lastLoad
	cache.mu.Unlock()
	return nil
}

func ResolvePolicy(channelId int, requestedModel, mappedModel string) ResolvedPolicy {
	ensureCacheLoaded()
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	if p := cache.channelModels[channelModelKey(channelId, mappedModel)]; p != nil {
		return buildResolved(p, "channel_model:mapped")
	}
	if p := cache.channelModels[channelModelKey(channelId, requestedModel)]; p != nil {
		return buildResolved(p, "channel_model:requested")
	}
	if p := cache.channelDefaults[channelId]; p != nil {
		return buildResolved(p, "channel")
	}
	if p := cache.modelDefaults[strings.ToLower(mappedModel)]; p != nil {
		return buildResolved(p, "model:mapped")
	}
	if p := cache.modelDefaults[strings.ToLower(requestedModel)]; p != nil {
		return buildResolved(p, "model:requested")
	}
	if cache.globalDefault != nil {
		return buildResolved(cache.globalDefault, "global")
	}
	return offPolicy()
}

func CacheStats() map[string]any {
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return map[string]any{
		"loaded":                 cache.loaded,
		"last_load":              cache.lastLoad.Unix(),
		"global_default":         cache.globalDefault != nil,
		"model_policies":         len(cache.modelDefaults),
		"channel_policies":       len(cache.channelDefaults),
		"channel_model_policies": len(cache.channelModels),
	}
}

func ensureCacheLoaded() {
	cache.mu.RLock()
	loaded := cache.loaded
	cache.mu.RUnlock()
	if loaded {
		return
	}
	_ = ReloadPolicyCache()
}

func buildResolved(p *model.ContextSanitizationPolicy, source string) ResolvedPolicy {
	cfg, err := MergeConfig(p.Config)
	if err != nil {
		cfg = DefaultConfig()
	}
	return ResolvedPolicy{Policy: p, Config: cfg, Source: source}
}

func offPolicy() ResolvedPolicy {
	return ResolvedPolicy{Policy: &model.ContextSanitizationPolicy{Mode: ModeOff, Enabled: false, Version: 0}, Config: DefaultConfig(), Source: "default_off"}
}

func channelModelKey(channelId int, modelName string) string {
	return fmt.Sprintf("%d:%s", channelId, strings.ToLower(modelName))
}
