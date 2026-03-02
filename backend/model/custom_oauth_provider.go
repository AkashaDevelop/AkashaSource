package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"STfreApi/common"
)

var customOAuthSlugRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

var supportedAccessPolicyOps = map[string]struct{}{
	"eq":           {},
	"ne":           {},
	"gt":           {},
	"gte":          {},
	"lt":           {},
	"lte":          {},
	"in":           {},
	"not_in":       {},
	"contains":     {},
	"not_contains": {},
	"exists":       {},
	"not_exists":   {},
}

type accessPolicyPayload struct {
	Logic      string                `json:"logic"`
	Conditions []accessConditionItem `json:"conditions"`
	Groups     []accessPolicyPayload `json:"groups"`
}

type accessConditionItem struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value any    `json:"value"`
}

// CustomOAuthProvider stores configuration for custom OAuth providers.
type CustomOAuthProvider struct {
	Id                    int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Name                  string    `json:"name" gorm:"type:varchar(64);not null"`
	Slug                  string    `json:"slug" gorm:"type:varchar(64);uniqueIndex;not null"`
	Icon                  string    `json:"icon" gorm:"type:varchar(128);default:''"`
	Enabled               bool      `json:"enabled" gorm:"default:false"`
	ClientId              string    `json:"client_id" gorm:"type:varchar(256);not null"`
	ClientSecret          string    `json:"-" gorm:"type:varchar(512);not null"`
	AuthorizationEndpoint string    `json:"authorization_endpoint" gorm:"type:varchar(512);not null"`
	TokenEndpoint         string    `json:"token_endpoint" gorm:"type:varchar(512);not null"`
	UserInfoEndpoint      string    `json:"user_info_endpoint" gorm:"type:varchar(512);not null"`
	Scopes                string    `json:"scopes" gorm:"type:varchar(256);default:'openid profile email'"`
	UserIdField           string    `json:"user_id_field" gorm:"type:varchar(128);default:'sub'"`
	UsernameField         string    `json:"username_field" gorm:"type:varchar(128);default:'preferred_username'"`
	DisplayNameField      string    `json:"display_name_field" gorm:"type:varchar(128);default:'name'"`
	EmailField            string    `json:"email_field" gorm:"type:varchar(128);default:'email'"`
	WellKnown             string    `json:"well_known" gorm:"type:varchar(512)"`
	AuthStyle             int       `json:"auth_style" gorm:"default:0"`
	AccessPolicy          string    `json:"access_policy" gorm:"type:text"`
	AccessDeniedMessage   string    `json:"access_denied_message" gorm:"type:varchar(512)"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (CustomOAuthProvider) TableName() string {
	return "custom_oauth_providers"
}

func GetAllCustomOAuthProviders() ([]*CustomOAuthProvider, error) {
	var providers []*CustomOAuthProvider
	err := common.DB.Order("id asc").Find(&providers).Error
	return providers, err
}

func GetEnabledCustomOAuthProviders() ([]*CustomOAuthProvider, error) {
	var providers []*CustomOAuthProvider
	err := common.DB.Where("enabled = ?", true).Order("id asc").Find(&providers).Error
	return providers, err
}

func GetCustomOAuthProviderById(id int) (*CustomOAuthProvider, error) {
	var provider CustomOAuthProvider
	if err := common.DB.First(&provider, id).Error; err != nil {
		return nil, err
	}
	return &provider, nil
}

func GetCustomOAuthProviderBySlug(slug string) (*CustomOAuthProvider, error) {
	var provider CustomOAuthProvider
	if err := common.DB.Where("slug = ?", slug).First(&provider).Error; err != nil {
		return nil, err
	}
	return &provider, nil
}

func CreateCustomOAuthProvider(provider *CustomOAuthProvider) error {
	if err := validateCustomOAuthProvider(provider); err != nil {
		return err
	}
	return common.DB.Create(provider).Error
}

func UpdateCustomOAuthProvider(provider *CustomOAuthProvider) error {
	if err := validateCustomOAuthProvider(provider); err != nil {
		return err
	}
	return common.DB.Save(provider).Error
}

func DeleteCustomOAuthProvider(id int) error {
	return common.DB.Delete(&CustomOAuthProvider{}, id).Error
}

// IsSlugTaken returns true on DB errors (fail-closed).
func IsSlugTaken(slug string, excludeID int) bool {
	var count int64
	query := common.DB.Model(&CustomOAuthProvider{}).Where("slug = ?", strings.ToLower(strings.TrimSpace(slug)))
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return true
	}
	return count > 0
}

func validateCustomOAuthProvider(provider *CustomOAuthProvider) error {
	provider.Name = strings.TrimSpace(provider.Name)
	provider.Slug = strings.ToLower(strings.TrimSpace(provider.Slug))
	provider.ClientId = strings.TrimSpace(provider.ClientId)
	provider.AuthorizationEndpoint = strings.TrimSpace(provider.AuthorizationEndpoint)
	provider.TokenEndpoint = strings.TrimSpace(provider.TokenEndpoint)
	provider.UserInfoEndpoint = strings.TrimSpace(provider.UserInfoEndpoint)
	provider.WellKnown = strings.TrimSpace(provider.WellKnown)
	provider.AccessPolicy = strings.TrimSpace(provider.AccessPolicy)
	provider.AccessDeniedMessage = strings.TrimSpace(provider.AccessDeniedMessage)

	if provider.Name == "" {
		return errors.New("provider name is required")
	}
	if provider.Slug == "" {
		return errors.New("provider slug is required")
	}
	if !customOAuthSlugRegex.MatchString(provider.Slug) {
		return errors.New("provider slug must contain only lowercase letters, numbers, and hyphens")
	}
	if provider.ClientId == "" {
		return errors.New("client ID is required")
	}
	if strings.TrimSpace(provider.ClientSecret) == "" {
		return errors.New("client secret is required")
	}
	if provider.AuthorizationEndpoint == "" {
		return errors.New("authorization endpoint is required")
	}
	if provider.TokenEndpoint == "" {
		return errors.New("token endpoint is required")
	}
	if provider.UserInfoEndpoint == "" {
		return errors.New("user info endpoint is required")
	}
	if provider.Scopes == "" {
		provider.Scopes = "openid profile email"
	}
	if provider.UserIdField == "" {
		provider.UserIdField = "sub"
	}
	if provider.UsernameField == "" {
		provider.UsernameField = "preferred_username"
	}
	if provider.DisplayNameField == "" {
		provider.DisplayNameField = "name"
	}
	if provider.EmailField == "" {
		provider.EmailField = "email"
	}
	if provider.AuthStyle < 0 || provider.AuthStyle > 2 {
		return errors.New("auth_style must be 0, 1 or 2")
	}
	if provider.AccessPolicy != "" {
		var payload accessPolicyPayload
		if err := json.Unmarshal([]byte(provider.AccessPolicy), &payload); err != nil {
			return errors.New("access_policy must be valid JSON")
		}
		if err := validateAccessPolicyPayload(&payload); err != nil {
			return fmt.Errorf("access_policy is invalid: %w", err)
		}
	}
	return nil
}

func validateAccessPolicyPayload(policy *accessPolicyPayload) error {
	if policy == nil {
		return errors.New("policy is nil")
	}
	logic := strings.ToLower(strings.TrimSpace(policy.Logic))
	if logic == "" {
		logic = "and"
	}
	if logic != "and" && logic != "or" {
		return fmt.Errorf("unsupported logic: %s", logic)
	}
	if len(policy.Conditions) == 0 && len(policy.Groups) == 0 {
		return errors.New("policy requires at least one condition or group")
	}

	for i, cond := range policy.Conditions {
		field := strings.TrimSpace(cond.Field)
		if field == "" {
			return fmt.Errorf("condition[%d].field is required", i)
		}
		op := strings.ToLower(strings.TrimSpace(cond.Op))
		if _, ok := supportedAccessPolicyOps[op]; !ok {
			return fmt.Errorf("condition[%d].op is unsupported: %s", i, op)
		}
		if op == "in" || op == "not_in" {
			if _, ok := cond.Value.([]any); !ok {
				return fmt.Errorf("condition[%d].value must be an array for op %s", i, op)
			}
		}
	}

	for i := range policy.Groups {
		if err := validateAccessPolicyPayload(&policy.Groups[i]); err != nil {
			return fmt.Errorf("group[%d]: %w", i, err)
		}
	}
	return nil
}
