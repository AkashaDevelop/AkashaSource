package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetAllModelsMeta(c *gin.Context) {
	page, pageSize, offset := parsePageParams(c)
	items, err := model.GetAllModelsMeta(offset, pageSize)
	if err != nil {
		common.Fail(c, common.CodeServerError, "获取模型元数据失败")
		return
	}
	var total int64
	if err = common.DB.Model(&model.ModelMeta{}).Count(&total).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取模型总数失败")
		return
	}
	common.OK(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func SearchModelsMeta(c *gin.Context) {
	keyword := c.Query("keyword")
	page, pageSize, offset := parsePageParams(c)
	items, total, err := model.SearchModelsMeta(keyword, offset, pageSize)
	if err != nil {
		common.Fail(c, common.CodeServerError, "搜索模型元数据失败")
		return
	}
	common.OK(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func GetModelMeta(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "模型 ID 非法")
		return
	}
	item, err := model.GetModelMetaByID(id)
	if err != nil {
		common.Fail(c, common.CodeNotFound, "模型不存在")
		return
	}
	common.OK(c, item)
}

func CreateModelMeta(c *gin.Context) {
	var item model.ModelMeta
	if err := c.ShouldBindJSON(&item); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if item.ModelName == "" {
		common.Fail(c, common.CodeParamError, "模型名称不能为空")
		return
	}
	if dup, err := model.IsModelNameDuplicated(0, item.ModelName); err != nil {
		common.Fail(c, common.CodeServerError, "校验模型名称失败")
		return
	} else if dup {
		common.Fail(c, common.CodeConflict, "模型名称已存在")
		return
	}
	if err := item.Insert(); err != nil {
		common.Fail(c, common.CodeServerError, "创建模型失败")
		return
	}
	common.OKMsg(c, "创建模型成功", item)
}

func UpdateModelMeta(c *gin.Context) {
	statusOnly := c.Query("status_only") == "true"
	var item model.ModelMeta
	if err := c.ShouldBindJSON(&item); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if item.Id == 0 {
		common.Fail(c, common.CodeParamError, "缺少模型 ID")
		return
	}
	if statusOnly {
		if err := common.DB.Model(&model.ModelMeta{}).Where("id = ?", item.Id).Update("enabled", item.Enabled).Error; err != nil {
			common.Fail(c, common.CodeServerError, "更新模型状态失败")
			return
		}
		common.OKMsg(c, "更新模型状态成功", nil)
		return
	}
	if dup, err := model.IsModelNameDuplicated(item.Id, item.ModelName); err != nil {
		common.Fail(c, common.CodeServerError, "校验模型名称失败")
		return
	} else if dup {
		common.Fail(c, common.CodeConflict, "模型名称已存在")
		return
	}
	if err := item.Update(); err != nil {
		common.Fail(c, common.CodeServerError, "更新模型失败")
		return
	}
	common.OKMsg(c, "更新模型成功", nil)
}

func DeleteModelMeta(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "模型 ID 非法")
		return
	}
	item := model.ModelMeta{Id: id}
	if err = item.Delete(); err != nil {
		common.Fail(c, common.CodeServerError, "删除模型失败")
		return
	}
	common.OKMsg(c, "删除模型成功", nil)
}

// 🎀 批量更新模型状态～
func BatchUpdateModels(c *gin.Context) {
	var req struct {
		IDs     []int `json:"ids" binding:"required"`
		Enabled bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if len(req.IDs) == 0 {
		common.Fail(c, common.CodeParamError, "请选择要操作的模型")
		return
	}
	if err := model.BatchUpdateModelsStatus(req.IDs, req.Enabled); err != nil {
		common.Fail(c, common.CodeServerError, "批量更新失败")
		return
	}
	common.OKMsg(c, "批量更新成功", nil)
}

// 🎀 批量删除模型～
func BatchDeleteModels(c *gin.Context) {
	var req struct {
		IDs []int `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if len(req.IDs) == 0 {
		common.Fail(c, common.CodeParamError, "请选择要删除的模型")
		return
	}
	if err := model.BatchDeleteModels(req.IDs); err != nil {
		common.Fail(c, common.CodeServerError, "批量删除失败")
		return
	}
	common.OKMsg(c, "批量删除成功", nil)
}
