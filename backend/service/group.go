package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"STfreApi/common"
	"STfreApi/model"
)

// ResolveUsingGroup 令牌鉴权时用：算出这次请求最终该用哪个分组～
// tokenGroup 为空就乖乖跟着用户自己的分组走；非空则必须在用户可用分组名单里，
// 而且要么在计费倍率表里挂了号，要么就是特殊值 "auto"，否则直接拒绝这次请求～
func ResolveUsingGroup(userGroup string, tokenGroup string) (string, error) {
	if tokenGroup == "" {
		return userGroup, nil
	}
	if !GroupInUserUsableGroups(userGroup, tokenGroup) {
		return "", fmt.Errorf("无权访问 %s 分组", tokenGroup)
	}
	if tokenGroup != "auto" && !common.ContainsGroupRatio(tokenGroup) {
		return "", fmt.Errorf("分组 %s 已被弃用", tokenGroup)
	}
	return tokenGroup, nil
}

// ResolveBillingGroup 算计费该用哪个分组的名字～auto 分组本身没有倍率可查（它只是"帮你自动挑分组"的壳子），
// 所以 auto 令牌老老实实按用户自己的分组算账；其它情况直接用解析出来的 usingGroup～
func ResolveBillingGroup(userGroup string, tokenGroup string) string {
	usingGroup, err := ResolveUsingGroup(userGroup, tokenGroup)
	if err != nil || usingGroup == "auto" {
		return userGroup
	}
	return usingGroup
}

// GetUserUsableGroups 汇总"这个用户分组能用哪些分组"：以 model.Group 表为权威名单，
// 再叠加 common.GroupSpecialUsableGroup 里 "+:xxx"/"-:xxx" 的增删规则，
// 最后保证用户自己的分组一定在里面（哪怕表里没登记）～
func GetUserUsableGroups(userGroup string) map[string]string {
	usable := make(map[string]string)

	var groups []model.Group
	if err := common.DB.Find(&groups).Error; err == nil {
		for _, g := range groups {
			name := strings.TrimSpace(g.Name)
			if name != "" {
				usable[name] = g.Description
			}
		}
	}

	if userGroup == "" {
		return usable
	}

	if rules, ok := common.GetGroupSpecialUsableGroup(userGroup); ok {
		for specialGroup, desc := range rules {
			switch {
			case strings.HasPrefix(specialGroup, "-:"):
				delete(usable, strings.TrimPrefix(specialGroup, "-:"))
			case strings.HasPrefix(specialGroup, "+:"):
				usable[strings.TrimPrefix(specialGroup, "+:")] = desc
			default:
				usable[specialGroup] = desc
			}
		}
	}

	if _, ok := usable[userGroup]; !ok {
		usable[userGroup] = "用户分组"
	}

	return usable
}

// GroupInUserUsableGroups 判断某分组名是否在该用户分组的可用名单里～
func GroupInUserUsableGroups(userGroup, groupName string) bool {
	_, ok := GetUserUsableGroups(userGroup)[groupName]
	return ok
}

// GetUserAutoGroup "auto" 分组自动轮询时，取全局候选分组 ∩ 用户实际可用分组，顺序跟着全局配置走～
func GetUserAutoGroup(userGroup string) []string {
	usable := GetUserUsableGroups(userGroup)
	result := make([]string, 0)
	for _, g := range common.GetAutoGroups() {
		if _, ok := usable[g]; ok {
			result = append(result, g)
		}
	}
	return result
}

// GetUserGroupRatio 用户分组倍率：先看"用户分组×使用分组"有没有专属折扣，没有就退回普通分组倍率～
func GetUserGroupRatio(userGroup, usingGroup string) float64 {
	if ratio, ok := common.GetGroupGroupRatio(userGroup, usingGroup); ok {
		return ratio
	}
	return common.GetGroupRatio(usingGroup)
}

// GetGroupModelRatioOverride 查某分组针对某模型是否配置了专属倍率覆盖(model.Group.ModelRatios)～
func GetGroupModelRatioOverride(groupName string, modelName string) (float64, bool) {
	if groupName == "" || modelName == "" {
		return 0, false
	}
	var g model.Group
	if err := common.DB.Where("name = ?", groupName).First(&g).Error; err != nil {
		return 0, false
	}
	if strings.TrimSpace(g.ModelRatios) == "" {
		return 0, false
	}
	var overrides map[string]float64
	if err := json.Unmarshal([]byte(g.ModelRatios), &overrides); err != nil {
		return 0, false
	}
	ratio, ok := overrides[modelName]
	return ratio, ok
}

// GetGroupAllowedChannels 查某分组是否配置了渠道白名单(model.Group.AllowedChannels)，nil 表示不限制～
func GetGroupAllowedChannels(groupName string) map[int]struct{} {
	if groupName == "" {
		return nil
	}
	var g model.Group
	if err := common.DB.Where("name = ?", groupName).First(&g).Error; err != nil {
		return nil
	}
	if strings.TrimSpace(g.AllowedChannels) == "" {
		return nil
	}
	allowed := make(map[int]struct{})
	for _, idStr := range strings.Split(g.AllowedChannels, ",") {
		idStr = strings.TrimSpace(idStr)
		if idStr == "" {
			continue
		}
		if id, err := strconv.Atoi(idStr); err == nil {
			allowed[id] = struct{}{}
		}
	}
	return allowed
}

// GetGroupQPM 查某分组配置的每分钟请求数上限，<=0 表示不限～
func GetGroupQPM(groupName string) int {
	if groupName == "" {
		return 0
	}
	var g model.Group
	if err := common.DB.Where("name = ?", groupName).First(&g).Error; err != nil {
		return 0
	}
	return g.QPM
}
