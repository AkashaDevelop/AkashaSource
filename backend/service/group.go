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
	return ResolveUsingGroupFull(userGroup, "", tokenGroup)
}

// ResolveUsingGroupFull 🌸 带 extra_groups(直接授予)的完整版～
// 校验令牌指定分组时，把"公开+基础+特殊解锁+额外授予"全算进可用集，extra 才能真正生效。
func ResolveUsingGroupFull(userGroup string, extraGroups string, tokenGroup string) (string, error) {
	if tokenGroup == "" {
		return userGroup, nil
	}
	usable := GetUserUsableGroupsFull(userGroup, extraGroups)
	if _, ok := usable[tokenGroup]; !ok && tokenGroup != "auto" {
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
	return ResolveBillingGroupFull(userGroup, "", tokenGroup)
}

// ResolveBillingGroupFull 🌸 带 extra_groups 的计费分组解析～
func ResolveBillingGroupFull(userGroup string, extraGroups string, tokenGroup string) string {
	usingGroup, err := ResolveUsingGroupFull(userGroup, extraGroups, tokenGroup)
	if err != nil || usingGroup == "auto" {
		return userGroup
	}
	return usingGroup
}

// GetUserUsableGroups 汇总"这个用户分组能用哪些分组"～重构后的裁决逻辑(以 Group 表为权威)：
//   ① 所有 visibility=public 的公开分组(游客/普通用户默认可选)
//   ② 用户自己的基础分组(哪怕它是 hidden 也一定能用)
//   ③ 该基础分组通过 GroupSpecialGrant 解锁的特殊分组(可解锁 hidden 分组)
// 注意：直接授予用户的 extra_groups 不在这里(拿不到用户对象)，由 GetUserUsableGroupsFull 叠加～
func GetUserUsableGroups(userGroup string) map[string]string {
	usable := model.GetAllPublicGroups() // ① 公开分组

	if userGroup == "" {
		return usable
	}

	// ② 基础分组自身
	if _, ok := usable[userGroup]; !ok {
		desc := model.GetGroupDescription(userGroup)
		if desc == "" {
			desc = "用户分组"
		}
		usable[userGroup] = desc
	}

	// ③ 基础分组解锁的特殊分组
	for _, sg := range model.GetSpecialGrantsByBase(userGroup) {
		if _, ok := usable[sg]; !ok {
			desc := model.GetGroupDescription(sg)
			if desc == "" {
				desc = "特殊分组"
			}
			usable[sg] = desc
		}
	}

	return usable
}

// GetUserUsableGroupsFull 在 GetUserUsableGroups 基础上，叠加"直接授予用户"的 extra_groups～
// 这是最完整的"用户实际可用分组"，请求准入/前端下拉都该用它(能拿到 user 对象时)。
func GetUserUsableGroupsFull(userGroup string, extraGroups string) map[string]string {
	usable := GetUserUsableGroups(userGroup)
	for _, g := range strings.Split(extraGroups, ",") {
		name := strings.TrimSpace(g)
		if name == "" {
			continue
		}
		if _, ok := usable[name]; !ok {
			desc := model.GetGroupDescription(name)
			if desc == "" {
				desc = "额外分组"
			}
			usable[name] = desc
		}
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

// GetGroupRateLimits 一次取回分组的三项速率限制：RPM / TPM / RPD，<=0 都表示不限～
func GetGroupRateLimits(groupName string) (rpm int, tpm int, rpd int) {
	if groupName == "" {
		return 0, 0, 0
	}
	var g model.Group
	if err := common.DB.Where("name = ?", groupName).First(&g).Error; err != nil {
		return 0, 0, 0
	}
	return g.QPM, g.TPM, g.RPD
}
