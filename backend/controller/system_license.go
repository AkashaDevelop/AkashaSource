// ⚠️ REMOVABLE MODULE — 系统授权门禁
// 瘦 controller，只做参数解析，具体逻辑都在 service/license 包里
package controller

import (
	"STfreApi/common"
	"STfreApi/service/license"

	"github.com/gin-gonic/gin"
)

// GetLicenseStatus 查当前系统授权状态（rootGroup，需要登录态）
func GetLicenseStatus(c *gin.Context) {
	common.OK(c, license.GetStatus())
}

// StartDeviceFlow 发起 Device Flow，返回设备码供前端展示（rootGroup）
func StartDeviceFlow(c *gin.Context) {
	info, err := license.RequestDeviceFlow()
	if err != nil {
		common.Fail(c, common.CodeForbidden, err.Error())
		return
	}
	common.OK(c, info)
}

// PollDeviceFlow 轮询 GitHub 换取 token 并完成授权绑定（rootGroup）
func PollDeviceFlow(c *gin.Context) {
	var req struct {
		DeviceCode string `json:"device_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "缺少 device_code 参数")
		return
	}

	completed, err := license.PollDeviceFlow(req.DeviceCode)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}
	common.OK(c, gin.H{
		"completed": completed,
		"message":   ternary(completed, "授权成功", "等待用户完成授权..."),
	})
}

// UnbindLicense 解绑当前实例的授权（rootGroup，需要登录态）
func UnbindLicense(c *gin.Context) {
	if err := license.Unbind(); err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}
	common.OKMsg(c, "已解绑", nil)
}

// ternary Go 没有三元运算符，写个小工具函数
func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
