package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func GetAllGroups(c *gin.Context) {
	var groups []model.Group
	if err := common.DB.Find(&groups).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取分组失败")
		return
	}
	common.OK(c, groups)
}

func AddGroup(c *gin.Context) {
	var group model.Group
	if err := c.ShouldBindJSON(&group); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if group.Visibility != model.GroupVisibilityHidden {
		group.Visibility = model.GroupVisibilityPublic // 默认公开～
	}
	if group.Ratio <= 0 {
		group.Ratio = 1
	}
	group.CreatedAt = time.Now().Unix()
	if err := common.DB.Create(&group).Error; err != nil {
		common.Fail(c, common.CodeServerError, "创建分组失败")
		return
	}
	model.SyncGroupRatioToMemory() // 🌸 倍率同步进计费表
	common.OKMsg(c, "分组创建成功", group)
}

func UpdateGroup(c *gin.Context) {
	var group model.Group
	if err := c.ShouldBindJSON(&group); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if group.Id == 0 {
		common.Fail(c, common.CodeParamError, "ID 必填")
		return
	}
	if group.Visibility != model.GroupVisibilityHidden {
		group.Visibility = model.GroupVisibilityPublic
	}
	if group.Ratio <= 0 {
		group.Ratio = 1
	}
	// Select("*") 强制连零值字段一起更新，不然 QPM/AllowedChannels 想清空成 0/空都清不掉～
	if err := common.DB.Model(&group).Select("*").Updates(group).Error; err != nil {
		common.Fail(c, common.CodeServerError, "更新分组失败")
		return
	}
	model.SyncGroupRatioToMemory() // 🌸 倍率同步进计费表
	common.OKMsg(c, "分组更新成功", nil)
}

func DeleteGroup(c *gin.Context) {
	id := c.Param("id")
	if err := common.DB.Delete(&model.Group{}, id).Error; err != nil {
		common.Fail(c, common.CodeServerError, "删除分组失败")
		return
	}
	model.SyncGroupRatioToMemory() // 🌸 倍率同步进计费表
	common.OKMsg(c, "分组删除成功", nil)
}

// ═══════════════ 🌸 特殊分组授权(基础分组→解锁的特殊分组) ═══════════════

// GetSpecialGrants 列出所有特殊授权规则～
func GetSpecialGrants(c *gin.Context) {
	var grants []model.GroupSpecialGrant
	if err := common.DB.Order("base_group asc").Find(&grants).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取特殊授权失败")
		return
	}
	common.OK(c, grants)
}

// AddSpecialGrant 新增一条"基础分组→特殊分组"授权～
func AddSpecialGrant(c *gin.Context) {
	var grant model.GroupSpecialGrant
	if err := c.ShouldBindJSON(&grant); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	grant.BaseGroup = strings.TrimSpace(grant.BaseGroup)
	grant.SpecialGroup = strings.TrimSpace(grant.SpecialGroup)
	if grant.BaseGroup == "" || grant.SpecialGroup == "" {
		common.Fail(c, common.CodeParamError, "基础分组和特殊分组都不能为空")
		return
	}
	if grant.BaseGroup == grant.SpecialGroup {
		common.Fail(c, common.CodeParamError, "基础分组和特殊分组不能相同")
		return
	}
	// 去重
	var cnt int64
	common.DB.Model(&model.GroupSpecialGrant{}).Where("base_group = ? AND special_group = ?", grant.BaseGroup, grant.SpecialGroup).Count(&cnt)
	if cnt > 0 {
		common.Fail(c, common.CodeConflict, "该授权规则已存在")
		return
	}
	grant.CreatedAt = time.Now().Unix()
	if err := common.DB.Create(&grant).Error; err != nil {
		common.Fail(c, common.CodeServerError, "创建授权失败")
		return
	}
	common.OKMsg(c, "授权创建成功", grant)
}

// DeleteSpecialGrant 删除一条特殊授权～
func DeleteSpecialGrant(c *gin.Context) {
	id := c.Param("id")
	if err := common.DB.Delete(&model.GroupSpecialGrant{}, id).Error; err != nil {
		common.Fail(c, common.CodeServerError, "删除授权失败")
		return
	}
	common.OKMsg(c, "授权删除成功", nil)
}
