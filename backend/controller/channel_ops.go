package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func BatchSetChannelTag(c *gin.Context) {
	var req struct {
		Ids []int  `json:"ids"`
		Tag string `json:"tag"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if len(req.Ids) == 0 || strings.TrimSpace(req.Tag) == "" {
		common.Fail(c, common.CodeParamError, "ids 和 tag 不能为空")
		return
	}
	if err := common.DB.Model(&model.Channel{}).Where("id IN ?", req.Ids).Update("tags", req.Tag).Error; err != nil {
		common.Fail(c, common.CodeServerError, "批量设置标签失败")
		return
	}
	common.OK(c, gin.H{"updated": len(req.Ids)})
}

func GetTagModels(c *gin.Context) {
	tag := strings.TrimSpace(c.Query("tag"))
	if tag == "" {
		common.Fail(c, common.CodeParamError, "tag 不能为空")
		return
	}
	var channels []model.Channel
	if err := common.DB.Where("tags LIKE ?", "%"+tag+"%").Find(&channels).Error; err != nil {
		common.Fail(c, common.CodeServerError, "查询失败")
		return
	}
	longestModels := ""
	maxLen := 0
	for _, ch := range channels {
		if strings.TrimSpace(ch.Models) == "" {
			continue
		}
		current := strings.Split(ch.Models, ",")
		if len(current) > maxLen {
			maxLen = len(current)
			longestModels = ch.Models
		}
	}
	common.OK(c, gin.H{"success": true, "message": "", "data": longestModels})
}

func CopyChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "渠道 ID 非法")
		return
	}
	var src model.Channel
	if err = common.DB.First(&src, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}
	clone := src
	clone.Id = 0
	suffix := c.DefaultQuery("suffix", "_复制")
	resetBalance := true
	if rbStr := c.DefaultQuery("reset_balance", "true"); rbStr != "" {
		if v, e := strconv.ParseBool(rbStr); e == nil {
			resetBalance = v
		}
	}
	clone.Name = src.Name + suffix
	clone.TestTime = 0
	clone.ResponseTime = 0
	if resetBalance {
		clone.Balance = 0
		clone.UsedQuota = 0
	}
	if err = common.DB.Create(&clone).Error; err != nil {
		common.Fail(c, common.CodeServerError, "复制渠道失败")
		return
	}
	common.OK(c, gin.H{"success": true, "message": "", "data": gin.H{"id": clone.Id}})
}
