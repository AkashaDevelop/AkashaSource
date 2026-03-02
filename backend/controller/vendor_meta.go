package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"strconv"

	"github.com/gin-gonic/gin"
)

func parsePageParams(c *gin.Context) (page int, pageSize int, offset int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	offset = (page - 1) * pageSize
	return
}

func GetAllVendors(c *gin.Context) {
	page, pageSize, offset := parsePageParams(c)
	vendors, err := model.GetAllVendors(offset, pageSize)
	if err != nil {
		common.Fail(c, common.CodeServerError, "获取供应商失败")
		return
	}
	var total int64
	if err = common.DB.Model(&model.Vendor{}).Count(&total).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取供应商总数失败")
		return
	}
	common.OK(c, gin.H{
		"items":     vendors,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func SearchVendors(c *gin.Context) {
	keyword := c.Query("keyword")
	page, pageSize, offset := parsePageParams(c)
	vendors, total, err := model.SearchVendors(keyword, offset, pageSize)
	if err != nil {
		common.Fail(c, common.CodeServerError, "搜索供应商失败")
		return
	}
	common.OK(c, gin.H{
		"items":     vendors,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func GetVendorMeta(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "供应商 ID 非法")
		return
	}
	vendor, err := model.GetVendorByID(id)
	if err != nil {
		common.Fail(c, common.CodeNotFound, "供应商不存在")
		return
	}
	common.OK(c, vendor)
}

func CreateVendorMeta(c *gin.Context) {
	var v model.Vendor
	if err := c.ShouldBindJSON(&v); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if v.Name == "" {
		common.Fail(c, common.CodeParamError, "供应商名称不能为空")
		return
	}
	if dup, err := model.IsVendorNameDuplicated(0, v.Name); err != nil {
		common.Fail(c, common.CodeServerError, "校验供应商名称失败")
		return
	} else if dup {
		common.Fail(c, common.CodeConflict, "供应商名称已存在")
		return
	}
	if err := v.Insert(); err != nil {
		common.Fail(c, common.CodeServerError, "创建供应商失败")
		return
	}
	common.OKMsg(c, "创建供应商成功", v)
}

func UpdateVendorMeta(c *gin.Context) {
	var v model.Vendor
	if err := c.ShouldBindJSON(&v); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if v.Id == 0 {
		common.Fail(c, common.CodeParamError, "缺少供应商 ID")
		return
	}
	if dup, err := model.IsVendorNameDuplicated(v.Id, v.Name); err != nil {
		common.Fail(c, common.CodeServerError, "校验供应商名称失败")
		return
	} else if dup {
		common.Fail(c, common.CodeConflict, "供应商名称已存在")
		return
	}
	if err := v.Update(); err != nil {
		common.Fail(c, common.CodeServerError, "更新供应商失败")
		return
	}
	common.OKMsg(c, "更新供应商成功", nil)
}

func DeleteVendorMeta(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "供应商 ID 非法")
		return
	}
	v := model.Vendor{Id: id}
	if err = v.Delete(); err != nil {
		common.Fail(c, common.CodeServerError, "删除供应商失败")
		return
	}
	common.OKMsg(c, "删除供应商成功", nil)
}
