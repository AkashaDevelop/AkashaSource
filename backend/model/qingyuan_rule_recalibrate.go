package model

import (
	"encoding/json"
	"log"
	"time"

	"STfreApi/common"
)

// 宸汐清源 · 规则谱校准仪 (｡•̀ᴗ-)✧
//
// 种子规则只在表为空时播种，所以已经跑起来的实例永远吃不到新的规则校准——
// 那批把 "sudo"、"执行命令"、"上传到" 判成 70+ 分的旧规则会一直留在库里误伤用户。
//
// 这个文件负责给存量库做一次性校准：按规则名找到旧记录，换上新的分数和关键词，
// 顺便把那条救不回来的规则停用掉。
//
// 三条安全线，避免踩到管理员自己的心血：
//   ① 只动 CreatedBy == 0 的记录（系统播种的），管理员自建规则一律不碰
//   ② 停用而不是删除，管理员想恢复随时能在后台打开
//   ③ 用 option 记版本号，只跑一次，重启不会反复覆盖管理员之后的调整

// qingyuanRecalibrateVersion 校准版本号，升一位就会再跑一轮校准
const qingyuanRecalibrateVersion = "2026.08.04"

// optionKeyQingyuanRuleCalibration 记录已完成的校准版本
const optionKeyQingyuanRuleCalibration = "qingyuan_rule_calibration"

// qingyuanRuleRenames 本次校准里改了名字的规则：旧名 → 新名
// 改名是因为原名描述的行为范围太宽，新名更准确地说明了它到底在抓什么
var qingyuanRuleRenames = map[string]string{
	"新系统消息":   "伪造系统指令",
	"系统命令执行":  "诱导执行系统命令",
	"sudo模拟":  "提权诱导",
	"外部URL发送": "外部URL外发",
	"导出全部数据":  "批量导出敏感数据",
	"敏感信息查询":  "凭证信息索取",
	"数据上传":    "隐蔽外传",
}

// qingyuanRetiredRules 本次校准停用的规则
// "禁止词突破" 的关键词是 "you can say" / "你可以说" / "允许你"——
// 纯日常口语，加任何上下文约束都没法把它和攻击区分开，只能请它退休喵～
var qingyuanRetiredRules = []string{"禁止词突破"}

// RecalibrateQingyuanRules 给存量规则库做一次性校准
func RecalibrateQingyuanRules() {
	if common.DB == nil {
		return
	}

	// 表还是空的说明是全新部署，播种时用的就是新规则谱，不用校准
	var total int64
	if err := common.DB.Model(&QingyuanRule{}).Count(&total).Error; err != nil || total == 0 {
		return
	}

	if calibrationDone() {
		return
	}

	updated := 0
	renamed := 0
	for _, r := range defaultQingyuanRules() {
		if applyRuleCalibration(r) {
			updated++
		}
	}
	renamed = countRenamed()

	retired := 0
	for _, name := range qingyuanRetiredRules {
		if disableRetiredRule(name) {
			retired++
		}
	}

	markCalibrationDone()
	log.Printf("[宸汐清源] 规则谱校准完成（版本 %s）：更新 %d 条，改名 %d 条，停用 %d 条",
		qingyuanRecalibrateVersion, updated, renamed, retired)
}

// applyRuleCalibration 把一条新规则的分数/关键词/上下文写回库里对应的旧记录
func applyRuleCalibration(r seedRule) bool {
	keywordsJSON, err := json.Marshal(r.Keywords)
	if err != nil {
		return false
	}

	// 先按新名字找；找不到再按旧名字找（说明这条规则本次改了名）
	var existing QingyuanRule
	err = common.DB.Where("category = ? AND name = ? AND created_by = ?", r.Category, r.Name, 0).
		First(&existing).Error
	if err != nil {
		oldName := lookupOldName(r.Name)
		if oldName == "" {
			return false
		}
		if err := common.DB.Where("category = ? AND name = ? AND created_by = ?", r.Category, oldName, 0).
			First(&existing).Error; err != nil {
			return false
		}
	}

	updates := map[string]interface{}{
		"name":             r.Name,
		"description":      r.Description,
		"score":            r.Score,
		"keywords":         string(keywordsJSON),
		"context_required": r.ContextRequired,
		"match_mode":       r.MatchMode,
		"updated_at":       time.Now().Unix(),
	}
	if err := common.DB.Model(&QingyuanRule{}).Where("id = ?", existing.Id).
		Updates(updates).Error; err != nil {
		log.Printf("[宸汐清源] 规则 %q 校准失败: %v", r.Name, err)
		return false
	}
	return true
}

// disableRetiredRule 停用一条退休规则（保留记录，管理员随时可以重新打开）
func disableRetiredRule(name string) bool {
	result := common.DB.Model(&QingyuanRule{}).
		Where("name = ? AND created_by = ?", name, 0).
		Updates(map[string]interface{}{
			"enabled":    false,
			"updated_at": time.Now().Unix(),
		})
	return result.Error == nil && result.RowsAffected > 0
}

// lookupOldName 反查某个新规则名对应的旧名
func lookupOldName(newName string) string {
	for old, current := range qingyuanRuleRenames {
		if current == newName {
			return old
		}
	}
	return ""
}

// countRenamed 统计实际发生改名的规则数量（旧名已经查不到了就说明改成功了）
func countRenamed() int {
	count := 0
	for oldName := range qingyuanRuleRenames {
		var n int64
		common.DB.Model(&QingyuanRule{}).Where("name = ?", oldName).Count(&n)
		if n == 0 {
			count++
		}
	}
	return count
}

func calibrationDone() bool {
	var opt Option
	if err := common.DB.Where("`key` = ?", optionKeyQingyuanRuleCalibration).First(&opt).Error; err != nil {
		return false
	}
	return opt.Value == qingyuanRecalibrateVersion
}

func markCalibrationDone() {
	if err := common.DB.Save(&Option{
		Key:   optionKeyQingyuanRuleCalibration,
		Value: qingyuanRecalibrateVersion,
	}).Error; err != nil {
		log.Printf("[宸汐清源] 记录校准版本失败: %v", err)
	}
}
