package controller

import (
	"STfreApi/common"
	"STfreApi/model"
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
	group.CreatedAt = time.Now().Unix()
	if err := common.DB.Create(&group).Error; err != nil {
		common.Fail(c, common.CodeServerError, "创建分组失败")
		return
	}
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
	// Select("*") 强制连零值字段一起更新，不然 QPM/AllowedChannels 想清空成 0/空都清不掉～
	if err := common.DB.Model(&group).Select("*").Updates(group).Error; err != nil {
		common.Fail(c, common.CodeServerError, "更新分组失败")
		return
	}
	common.OKMsg(c, "分组更新成功", nil)
}

func DeleteGroup(c *gin.Context) {
	id := c.Param("id")
	if err := common.DB.Delete(&model.Group{}, id).Error; err != nil {
		common.Fail(c, common.CodeServerError, "删除分组失败")
		return
	}
	common.OKMsg(c, "分组删除成功", nil)
}
