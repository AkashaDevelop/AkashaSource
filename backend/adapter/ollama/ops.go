package ollama

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OllamaPullRequest struct {
	Name   string `json:"name"`
	Stream bool   `json:"stream,omitempty"`
}

type OllamaPullResponse struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
}

type OllamaDeleteRequest struct {
	Name string `json:"name"`
}

// PullOllamaModel 拉取 Ollama 模型（非流式）
func PullOllamaModel(baseURL, apiKey, modelName string) error {
	url := fmt.Sprintf("%s/api/pull", strings.TrimRight(baseURL, "/"))
	payload := OllamaPullRequest{Name: modelName, Stream: false}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %v", err)
	}

	client := &http.Client{Timeout: 30 * time.Minute}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("拉取模型失败 %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

// PullOllamaModelStream 流式拉取 Ollama 模型（支持进度回调）
func PullOllamaModelStream(baseURL, apiKey, modelName string, progressCallback func(OllamaPullResponse)) error {
	url := fmt.Sprintf("%s/api/pull", strings.TrimRight(baseURL, "/"))
	payload := OllamaPullRequest{Name: modelName, Stream: true}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %v", err)
	}

	client := &http.Client{Timeout: time.Hour}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("拉取模型失败 %d: %s", resp.StatusCode, string(raw))
	}

	scanner := bufio.NewScanner(resp.Body)
	successful := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var pr OllamaPullResponse
		if err := json.Unmarshal([]byte(line), &pr); err != nil {
			continue
		}
		if progressCallback != nil {
			progressCallback(pr)
		}
		if strings.EqualFold(pr.Status, "error") {
			return fmt.Errorf("拉取模型失败: %s", line)
		}
		if strings.EqualFold(pr.Status, "success") {
			successful = true
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取流式响应失败: %v", err)
	}
	if !successful {
		return fmt.Errorf("拉取模型未完成: 未收到成功状态")
	}
	return nil
}

// DeleteOllamaModel 删除 Ollama 模型
func DeleteOllamaModel(baseURL, apiKey, modelName string) error {
	url := fmt.Sprintf("%s/api/delete", strings.TrimRight(baseURL, "/"))
	payload := OllamaDeleteRequest{Name: modelName}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %v", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("删除模型失败 %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

// FetchOllamaVersion 获取 Ollama 版本
func FetchOllamaVersion(baseURL, apiKey string) (string, error) {
	trimmedBase := strings.TrimRight(baseURL, "/")
	if trimmedBase == "" {
		return "", fmt.Errorf("baseURL 为空")
	}

	url := fmt.Sprintf("%s/api/version", trimmedBase)
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("查询版本失败 %d: %s", resp.StatusCode, string(body))
	}

	var versionResp struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(body, &versionResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}
	if strings.TrimSpace(versionResp.Version) == "" {
		return "", fmt.Errorf("未返回版本信息")
	}
	return versionResp.Version, nil
}
