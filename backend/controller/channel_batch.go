package controller

import (
	"STfreApi/common"
	"STfreApi/model"

	"github.com/gin-gonic/gin"
)

func BatchUpdateChannelStatus(c *gin.Context) {
	var req struct {
		ChannelIDs []int `json:"channel_ids"`
		Status     int   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}

	if len(req.ChannelIDs) == 0 {
		common.Fail(c, common.CodeParamError, "渠道ID列表不能为空")
		return
	}

	if err := common.DB.Model(&model.Channel{}).Where("id IN ?", req.ChannelIDs).Update("status", req.Status).Error; err != nil {
		common.Fail(c, common.CodeServerError, "更新失败")
		return
	}

	common.OKMsg(c, "批量更新成功", nil)
}

func BatchUpdateChannelPriority(c *gin.Context) {
	var req struct {
		ChannelIDs []int `json:"channel_ids"`
		Priority   int   `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}

	if len(req.ChannelIDs) == 0 {
		common.Fail(c, common.CodeParamError, "渠道ID列表不能为空")
		return
	}

	if err := common.DB.Model(&model.Channel{}).Where("id IN ?", req.ChannelIDs).Update("priority", req.Priority).Error; err != nil {
		common.Fail(c, common.CodeServerError, "更新失败")
		return
	}

	common.OKMsg(c, "批量更新成功", nil)
}

func BatchDeleteChannels(c *gin.Context) {
	var req struct {
		ChannelIDs []int `json:"channel_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}

	if len(req.ChannelIDs) == 0 {
		common.Fail(c, common.CodeParamError, "渠道ID列表不能为空")
		return
	}

	if err := common.DB.Where("id IN ?", req.ChannelIDs).Delete(&model.Channel{}).Error; err != nil {
		common.Fail(c, common.CodeServerError, "删除失败")
		return
	}

	common.OKMsg(c, "批量删除成功", nil)
}
