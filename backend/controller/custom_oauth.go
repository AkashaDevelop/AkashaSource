package controller

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"STfreApi/common"
	"STfreApi/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CustomOAuthProviderResponse struct {
	Id                    int    `json:"id"`
	Name                  string `json:"name"`
	Slug                  string `json:"slug"`
	Icon                  string `json:"icon"`
	Enabled               bool   `json:"enabled"`
	ClientId              string `json:"client_id"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"user_info_endpoint"`
	Scopes                string `json:"scopes"`
	UserIdField           string `json:"user_id_field"`
	UsernameField         string `json:"username_field"`
	DisplayNameField      string `json:"display_name_field"`
	EmailField            string `json:"email_field"`
	WellKnown             string `json:"well_known"`
	AuthStyle             int    `json:"auth_style"`
	AccessPolicy          string `json:"access_policy"`
	AccessDeniedMessage   string `json:"access_denied_message"`
}

type UserOAuthBindingResponse struct {
	ProviderId     int    `json:"provider_id"`
	ProviderName   string `json:"provider_name"`
	ProviderSlug   string `json:"provider_slug"`
	ProviderIcon   string `json:"provider_icon"`
	ProviderUserId string `json:"provider_user_id"`
}

type FetchCustomOAuthDiscoveryRequest struct {
	WellKnownURL string `json:"well_known_url"`
	IssuerURL    string `json:"issuer_url"`
}

type CreateCustomOAuthProviderRequest struct {
	Name                  string `json:"name" binding:"required"`
	Slug                  string `json:"slug" binding:"required"`
	Icon                  string `json:"icon"`
	Enabled               bool   `json:"enabled"`
	ClientId              string `json:"client_id" binding:"required"`
	ClientSecret          string `json:"client_secret" binding:"required"`
	AuthorizationEndpoint string `json:"authorization_endpoint" binding:"required"`
	TokenEndpoint         string `json:"token_endpoint" binding:"required"`
	UserInfoEndpoint      string `json:"user_info_endpoint" binding:"required"`
	Scopes                string `json:"scopes"`
	UserIdField           string `json:"user_id_field"`
	UsernameField         string `json:"username_field"`
	DisplayNameField      string `json:"display_name_field"`
	EmailField            string `json:"email_field"`
	WellKnown             string `json:"well_known"`
	AuthStyle             int    `json:"auth_style"`
	AccessPolicy          string `json:"access_policy"`
	AccessDeniedMessage   string `json:"access_denied_message"`
}

type UpdateCustomOAuthProviderRequest struct {
	Name                  string  `json:"name"`
	Slug                  string  `json:"slug"`
	Icon                  *string `json:"icon"`
	Enabled               *bool   `json:"enabled"`
	ClientId              string  `json:"client_id"`
	ClientSecret          string  `json:"client_secret"`
	AuthorizationEndpoint string  `json:"authorization_endpoint"`
	TokenEndpoint         string  `json:"token_endpoint"`
	UserInfoEndpoint      string  `json:"user_info_endpoint"`
	Scopes                string  `json:"scopes"`
	UserIdField           string  `json:"user_id_field"`
	UsernameField         string  `json:"username_field"`
	DisplayNameField      string  `json:"display_name_field"`
	EmailField            string  `json:"email_field"`
	WellKnown             *string `json:"well_known"`
	AuthStyle             *int    `json:"auth_style"`
	AccessPolicy          *string `json:"access_policy"`
	AccessDeniedMessage   *string `json:"access_denied_message"`
}

func toCustomOAuthProviderResponse(p *model.CustomOAuthProvider) *CustomOAuthProviderResponse {
	return &CustomOAuthProviderResponse{
		Id:                    p.Id,
		Name:                  p.Name,
		Slug:                  p.Slug,
		Icon:                  p.Icon,
		Enabled:               p.Enabled,
		ClientId:              p.ClientId,
		AuthorizationEndpoint: p.AuthorizationEndpoint,
		TokenEndpoint:         p.TokenEndpoint,
		UserInfoEndpoint:      p.UserInfoEndpoint,
		Scopes:                p.Scopes,
		UserIdField:           p.UserIdField,
		UsernameField:         p.UsernameField,
		DisplayNameField:      p.DisplayNameField,
		EmailField:            p.EmailField,
		WellKnown:             p.WellKnown,
		AuthStyle:             p.AuthStyle,
		AccessPolicy:          p.AccessPolicy,
		AccessDeniedMessage:   p.AccessDeniedMessage,
	}
}

func isBuiltinOAuthSlug(slug string) bool {
	switch strings.ToLower(strings.TrimSpace(slug)) {
	case "github", "linuxdo", "discord", "oidc", "telegram", "wechat":
		return true
	default:
		return false
	}
}

func FetchCustomOAuthDiscovery(c *gin.Context) {
	var req FetchCustomOAuthDiscoveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "无效的请求参数: "+err.Error())
		return
	}

	wellKnownURL := strings.TrimSpace(req.WellKnownURL)
	issuerURL := strings.TrimSpace(req.IssuerURL)
	if wellKnownURL == "" && issuerURL == "" {
		common.Fail(c, common.CodeParamError, "请先填写 Discovery URL 或 Issuer URL")
		return
	}

	targetURL := wellKnownURL
	if targetURL == "" {
		targetURL = strings.TrimRight(issuerURL, "/") + "/.well-known/openid-configuration"
	}
	targetURL = strings.TrimSpace(targetURL)

	parsedURL, err := url.Parse(targetURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		common.Fail(c, common.CodeParamError, "Discovery URL 无效，仅支持 http/https")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		common.Fail(c, common.CodeServerError, "创建 Discovery 请求失败: "+err.Error())
		return
	}
	httpReq.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(httpReq)
	if err != nil {
		common.Fail(c, common.CodeServerError, "获取 Discovery 配置失败: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		common.Fail(c, common.CodeServerError, "获取 Discovery 配置失败: "+msg)
		return
	}

	var discovery map[string]any
	if err = json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		common.Fail(c, common.CodeServerError, "解析 Discovery 配置失败: "+err.Error())
		return
	}

	common.OK(c, gin.H{"well_known_url": targetURL, "discovery": discovery})
}

func GetCustomOAuthProviders(c *gin.Context) {
	providers, err := model.GetAllCustomOAuthProviders()
	if err != nil {
		common.Fail(c, common.CodeServerError, "获取自定义 OAuth 提供商失败")
		return
	}
	resp := make([]*CustomOAuthProviderResponse, 0, len(providers))
	for _, p := range providers {
		resp = append(resp, toCustomOAuthProviderResponse(p))
	}
	common.OK(c, resp)
}

func GetCustomOAuthProvider(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "无效的 ID")
		return
	}
	provider, err := model.GetCustomOAuthProviderById(id)
	if err != nil {
		if errorsIsNotFound(err) {
			common.Fail(c, common.CodeNotFound, "未找到该 OAuth 提供商")
			return
		}
		common.Fail(c, common.CodeServerError, "获取自定义 OAuth 提供商失败")
		return
	}
	common.OK(c, toCustomOAuthProviderResponse(provider))
}

func CreateCustomOAuthProvider(c *gin.Context) {
	var req CreateCustomOAuthProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "无效的请求参数: "+err.Error())
		return
	}
	if model.IsSlugTaken(req.Slug, 0) {
		common.Fail(c, common.CodeConflict, "该 Slug 已被使用")
		return
	}
	if isBuiltinOAuthSlug(req.Slug) {
		common.Fail(c, common.CodeConflict, "该 Slug 与内置 OAuth 提供商冲突")
		return
	}

	provider := &model.CustomOAuthProvider{
		Name:                  req.Name,
		Slug:                  req.Slug,
		Icon:                  req.Icon,
		Enabled:               req.Enabled,
		ClientId:              req.ClientId,
		ClientSecret:          req.ClientSecret,
		AuthorizationEndpoint: req.AuthorizationEndpoint,
		TokenEndpoint:         req.TokenEndpoint,
		UserInfoEndpoint:      req.UserInfoEndpoint,
		Scopes:                req.Scopes,
		UserIdField:           req.UserIdField,
		UsernameField:         req.UsernameField,
		DisplayNameField:      req.DisplayNameField,
		EmailField:            req.EmailField,
		WellKnown:             req.WellKnown,
		AuthStyle:             req.AuthStyle,
		AccessPolicy:          req.AccessPolicy,
		AccessDeniedMessage:   req.AccessDeniedMessage,
	}

	if err := model.CreateCustomOAuthProvider(provider); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	common.OKMsg(c, "创建成功", toCustomOAuthProviderResponse(provider))
}

func UpdateCustomOAuthProvider(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "无效的 ID")
		return
	}

	var req UpdateCustomOAuthProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "无效的请求参数: "+err.Error())
		return
	}

	provider, err := model.GetCustomOAuthProviderById(id)
	if err != nil {
		common.Fail(c, common.CodeNotFound, "未找到该 OAuth 提供商")
		return
	}

	if req.Slug != "" && req.Slug != provider.Slug {
		if model.IsSlugTaken(req.Slug, id) {
			common.Fail(c, common.CodeConflict, "该 Slug 已被使用")
			return
		}
		if isBuiltinOAuthSlug(req.Slug) {
			common.Fail(c, common.CodeConflict, "该 Slug 与内置 OAuth 提供商冲突")
			return
		}
	}

	if req.Name != "" {
		provider.Name = req.Name
	}
	if req.Slug != "" {
		provider.Slug = req.Slug
	}
	if req.Icon != nil {
		provider.Icon = *req.Icon
	}
	if req.Enabled != nil {
		provider.Enabled = *req.Enabled
	}
	if req.ClientId != "" {
		provider.ClientId = req.ClientId
	}
	if req.ClientSecret != "" {
		provider.ClientSecret = req.ClientSecret
	}
	if req.AuthorizationEndpoint != "" {
		provider.AuthorizationEndpoint = req.AuthorizationEndpoint
	}
	if req.TokenEndpoint != "" {
		provider.TokenEndpoint = req.TokenEndpoint
	}
	if req.UserInfoEndpoint != "" {
		provider.UserInfoEndpoint = req.UserInfoEndpoint
	}
	if req.Scopes != "" {
		provider.Scopes = req.Scopes
	}
	if req.UserIdField != "" {
		provider.UserIdField = req.UserIdField
	}
	if req.UsernameField != "" {
		provider.UsernameField = req.UsernameField
	}
	if req.DisplayNameField != "" {
		provider.DisplayNameField = req.DisplayNameField
	}
	if req.EmailField != "" {
		provider.EmailField = req.EmailField
	}
	if req.WellKnown != nil {
		provider.WellKnown = *req.WellKnown
	}
	if req.AuthStyle != nil {
		provider.AuthStyle = *req.AuthStyle
	}
	if req.AccessPolicy != nil {
		provider.AccessPolicy = *req.AccessPolicy
	}
	if req.AccessDeniedMessage != nil {
		provider.AccessDeniedMessage = *req.AccessDeniedMessage
	}

	if err := model.UpdateCustomOAuthProvider(provider); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	common.OKMsg(c, "更新成功", toCustomOAuthProviderResponse(provider))
}

func DeleteCustomOAuthProvider(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "无效的 ID")
		return
	}

	if _, err := model.GetCustomOAuthProviderById(id); err != nil {
		common.Fail(c, common.CodeNotFound, "未找到该 OAuth 提供商")
		return
	}

	count, err := model.GetBindingCountByProviderId(id)
	if err != nil {
		common.Fail(c, common.CodeServerError, "检查用户绑定失败")
		return
	}
	if count > 0 {
		common.Fail(c, common.CodeConflict, "该 OAuth 提供商还有用户绑定，无法删除")
		return
	}

	if err := model.DeleteCustomOAuthProvider(id); err != nil {
		common.Fail(c, common.CodeServerError, "删除自定义 OAuth 提供商失败")
		return
	}
	common.OKMsg(c, "删除成功", nil)
}

func buildUserOAuthBindingsResponse(userId int) ([]UserOAuthBindingResponse, error) {
	bindings, err := model.GetUserOAuthBindingsByUserId(userId)
	if err != nil {
		return nil, err
	}
	result := make([]UserOAuthBindingResponse, 0, len(bindings))
	for _, binding := range bindings {
		provider, err := model.GetCustomOAuthProviderById(binding.ProviderId)
		if err != nil {
			continue
		}
		result = append(result, UserOAuthBindingResponse{
			ProviderId:     binding.ProviderId,
			ProviderName:   provider.Name,
			ProviderSlug:   provider.Slug,
			ProviderIcon:   provider.Icon,
			ProviderUserId: binding.ProviderUserId,
		})
	}
	return result, nil
}

func GetUserOAuthBindings(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}
	resp, err := buildUserOAuthBindingsResponse(userId)
	if err != nil {
		common.Fail(c, common.CodeServerError, "获取 OAuth 绑定失败")
		return
	}
	common.OK(c, resp)
}

func GetUserOAuthBindingsByAdmin(c *gin.Context) {
	targetUserId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "invalid user id")
		return
	}

	var target model.User
	if err := common.DB.First(&target, targetUserId).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}

	myRole := c.GetInt("role")
	if myRole <= target.Role && myRole != model.RoleRoot {
		common.Fail(c, common.CodeForbidden, "no permission")
		return
	}

	resp, err := buildUserOAuthBindingsResponse(targetUserId)
	if err != nil {
		common.Fail(c, common.CodeServerError, "获取 OAuth 绑定失败")
		return
	}
	common.OK(c, resp)
}

func UnbindCustomOAuth(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}
	providerId, err := strconv.Atoi(c.Param("provider_id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "无效的提供商 ID")
		return
	}
	if err := model.DeleteUserOAuthBinding(userId, providerId); err != nil {
		common.Fail(c, common.CodeServerError, "解绑失败")
		return
	}
	common.OKMsg(c, "解绑成功", nil)
}

func UnbindCustomOAuthByAdmin(c *gin.Context) {
	targetUserId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "invalid user id")
		return
	}
	providerId, err := strconv.Atoi(c.Param("provider_id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "invalid provider id")
		return
	}

	var target model.User
	if err := common.DB.First(&target, targetUserId).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}
	myRole := c.GetInt("role")
	if myRole <= target.Role && myRole != model.RoleRoot {
		common.Fail(c, common.CodeForbidden, "no permission")
		return
	}

	if err := model.DeleteUserOAuthBinding(targetUserId, providerId); err != nil {
		common.Fail(c, common.CodeServerError, "解绑失败")
		return
	}
	common.OKMsg(c, "success", nil)
}

func errorsIsNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}
