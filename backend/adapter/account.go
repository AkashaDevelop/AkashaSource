package adapter

type CheckinResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Reward  string `json:"reward,omitempty"`
}

type BalanceInfo struct {
	Balance float64 `json:"balance"`
	Used    float64 `json:"used"`
	Quota   float64 `json:"quota"`
}

type AccountInfo struct {
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name,omitempty"`
	Email       string  `json:"email,omitempty"`
	Role        int     `json:"role,omitempty"`
	Balance     float64 `json:"balance,omitempty"`
}

type AccountAdaptor interface {
	Checkin(baseUrl, accessToken string, platformUserId ...int) (*CheckinResult, error)
	GetBalance(baseUrl, accessToken string, platformUserId ...int) (*BalanceInfo, error)
	GetUserInfo(baseUrl, accessToken string, platformUserId ...int) (*AccountInfo, error)
	Login(baseUrl, username, password string) (string, error)
}
