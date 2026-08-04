package xuanjian

// 宸汐玄鉴 · 规则谱校准仪 (｡•̀ᴗ-)✧
//
// SeedBuiltinRules 只在表为空时播种，所以已经跑起来的实例永远吃不到规则校准——
// 那批把 "whoami"、"有哪些可用的模型"、"帮我数 token" 当成攻击的旧规则会一直误报，
// 而 throttle / suspend_token / billing_penalty 那套处置也会继续空转下去。
//
// 这里给存量库做一次性校准。三条安全线和清源那边一致：
//   ① 只动 is_builtin = true 的记录，管理员自建规则一律不碰
//   ② 停用而不是删除，管理员想恢复随时能在后台打开
//   ③ 用 option 记版本号，只跑一次，不会反复覆盖管理员之后的调整

import (
	"encoding/json"
	"log"
	"time"

	"STfreApi/common"
	"STfreApi/model"
)

// rulesRecalibrateVersion 校准版本号，升一位就会再跑一轮
const rulesRecalibrateVersion = "2026.08.04"

// optionKeyRuleCalibration 记录已完成的校准版本
const optionKeyRuleCalibration = "xuanjian_rule_calibration"

// retiredRuleKeys 本次校准停用的规则
//
// R4 pricing_probe 的关键词是 "count the tokens in" / "每个token多少钱" /
// "计算一下这段文字的token数"——这些是 API 用户最正常不过的问题，
// 甚至本平台自己就该提供这个能力。把它当成"逆向探测"实在冤枉人喵～
var retiredRuleKeys = []string{"R4"}

// RecalibrateRules 给存量规则库做一次性校准
func RecalibrateRules() {
	if common.DB == nil {
		return
	}

	// 表还空着说明是全新部署，播种用的就是新规则谱，不用校准
	var total int64
	if err := common.DB.Model(&model.XuanJianRule{}).Count(&total).Error; err != nil || total == 0 {
		return
	}

	if ruleCalibrationDone() {
		return
	}

	updated := 0
	for _, kr := range DefaultRules() {
		if applyRuleCalibration(kr) {
			updated++
		}
	}

	retired := 0
	for _, key := range retiredRuleKeys {
		if disableRule(key) {
			retired++
		}
	}

	markRuleCalibrationDone()
	log.Printf("[xuanjian] 规则谱校准完成（版本 %s）：更新 %d 条，停用 %d 条",
		rulesRecalibrateVersion, updated, retired)
}

// applyRuleCalibration 把一条新规则的分数/关键词/上下文/处置动作写回库里对应的旧记录
func applyRuleCalibration(kr KeywordRule) bool {
	var existing model.XuanJianRule
	err := common.DB.Where("rule_key = ? AND is_builtin = ?", kr.ID, true).First(&existing).Error
	if err != nil {
		return false
	}

	keywordsJSON, err := json.Marshal(kr.Keywords)
	if err != nil {
		return false
	}
	var requireContextJSON string
	if len(kr.RequireContext) > 0 {
		b, err := json.Marshal(kr.RequireContext)
		if err != nil {
			return false
		}
		requireContextJSON = string(b)
	}

	updates := map[string]interface{}{
		"finding_type":          kr.FindingType,
		"group":                 string(kr.Group),
		"base_score":            kr.BaseScore,
		"keywords_json":         string(keywordsJSON),
		"require_context_json":  requireContextJSON,
		"prompt_only":           kr.PromptOnly,
		"min_completion_tokens": kr.MinCompletionTokens,
		"action":                kr.Action,
		"updated_at":            time.Now().Unix(),
	}
	if err := common.DB.Model(&model.XuanJianRule{}).Where("id = ?", existing.Id).
		Updates(updates).Error; err != nil {
		log.Printf("[xuanjian] 规则 %s 校准失败: %v", kr.ID, err)
		return false
	}
	return true
}

// disableRule 停用一条退休规则（保留记录，管理员随时可以重新打开）
func disableRule(ruleKey string) bool {
	result := common.DB.Model(&model.XuanJianRule{}).
		Where("rule_key = ? AND is_builtin = ?", ruleKey, true).
		Updates(map[string]interface{}{
			"enabled":    false,
			"updated_at": time.Now().Unix(),
		})
	return result.Error == nil && result.RowsAffected > 0
}

func ruleCalibrationDone() bool {
	var opt model.Option
	if err := common.DB.Where("`key` = ?", optionKeyRuleCalibration).First(&opt).Error; err != nil {
		return false
	}
	return opt.Value == rulesRecalibrateVersion
}

func markRuleCalibrationDone() {
	if err := common.DB.Save(&model.Option{
		Key:   optionKeyRuleCalibration,
		Value: rulesRecalibrateVersion,
	}).Error; err != nil {
		log.Printf("[xuanjian] 记录校准版本失败: %v", err)
	}
}
