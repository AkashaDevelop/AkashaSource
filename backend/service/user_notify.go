package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
)

func NotifyUser(userId int, userEmail string, userSetting dto.UserSetting, data dto.Notify) error {
	notifyType := userSetting.NotifyType
	if notifyType == "" {
		notifyType = dto.NotifyTypeEmail
	}

	canSend, err := model.CheckNotificationLimit(userId, data.Type)
	if err != nil {
		return err
	}
	if !canSend {
		return fmt.Errorf("notification limit exceeded for user %d", userId)
	}

	switch notifyType {
	case dto.NotifyTypeEmail:
		emailToUse := userSetting.NotificationEmail
		if emailToUse == "" {
			emailToUse = userEmail
		}
		if emailToUse == "" {
			return nil
		}
		return sendEmailNotify(emailToUse, data)
	case dto.NotifyTypeWebhook:
		if userSetting.WebhookUrl == "" {
			return nil
		}
		return SendWebhookNotify(userSetting.WebhookUrl, userSetting.WebhookSecret, data.Title, data.Content, data.Values)
	case dto.NotifyTypeBark:
		if userSetting.BarkUrl == "" {
			return nil
		}
		return sendBarkNotify(userSetting.BarkUrl, data)
	case dto.NotifyTypeGotify:
		if userSetting.GotifyUrl == "" || userSetting.GotifyToken == "" {
			return nil
		}
		return sendGotifyNotify(userSetting.GotifyUrl, userSetting.GotifyToken, userSetting.GotifyPriority, data)
	}
	return nil
}

func sendEmailNotify(userEmail string, data dto.Notify) error {
	content := data.Content
	for _, value := range data.Values {
		content = strings.Replace(content, dto.ContentValueParam, fmt.Sprintf("%v", value), 1)
	}
	return common.SendEmail(data.Title, userEmail, content)
}

func sendBarkNotify(barkURL string, data dto.Notify) error {
	content := data.Content
	for _, value := range data.Values {
		content = strings.Replace(content, dto.ContentValueParam, fmt.Sprintf("%v", value), 1)
	}

	finalURL := strings.ReplaceAll(barkURL, "{{title}}", url.QueryEscape(data.Title))
	finalURL = strings.ReplaceAll(finalURL, "{{content}}", url.QueryEscape(content))

	req, err := http.NewRequest(http.MethodGet, finalURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create bark request: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send bark request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("bark request failed with status code: %d", resp.StatusCode)
	}
	return nil
}

func sendGotifyNotify(gotifyUrl string, gotifyToken string, priority int, data dto.Notify) error {
	content := data.Content
	for _, value := range data.Values {
		content = strings.Replace(content, dto.ContentValueParam, fmt.Sprintf("%v", value), 1)
	}

	finalURL := strings.TrimSuffix(gotifyUrl, "/") + "/message?token=" + url.QueryEscape(gotifyToken)

	if priority < 0 || priority > 10 {
		priority = 5
	}

	type GotifyMessage struct {
		Title    string `json:"title"`
		Message  string `json:"message"`
		Priority int    `json:"priority"`
	}

	payload := GotifyMessage{Title: data.Title, Message: content, Priority: priority}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal gotify payload: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, finalURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return fmt.Errorf("failed to create gotify request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send gotify request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gotify request failed with status code: %d", resp.StatusCode)
	}
	return nil
}
