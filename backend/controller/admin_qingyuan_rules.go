package controller

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service/qingyuan"

	"github.com/gin-gonic/gin"
)

// AdminListQingyuanRules 列出所有规则
// GET /api/admin/qingyuan/rules
func AdminListQingyuanRules(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	category := c.Query("category")

	var enabled *bool
	if enabledStr := c.Query("enabled"); enabledStr != "" {
		e := enabledStr == "true" || enabledStr == "1"
		enabled = &e
	}

	rules, total, err := model.GetAllQingyuanRules(page, pageSize, category, enabled)
	if err != nil {
		common.Fail(c, common.CodeServerError, "查询规则失败")
		return
	}

	common.OK(c, gin.H{
		"rules": rules,
		"total": total,
		"page":  page,
	})
}

// AdminGetQingyuanRule 获取单条规则详情
// GET /api/admin/qingyuan/rules/:id
func AdminGetQingyuanRule(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	rule, err := model.GetQingyuanRuleById(id)
	if err != nil {
		common.Fail(c, common.CodeNotFound, "规则不存在")
		return
	}
	common.OK(c, rule)
}

// AdminCreateQingyuanRule 新增规则
// POST /api/admin/qingyuan/rules
func AdminCreateQingyuanRule(c *gin.Context) {
	userId := c.GetInt("id")
	var req model.QingyuanRule
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}

	// 校验必填字段
	if req.Category == "" || req.Name == "" || req.Keywords == "" {
		common.Fail(c, common.CodeParamError, "分类、名称、关键词不能为空")
		return
	}

	// 校验分数范围
	if req.Score < 0 || req.Score > 100 {
		common.Fail(c, common.CodeParamError, "分数必须在 0-100 之间")
		return
	}

	// 设置默认值
	if req.MatchMode == "" {
		req.MatchMode = "any"
	}
	if req.Language == "" {
		req.Language = "all"
	}

	req.CreatedBy = userId
	req.CreatedAt = time.Now().Unix()
	req.UpdatedAt = req.CreatedAt

	if err := req.Insert(); err != nil {
		common.Fail(c, common.CodeServerError, "创建失败")
		return
	}

	// 立即刷新规则缓存
	qingyuan.ReloadRulesNow()

	common.OKMsg(c, "规则创建成功", req)
}

// AdminUpdateQingyuanRule 修改规则
// PUT /api/admin/qingyuan/rules/:id
func AdminUpdateQingyuanRule(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	// 先检查规则是否存在
	existingRule, err := model.GetQingyuanRuleById(id)
	if err != nil {
		common.Fail(c, common.CodeNotFound, "规则不存在")
		return
	}

	var req model.QingyuanRule
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}

	// 校验必填字段
	if req.Category == "" || req.Name == "" || req.Keywords == "" {
		common.Fail(c, common.CodeParamError, "分类、名称、关键词不能为空")
		return
	}

	// 校验分数范围
	if req.Score < 0 || req.Score > 100 {
		common.Fail(c, common.CodeParamError, "分数必须在 0-100 之间")
		return
	}

	req.Id = id
	req.CreatedBy = existingRule.CreatedBy   // 保持创建人不变
	req.CreatedAt = existingRule.CreatedAt   // 保持创建时间不变
	req.UpdatedAt = time.Now().Unix()

	if err := req.Update(); err != nil {
		common.Fail(c, common.CodeServerError, "更新失败")
		return
	}

	qingyuan.ReloadRulesNow()
	common.OKMsg(c, "规则更新成功", req)
}

// AdminDeleteQingyuanRule 删除规则
// DELETE /api/admin/qingyuan/rules/:id
func AdminDeleteQingyuanRule(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	if err := model.DeleteQingyuanRule(id); err != nil {
		common.Fail(c, common.CodeServerError, "删除失败")
		return
	}

	qingyuan.ReloadRulesNow()
	common.OKMsg(c, "规则删除成功", nil)
}

// AdminToggleQingyuanRule 切换规则启用状态
// POST /api/admin/qingyuan/rules/:id/toggle
func AdminToggleQingyuanRule(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	rule, err := model.GetQingyuanRuleById(id)
	if err != nil {
		common.Fail(c, common.CodeNotFound, "规则不存在")
		return
	}

	rule.Enabled = !rule.Enabled
	rule.UpdatedAt = time.Now().Unix()

	if err := rule.Update(); err != nil {
		common.Fail(c, common.CodeServerError, "更新失败")
		return
	}

	qingyuan.ReloadRulesNow()
	common.OKMsg(c, "规则状态已更新", gin.H{"enabled": rule.Enabled})
}

// AdminReloadQingyuanRules 手动刷新规则缓存
// POST /api/admin/qingyuan/rules/reload
func AdminReloadQingyuanRules(c *gin.Context) {
	qingyuan.ReloadRulesNow()
	common.OKMsg(c, "规则缓存已刷新", gin.H{
		"reload_time": time.Now().Unix(),
		"last_reload": qingyuan.GetLastReloadTime().Unix(),
	})
}

// AdminListQingyuanCategories 获取规则分类列表
// GET /api/admin/qingyuan/categories
func AdminListQingyuanCategories(c *gin.Context) {
	categories, err := model.GetAllQingyuanCategories()
	if err != nil {
		common.Fail(c, common.CodeServerError, "查询分类失败")
		return
	}
	common.OK(c, categories)
}

// AdminTestQingyuanRule 测试规则（输入文本，查看哪些规则会触发）
// POST /api/admin/qingyuan/rules/test
func AdminTestQingyuanRule(c *gin.Context) {
	var req struct {
		Text     string `json:"text" binding:"required"`
		Category string `json:"category"` // 可选，只测试指定分类
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}

	// 获取所有启用的规则
	var rules []model.QingyuanRule
	query := common.DB.Where("enabled = ?", true)
	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}
	query.Find(&rules)

	// 逐个测试
	matches := []gin.H{}
	for _, rule := range rules {
		// 简单模拟检测逻辑（实际检测由 qingyuan 模块完成）
		var keywords []string
		if err := json.Unmarshal([]byte(rule.Keywords), &keywords); err != nil {
			continue
		}

		lowerText := strings.ToLower(req.Text)
		matched := false
		matchedKeyword := ""

		for _, kw := range keywords {
			if strings.Contains(lowerText, strings.ToLower(kw)) {
				matched = true
				matchedKeyword = kw
				break
			}
		}

		if matched {
			matches = append(matches, gin.H{
				"rule_id":         rule.Id,
				"category":        rule.Category,
				"name":            rule.Name,
				"score":           rule.Score,
				"matched_keyword": matchedKeyword,
			})
		}
	}

	common.OK(c, gin.H{
		"text":    req.Text,
		"matches": matches,
		"count":   len(matches),
	})
}

// AdminBatchImportQingyuanRules 批量导入规则
// POST /api/admin/qingyuan/rules/import
func AdminBatchImportQingyuanRules(c *gin.Context) {
	userId := c.GetInt("id")
	var req struct {
		Rules []model.QingyuanRule `json:"rules" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}

	successCount := 0
	now := time.Now().Unix()

	for _, rule := range req.Rules {
		rule.CreatedBy = userId
		rule.CreatedAt = now
		rule.UpdatedAt = now
		if err := rule.Insert(); err == nil {
			successCount++
		}
	}

	qingyuan.ReloadRulesNow()
	common.OKMsg(c, "批量导入完成", gin.H{
		"total":   len(req.Rules),
		"success": successCount,
	})
}

// AdminExportQingyuanRules 导出所有规则
// GET /api/admin/qingyuan/rules/export
func AdminExportQingyuanRules(c *gin.Context) {
	var rules []model.QingyuanRule
	if err := common.DB.Order("category, sort_order, id").Find(&rules).Error; err != nil {
		common.Fail(c, common.CodeServerError, "导出失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "导出成功",
		"data": gin.H{
			"rules":       rules,
			"export_time": time.Now().Unix(),
			"count":       len(rules),
		},
	})
}
