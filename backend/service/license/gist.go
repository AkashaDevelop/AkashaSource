// ⚠️ REMOVABLE MODULE — 系统授权门禁
// 用 GitHub Gist 当"跨部署实例"的中心化绑定记录存储：一个 GitHub 账号只能绑定一个实例的指纹
package license

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const gistFileName = "akasha-license-bindings.json"

type gistBinding struct {
	Fingerprint string `json:"fingerprint"`
	BoundAt     int64  `json:"bound_at"`
}

type bindingsPayload struct {
	Bindings map[string]gistBinding `json:"bindings"`
}

type gistFile struct {
	Content string `json:"content"`
}

type gistResponse struct {
	Files map[string]gistFile `json:"files"`
}

// readBindings 读整份绑定表；Gist 里还没有内容（第一次使用）时返回空表，不当错误
func readBindings() (map[string]gistBinding, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/gists/"+getSecretGistId(), nil)
	req.Header.Set("Authorization", "Bearer "+getSecretGistToken())
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("读取 Gist 失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Gist 接口返回 %d: %s", resp.StatusCode, string(body))
	}

	var gr gistResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return nil, err
	}

	file, ok := gr.Files[gistFileName]
	if !ok || file.Content == "" {
		return map[string]gistBinding{}, nil
	}

	var payload bindingsPayload
	if err := json.Unmarshal([]byte(file.Content), &payload); err != nil {
		return nil, fmt.Errorf("解析 Gist 绑定记录失败: %w", err)
	}
	if payload.Bindings == nil {
		payload.Bindings = map[string]gistBinding{}
	}
	return payload.Bindings, nil
}

// writeBindings 把整份绑定表写回 Gist（已知限制：无原子 CAS，接受极小概率的读改写竞态）
func writeBindings(bindings map[string]gistBinding) error {
	content, err := json.Marshal(bindingsPayload{Bindings: bindings})
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]interface{}{
		"files": map[string]interface{}{
			gistFileName: map[string]string{"content": string(content)},
		},
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", "https://api.github.com/gists/"+getSecretGistId(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+getSecretGistToken())
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("写入 Gist 失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Gist 写入接口返回 %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
