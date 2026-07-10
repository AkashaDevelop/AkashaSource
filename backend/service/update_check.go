package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"STfreApi/common"
	"STfreApi/model"
)

// VersionMeta Gist 中的版本元数据
type VersionMeta struct {
	LatestVersion    string          `json:"latest_version"`
	ReleaseURL       string          `json:"release_url"`
	ForceUpdate      bool            `json:"force_update"`
	ChangelogSummary string          `json:"changelog_summary"`
	PublishedAt      string          `json:"published_at"`
}

// rawVersionMeta 用于兼容 Gist 中 force_update 为字符串或布尔值的情况
type rawVersionMeta struct {
	LatestVersion    string          `json:"latest_version"`
	ReleaseURL       string          `json:"release_url"`
	ForceUpdate      json.RawMessage `json:"force_update"`
	ChangelogSummary string          `json:"changelog_summary"`
	PublishedAt      string          `json:"published_at"`
}

// UpdateInfo 检测结果（返回给前端）
type UpdateInfo struct {
	HasUpdate        bool   `json:"has_update"`
	LatestVersion    string `json:"latest_version"`
	CurrentVersion   string `json:"current_version"`
	ReleaseURL       string `json:"release_url"`
	ChangelogSummary string `json:"changelog_summary"`
	ForceUpdate      bool   `json:"force_update"`
	LastChecked      string `json:"last_checked"`
}

var (
	updateCheckMutex sync.Mutex
	cachedUpdateInfo *UpdateInfo
)

const gistID = "975ecfe6822d971e0a566c4115778e1b"
const gistFileName = "version.json"

// gistAPIResponse GitHub Gist API 响应结构
type gistAPIResponse struct {
	Files map[string]struct {
		Content string `json:"content"`
	} `json:"files"`
}

// FetchLatestVersion 从 GitHub Gist API 拉取版本元数据（不经过 CDN 缓存）
func FetchLatestVersion() (*VersionMeta, error) {
	url := "https://api.github.com/gists/" + gistID
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求版本信息失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("版本服务返回 %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	// 先解析 Gist API 响应，提取文件内容
	var gistResp gistAPIResponse
	if err := json.Unmarshal(body, &gistResp); err != nil {
		return nil, fmt.Errorf("解析 Gist 响应失败: %w", err)
	}
	fileData, ok := gistResp.Files[gistFileName]
	if !ok {
		return nil, fmt.Errorf("Gist 中未找到 %s 文件", gistFileName)
	}
	// 再解析版本元数据 JSON
	// 先用 rawVersionMeta 解析，兼容 force_update 为字符串或布尔值
	var raw rawVersionMeta
	if err := json.Unmarshal([]byte(fileData.Content), &raw); err != nil {
		return nil, fmt.Errorf("解析版本信息失败: %w", err)
	}
	meta := &VersionMeta{
		LatestVersion:    raw.LatestVersion,
		ReleaseURL:       raw.ReleaseURL,
		ChangelogSummary: raw.ChangelogSummary,
		PublishedAt:      raw.PublishedAt,
	}
	meta.ForceUpdate = parseBoolRaw(raw.ForceUpdate)
	return meta, nil
}

// parseBoolRaw 兼容 force_update 为 bool 或字符串 "true"/"false"
func parseBoolRaw(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	// 尝试直接解析为 bool
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		return b
	}
	// 尝试解析为字符串
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		b, _ = strconv.ParseBool(strings.TrimSpace(s))
		return b
	}
	return false
}

// CompareVersion 比较版本号，返回 -1 (current < target), 0 (相等), 1 (current > target)
// 支持 "v1.0.0"、"公测1.0.0"、"1.0.0" 等格式
func CompareVersion(current, target string) int {
	c := normalizeVersion(current)
	t := normalizeVersion(target)
	return compareSemver(c, t)
}

// normalizeVersion 去除中文前缀和 v 前缀，返回纯数字段
func normalizeVersion(s string) []int {
	s = strings.TrimSpace(s)
	// 去除中文前缀（如 "公测1.0.0" -> "1.0.0"）
	for i, c := range s {
		if c >= '0' && c <= '9' {
			s = s[i:]
			break
		}
	}
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimPrefix(s, "V")
	parts := strings.Split(s, ".")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		n := 0
		for _, c := range p {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			} else {
				break
			}
		}
		result = append(result, n)
	}
	return result
}

func compareSemver(a, b []int) int {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	for i := 0; i < maxLen; i++ {
		var av, bv int
		if i < len(a) {
			av = a[i]
		}
		if i < len(b) {
			bv = b[i]
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

// DoVersionCheck 执行一次完整的版本检测
func DoVersionCheck() (*UpdateInfo, error) {
	updateCheckMutex.Lock()
	defer updateCheckMutex.Unlock()

	common.OptionLock.RLock()
	enabled := common.OptionMap["version_check_enabled"] == "true"
	common.OptionLock.RUnlock()

	if !enabled {
		return nil, fmt.Errorf("版本检查已禁用")
	}

	meta, err := FetchLatestVersion()
	if err != nil {
		log.Printf("[版本检查] 拉取远端版本失败: %v", err)
		return nil, err
	}

	currentVersion := common.Version
	hasUpdate := CompareVersion(currentVersion, meta.LatestVersion) < 0

	info := &UpdateInfo{
		HasUpdate:        hasUpdate,
		LatestVersion:    meta.LatestVersion,
		CurrentVersion:   currentVersion,
		ReleaseURL:       meta.ReleaseURL,
		ChangelogSummary: meta.ChangelogSummary,
		ForceUpdate:      meta.ForceUpdate && hasUpdate,
		LastChecked:      time.Now().Format(time.RFC3339),
	}

	cachedUpdateInfo = info

	// 持久化检测结果摘要到 OptionMap
	resultJSON, _ := json.Marshal(info)
	common.DB.Where(model.Option{Key: "version_check_last_result"}).
		Assign(model.Option{Value: string(resultJSON)}).
		FirstOrCreate(&model.Option{Key: "version_check_last_result", Value: string(resultJSON)})
	common.UpdateOptionMap("version_check_last_result", string(resultJSON))

	log.Printf("[版本检查] 当前: %s, 最新: %s, 有更新: %v, 强制: %v",
		currentVersion, meta.LatestVersion, hasUpdate, info.ForceUpdate)

	return info, nil
}

// GetCachedUpdateInfo 返回缓存的检测结果（不触发网络请求）
func GetCachedUpdateInfo() *UpdateInfo {
	updateCheckMutex.Lock()
	defer updateCheckMutex.Unlock()

	if cachedUpdateInfo != nil {
		return cachedUpdateInfo
	}

	// 从 OptionMap 恢复上次结果
	common.OptionLock.RLock()
	resultStr := common.OptionMap["version_check_last_result"]
	common.OptionLock.RUnlock()

	if resultStr != "" {
		var info UpdateInfo
		if err := json.Unmarshal([]byte(resultStr), &info); err == nil {
			cachedUpdateInfo = &info
			return cachedUpdateInfo
		}
	}
	return nil
}

// IsVersionCheckEnabled 返回是否启用了版本检查
func IsVersionCheckEnabled() bool {
	common.OptionLock.RLock()
	defer common.OptionLock.RUnlock()
	enabled, ok := common.OptionMap["version_check_enabled"]
	if !ok {
		return true // 默认启用
	}
	return enabled == "true"
}

// GetVersionCheckIntervalHours 返回检查间隔（小时）
func GetVersionCheckIntervalHours() int {
	common.OptionLock.RLock()
	defer common.OptionLock.RUnlock()
	hoursStr := common.OptionMap["version_check_interval_hours"]
	if hoursStr == "" {
		return 24
	}
	h, err := strconv.Atoi(hoursStr)
	if err != nil || h <= 0 {
		return 24
	}
	return h
}
