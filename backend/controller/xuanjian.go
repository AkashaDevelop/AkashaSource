package controller

// 宸汐玄鉴·管理 API (｡•ᴗ•｡)
// 所有接口仅 RoleRoot=100 可访问，普通管理员看不见这里的数据。

import (
	"encoding/json"
	"strconv"
	"time"

	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service/xuanjian"

	"github.com/gin-gonic/gin"
)

// GetXJConfig 获取当前配置（开关 + 完整策略）
func GetXJConfig(c *gin.Context) {
	cfg, enabled := xuanjian.GetConfig()
	common.OK(c, gin.H{"enabled": enabled, "config": cfg})
}

// UpdateXJConfig 保存配置并热更新
func UpdateXJConfig(c *gin.Context) {
	var req struct {
		Enabled bool             `json:"enabled"`
		Config  xuanjian.XJConfig `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if err := xuanjian.UpdateConfig(req.Config, req.Enabled); err != nil {
		common.Failf(c, common.CodeServerError, "保存配置失败: %s", err.Error())
		return
	}
	common.OKMsg(c, "配置已更新", nil)
}

// GetXJEvents 查询事件日志
func GetXJEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	db := common.DB.Model(&model.XuanJianEvent{})
	for _, key := range []string{"finding_type", "finding_group", "token_id", "user_id", "action"} {
		if v := c.Query(key); v != "" {
			db = db.Where(key+" = ?", v)
		}
	}
	if minScore := c.Query("min_score"); minScore != "" {
		db = db.Where("risk_score >= ?", minScore)
	}
	var total int64
	db.Count(&total)
	var events []model.XuanJianEvent
	if err := db.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&events).Error; err != nil {
		common.Fail(c, common.CodeServerError, "查询事件失败")
		return
	}
	common.OK(c, gin.H{"total": total, "items": events})
}

// GetXJProfiles 获取当前内存画像概览（Top 20 高风险）
func GetXJProfiles(c *gin.Context) {
	stats := xuanjian.ProfileStats()
	topRisk := xuanjian.TopRiskTokens(20)
	common.OK(c, gin.H{"stats": stats, "top_risk": topRisk})
}

// PostXJResetProfile 手动清除某 token 画像
func PostXJResetProfile(c *gin.Context) {
	tokenIDStr := c.Param("token_id")
	tokenID, err := strconv.Atoi(tokenIDStr)
	if err != nil || tokenID <= 0 {
		common.Fail(c, common.CodeParamError, "token_id 无效")
		return
	}
	xuanjian.ResetTokenProfile(tokenID)
	common.OKMsg(c, "画像已清除", nil)
}

// ── 规则管理 ──────────────────────────────────────────────────────────

type xjRuleRequest struct {
	FindingType         string   `json:"finding_type"`
	Group               string   `json:"group"`
	BaseScore           int      `json:"base_score"`
	Keywords            []string `json:"keywords"`
	RequireContext      []string `json:"require_context"`
	PromptOnly          bool     `json:"prompt_only"`
	MinCompletionTokens int      `json:"min_completion_tokens"`
	Action              string   `json:"action"`
	Enabled             *bool    `json:"enabled"`
}

// GetXJRules 列出全部规则（含内置与自定义）
func GetXJRules(c *gin.Context) {
	db := common.DB.Model(&model.XuanJianRule{})
	if group := c.Query("group"); group != "" {
		db = db.Where("`group` = ?", group)
	}
	var rows []model.XuanJianRule
	if err := db.Order("is_builtin desc, id asc").Find(&rows).Error; err != nil {
		common.Fail(c, common.CodeServerError, "查询规则失败")
		return
	}
	for i := range rows {
		fillRuleArrays(&rows[i])
	}
	common.OK(c, rows)
}

// PostXJRule 新增自定义规则
func PostXJRule(c *gin.Context) {
	var req xjRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if req.FindingType == "" || req.Group == "" || len(req.Keywords) == 0 {
		common.Fail(c, common.CodeParamError, "finding_type/group/keywords 不能为空")
		return
	}
	row := model.XuanJianRule{
		FindingType:         req.FindingType,
		Group:               req.Group,
		BaseScore:           req.BaseScore,
		PromptOnly:          req.PromptOnly,
		MinCompletionTokens: req.MinCompletionTokens,
		Action:              req.Action,
		Enabled:             true,
		IsBuiltin:           false,
	}
	applyRuleArrays(&row, req.Keywords, req.RequireContext)
	now := time.Now().Unix()
	row.CreatedAt = now
	row.UpdatedAt = now
	if err := common.DB.Create(&row).Error; err != nil {
		common.Fail(c, common.CodeServerError, "创建规则失败")
		return
	}
	row.RuleKey = "custom_" + strconv.Itoa(row.Id)
	if err := common.DB.Model(&row).Update("rule_key", row.RuleKey).Error; err != nil {
		common.Fail(c, common.CodeServerError, "写入规则标识失败")
		return
	}
	_ = xuanjian.ReloadRuleCache()
	fillRuleArrays(&row)
	common.OK(c, row)
}

// PutXJRule 编辑规则（内置规则只允许改关键词/分数/动作/启用状态，字段照传即可）
func PutXJRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.Fail(c, common.CodeParamError, "id 无效")
		return
	}
	var row model.XuanJianRule
	if err := common.DB.First(&row, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "规则不存在")
		return
	}
	var req xjRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if req.FindingType != "" {
		row.FindingType = req.FindingType
	}
	if !row.IsBuiltin && req.Group != "" {
		row.Group = req.Group
	}
	row.BaseScore = req.BaseScore
	row.PromptOnly = req.PromptOnly
	row.MinCompletionTokens = req.MinCompletionTokens
	if req.Action != "" {
		row.Action = req.Action
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if len(req.Keywords) > 0 {
		applyRuleArrays(&row, req.Keywords, req.RequireContext)
	}
	row.UpdatedAt = time.Now().Unix()
	if err := common.DB.Save(&row).Error; err != nil {
		common.Fail(c, common.CodeServerError, "保存规则失败")
		return
	}
	_ = xuanjian.ReloadRuleCache()
	fillRuleArrays(&row)
	common.OK(c, row)
}

// DeleteXJRule 删除规则（仅允许删除自定义规则，内置规则只能禁用）
func DeleteXJRule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.Fail(c, common.CodeParamError, "id 无效")
		return
	}
	var row model.XuanJianRule
	if err := common.DB.First(&row, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "规则不存在")
		return
	}
	if row.IsBuiltin {
		common.Fail(c, common.CodeParamError, "内置规则不能删除，只能禁用或恢复默认")
		return
	}
	if err := common.DB.Delete(&row).Error; err != nil {
		common.Fail(c, common.CodeServerError, "删除规则失败")
		return
	}
	_ = xuanjian.ReloadRuleCache()
	common.OKMsg(c, "规则已删除", nil)
}

// PostXJRuleToggle 快速启用/禁用某条规则
func PostXJRuleToggle(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.Fail(c, common.CodeParamError, "id 无效")
		return
	}
	var row model.XuanJianRule
	if err := common.DB.First(&row, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "规则不存在")
		return
	}
	row.Enabled = !row.Enabled
	row.UpdatedAt = time.Now().Unix()
	if err := common.DB.Save(&row).Error; err != nil {
		common.Fail(c, common.CodeServerError, "切换状态失败")
		return
	}
	_ = xuanjian.ReloadRuleCache()
	common.OK(c, gin.H{"enabled": row.Enabled})
}

// PostXJRuleReset 内置规则恢复出厂默认值
func PostXJRuleReset(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.Fail(c, common.CodeParamError, "id 无效")
		return
	}
	var row model.XuanJianRule
	if err := common.DB.First(&row, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "规则不存在")
		return
	}
	if !row.IsBuiltin {
		common.Fail(c, common.CodeParamError, "只有内置规则支持恢复默认")
		return
	}
	defaultRule, ok := xuanjian.FindBuiltinDefault(row.RuleKey)
	if !ok {
		common.Fail(c, common.CodeNotFound, "未找到该规则的出厂默认值")
		return
	}
	row.FindingType = defaultRule.FindingType
	row.Group = string(defaultRule.Group)
	row.BaseScore = defaultRule.BaseScore
	row.PromptOnly = defaultRule.PromptOnly
	row.MinCompletionTokens = defaultRule.MinCompletionTokens
	row.Action = defaultRule.Action
	row.Enabled = true
	applyRuleArrays(&row, defaultRule.Keywords, defaultRule.RequireContext)
	row.UpdatedAt = time.Now().Unix()
	if err := common.DB.Save(&row).Error; err != nil {
		common.Fail(c, common.CodeServerError, "恢复默认失败")
		return
	}
	_ = xuanjian.ReloadRuleCache()
	fillRuleArrays(&row)
	common.OK(c, row)
}

func applyRuleArrays(row *model.XuanJianRule, keywords, requireContext []string) {
	kb, _ := json.Marshal(keywords)
	row.KeywordsJSON = string(kb)
	if len(requireContext) > 0 {
		rb, _ := json.Marshal(requireContext)
		row.RequireContextJSON = string(rb)
	} else {
		row.RequireContextJSON = ""
	}
}

func fillRuleArrays(row *model.XuanJianRule) {
	_ = json.Unmarshal([]byte(row.KeywordsJSON), &row.Keywords)
	if row.RequireContextJSON != "" {
		_ = json.Unmarshal([]byte(row.RequireContextJSON), &row.RequireContext)
	}
}
