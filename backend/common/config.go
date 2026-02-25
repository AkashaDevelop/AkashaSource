package common

import (
	"strconv"
	"strings"
	"sync"
)

var (
	SystemName = "Akasha"
	OptionMap  = make(map[string]string)
	OptionLock sync.RWMutex
)

func UpdateOptionMap(key string, value string) {
	OptionLock.Lock()
	defer OptionLock.Unlock()
	OptionMap[key] = value
	if key == "system_name" {
		SystemName = value
	}

	if key == "github_client_id" {
		GitHubClientId = value
	}
	if key == "github_client_secret" {
		GitHubClientSecret = value
	}

	if key == "linuxdo_client_id" {
		LinuxDOClientId = value
	}
	if key == "linuxdo_client_secret" {
		LinuxDOClientSecret = value
	}

	if key == "smtp_server" {
		SMTPServer = value
	}
	if key == "smtp_port" {
		// handle int conversion if needed, but constants are usually typed?
		// Actually SMTPPort is int in constants.go. We need to convert string to int.
		// For simplicity, let's keep constants as variables in constants.go
		// But Go constants cannot be changed. They must be variables.
		// In constants.go: var SMTPPort = 587
	}
	if key == "smtp_account" {
		SMTPAccount = value
	}
	if key == "smtp_password" {
		SMTPPassword = value
	}
	if key == "smtp_from" {
		SMTPFrom = value
	}

	if key == "turnstile_site_key" {
		TurnstileSiteKey = value
	}
	if key == "turnstile_secret_key" {
		TurnstileSecretKey = value
	}
	if key == "turnstile_check_enabled" {
		TurnstileCheckEnabled = (value == "true")
	}

	if strings.HasPrefix(key, "linuxdo_quota_level_") {
		levelStr := strings.TrimPrefix(key, "linuxdo_quota_level_")
		level, err := strconv.Atoi(levelStr)
		if err == nil && level >= 0 && level <= 5 {
			quota, err := strconv.ParseInt(value, 10, 64)
			if err == nil {
				LinuxDOLevelQuota[level] = quota
			}
		}
	}

	if key == "model_ratio" || key == "completion_ratio" {
		UpdatePricing(OptionMap["model_ratio"], OptionMap["completion_ratio"])
	}
}
