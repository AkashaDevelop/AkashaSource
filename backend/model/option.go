package model

import "STfreApi/common"

type Option struct {
	Key   string `json:"key" gorm:"primaryKey"`
	Value string `json:"value"`
}

const (
	OptionKeySystemName                    = "system_name"
	OptionKeyLogoUrl                       = "logo_url"
	OptionKeyFooterHtml                    = "footer_html"
	OptionKeyNotice                        = "notice"
	OptionKeyAbout                         = "about"
	OptionKeySystemUrl                     = "system_url"
	OptionKeyPrice                         = "price"
	OptionKeyMinTopup                      = "min_topup"
	OptionKeyTopupLink                     = "topup_link"
	OptionKeyChatLink                      = "chat_link"
	OptionKeyChatLink2                     = "chat_link2"
	OptionKeyEmailDomainRestrictionEnabled = "email_domain_restriction_enabled"
	OptionKeyEmailDomainWhitelist          = "email_domain_whitelist"

	// Pricing Options
	OptionKeyModelRatio      = "model_ratio"
	OptionKeyCompletionRatio = "completion_ratio"
)

func InitOptions() {
	common.DB.AutoMigrate(&Option{})
	common.DB.AutoMigrate(&User{})
	common.DB.AutoMigrate(&Token{})
	common.DB.AutoMigrate(&Channel{})
	common.DB.AutoMigrate(&Log{})
	common.DB.AutoMigrate(&Redemption{})
	common.DB.AutoMigrate(&MidjourneyTask{})

	// Load options from DB
	var options []Option
	common.DB.Find(&options)
	for _, option := range options {
		common.UpdateOptionMap(option.Key, option.Value)
	}

	// Initialize pricing if not present in DB
	if _, ok := common.OptionMap[OptionKeyModelRatio]; !ok {
		common.UpdatePricing("", "") // Load defaults
		// Save defaults to DB
		// Note: We should check if they exist before creating to avoid duplicates if Find failed but DB has them
		// But since we did Find above, if map is empty, DB is likely empty or key missing.
		// Use FirstOrCreate or similar

		mrJSON := common.ModelRatio2JSONString()
		crJSON := common.CompletionRatio2JSONString()

		common.DB.FirstOrCreate(&Option{Key: OptionKeyModelRatio, Value: mrJSON}, Option{Key: OptionKeyModelRatio})
		common.DB.FirstOrCreate(&Option{Key: OptionKeyCompletionRatio, Value: crJSON}, Option{Key: OptionKeyCompletionRatio})
	}
}
