package dto

const (
	NotifyTypeEmail   = "email"
	NotifyTypeWebhook = "webhook"
	NotifyTypeBark    = "bark"
	NotifyTypeGotify  = "gotify"

	NotifyTypeChannelUpdate = "channel_update"
	NotifyTypeSystem        = "system"
	NotifyTypeQuota         = "quota"

	ContentValueParam = "{}"
)

type Notify struct {
	Type    string        `json:"type"`
	Title   string        `json:"title"`
	Content string        `json:"content"`
	Values  []interface{} `json:"values,omitempty"`
}

func NewNotify(t, title, content string, values []interface{}) Notify {
	return Notify{Type: t, Title: title, Content: content, Values: values}
}

type UserSetting struct {
	NotifyType                       string `json:"notify_type"`
	NotificationEmail                string `json:"notification_email"`
	WebhookUrl                       string `json:"webhook_url"`
	WebhookSecret                    string `json:"webhook_secret"`
	BarkUrl                          string `json:"bark_url"`
	GotifyUrl                        string `json:"gotify_url"`
	GotifyToken                      string `json:"gotify_token"`
	GotifyPriority                   int    `json:"gotify_priority"`
	UpstreamModelUpdateNotifyEnabled bool   `json:"upstream_model_update_notify_enabled"`
}
