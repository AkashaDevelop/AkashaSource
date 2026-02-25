package model

type MidjourneyTask struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId      int    `json:"user_id" gorm:"index"`
	Code        int    `json:"code"`
	Action      string `json:"action"`
	MjId        string `json:"mj_id" gorm:"index"`
	Prompt      string `json:"prompt"`
	PromptEn    string `json:"prompt_en"`
	Description string `json:"description"`
	State       string `json:"state"`
	SubmitTime  int64  `json:"submit_time"`
	StartTime   int64  `json:"start_time"`
	FinishTime  int64  `json:"finish_time"`
	ImageUrl    string `json:"image_url"`
	Status      string `json:"status"` // SUBMITTED, PROCESSING, SUCCESS, FAILURE
	Progress    string `json:"progress"`
	FailReason  string `json:"fail_reason"`
	ChannelId   int    `json:"channel_id"`
	Quota       int64  `json:"quota"`
	CreatedAt   int64  `json:"created_at"`
}
