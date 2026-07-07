// ⚠️ REMOVABLE MODULE — 系统授权门禁
// 瘦 controller，只做参数解析和跳转，具体逻辑都在 service/license 包里
package controller

import (
	"net/http"
	"net/url"

	"STfreApi/common"
	"STfreApi/service/license"

	"github.com/gin-gonic/gin"
)

// GetLicenseStatus 查当前系统授权状态（rootGroup，需要登录态）
func GetLicenseStatus(c *gin.Context) {
	common.OK(c, license.GetStatus())
}

// StartGitHubAuth 跳转去 GitHub 授权页（匿名可达——前端是纯 Bearer Token 鉴权，
// 整页跳转天然带不上 Authorization 头；这个接口本身只是拼链接重定向，不涉及任何
// 本地敏感操作，真正的权限判断在拿到 GitHub 身份后的 callback 阶段完成）
func StartGitHubAuth(c *gin.Context) {
	url, err := license.BuildAuthorizeURL()
	if err != nil {
		common.Fail(c, common.CodeForbidden, err.Error())
		return
	}
	c.Redirect(http.StatusFound, url)
}

// GitHubAuthCallback GitHub 授权回调（匿名可达，靠 state 防 CSRF）
func GitHubAuthCallback(c *gin.Context) {
	err := license.HandleCallback(c.Query("state"), c.Query("code"))
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/security?license=error&reason="+url.QueryEscape(err.Error()))
		return
	}
	c.Redirect(http.StatusFound, "/admin/security?license=success")
}

// UnbindLicense 解绑当前实例的授权（rootGroup，需要登录态）
func UnbindLicense(c *gin.Context) {
	if err := license.Unbind(); err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}
	common.OKMsg(c, "已解绑", nil)
}
