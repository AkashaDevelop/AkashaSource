package controller

// 宸汐玄鉴·管理 API (｡•ᴗ•｡)
// 所有接口仅 RoleRoot=100 可访问，普通管理员看不见这里的数据。

import (
	"strconv"

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
