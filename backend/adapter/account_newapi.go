package adapter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type NewApiAccountAdaptor struct {
	client *http.Client
}

func NewNewApiAccountAdaptor() *NewApiAccountAdaptor {
	return &NewApiAccountAdaptor{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type newApiResp struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type newApiUserData struct {
	Id          int     `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	Email       string  `json:"email"`
	Role        int     `json:"role"`
	Quota       int64   `json:"quota"`
	UsedQuota   int64   `json:"used_quota"`
}

type newApiCheckinData struct {
	Reward int64 `json:"reward"`
}

func (a *NewApiAccountAdaptor) buildAuthHeaders(accessToken string, userId ...int) map[string]string {
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Content-Type":  "application/json",
	}
	if len(userId) > 0 && userId[0] > 0 {
		uid := strconv.Itoa(userId[0])
		headers["New-Api-User"] = uid
		headers["Veloera-User"] = uid
		headers["voapi-user"] = uid
		headers["User-id"] = uid
	}
	return headers
}

func (a *NewApiAccountAdaptor) doRequest(method, url string, headers map[string]string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (a *NewApiAccountAdaptor) Checkin(baseUrl, accessToken string, platformUserId ...int) (*CheckinResult, error) {
	url := strings.TrimSuffix(baseUrl, "/") + "/api/user/checkin"
	headers := a.buildAuthHeaders(accessToken, platformUserId...)
	data, err := a.doRequest("POST", url, headers, nil)
	if err != nil {
		return &CheckinResult{Success: false, Message: err.Error()}, nil
	}
	var resp newApiResp
	if err := json.Unmarshal(data, &resp); err != nil {
		return &CheckinResult{Success: false, Message: string(data)}, nil
	}
	result := &CheckinResult{
		Success: resp.Success,
		Message: resp.Message,
	}
	if resp.Success {
		if m, ok := resp.Data.(map[string]interface{}); ok {
			if reward, ok := m["reward"]; ok {
				switch v := reward.(type) {
				case float64:
					result.Reward = strconv.FormatInt(int64(v), 10)
				case int64:
					result.Reward = strconv.FormatInt(v, 10)
				case string:
					result.Reward = v
				}
			}
		}
	}
	return result, nil
}

func (a *NewApiAccountAdaptor) GetBalance(baseUrl, accessToken string, platformUserId ...int) (*BalanceInfo, error) {
	url := strings.TrimSuffix(baseUrl, "/") + "/api/user/self"
	headers := a.buildAuthHeaders(accessToken, platformUserId...)
	data, err := a.doRequest("GET", url, headers, nil)
	if err != nil {
		return nil, err
	}
	var resp newApiResp
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(resp.Message)
	}
	userData, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response data")
	}
	quota := int64(0)
	usedQuota := int64(0)
	if v, ok := userData["quota"]; ok {
		switch val := v.(type) {
		case float64:
			quota = int64(val)
		case int64:
			quota = val
		}
	}
	if v, ok := userData["used_quota"]; ok {
		switch val := v.(type) {
		case float64:
			usedQuota = int64(val)
		case int64:
			usedQuota = val
		}
	}
	return &BalanceInfo{
		Balance: float64(quota) / 500000,
		Used:    float64(usedQuota) / 500000,
		Quota:   float64(quota+usedQuota) / 500000,
	}, nil
}

func (a *NewApiAccountAdaptor) GetUserInfo(baseUrl, accessToken string, platformUserId ...int) (*AccountInfo, error) {
	url := strings.TrimSuffix(baseUrl, "/") + "/api/user/self"
	headers := a.buildAuthHeaders(accessToken, platformUserId...)
	data, err := a.doRequest("GET", url, headers, nil)
	if err != nil {
		return nil, err
	}
	var resp newApiResp
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(resp.Message)
	}
	userData, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response data")
	}
	info := &AccountInfo{}
	if v, ok := userData["username"]; ok {
		info.Username, _ = v.(string)
	}
	if v, ok := userData["display_name"]; ok {
		info.DisplayName, _ = v.(string)
	}
	if v, ok := userData["email"]; ok {
		info.Email, _ = v.(string)
	}
	if v, ok := userData["role"]; ok {
		switch val := v.(type) {
		case float64:
			info.Role = int(val)
		case int:
			info.Role = val
		}
	}
	return info, nil
}

func (a *NewApiAccountAdaptor) Login(baseUrl, username, password string) (string, error) {
	url := strings.TrimSuffix(baseUrl, "/") + "/api/user/login"
	payload := fmt.Sprintf(`{"username":"%s","password":"%s"}`, username, password)
	headers := map[string]string{"Content-Type": "application/json"}
	data, err := a.doRequest("POST", url, headers, strings.NewReader(payload))
	if err != nil {
		return "", err
	}
	var resp newApiResp
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", err
	}
	if !resp.Success {
		return "", fmt.Errorf(resp.Message)
	}
	switch v := resp.Data.(type) {
	case string:
		return v, nil
	case map[string]interface{}:
		if token, ok := v["token"].(string); ok {
			return token, nil
		}
		if token, ok := v["access_token"].(string); ok {
			return token, nil
		}
	}
	return "", fmt.Errorf("no token in response")
}
