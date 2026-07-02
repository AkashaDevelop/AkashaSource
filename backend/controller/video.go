package controller

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"

	"STfreApi/common"
	"STfreApi/model"

	"github.com/gin-gonic/gin"
)

// RelayTask 兼容 new-api 任务入口，当前复用现有 Relay。
func RelayTask(c *gin.Context) {
	Relay(c)
}

// RelayTaskFetch 兼容 new-api 的视频任务查询接口。
func RelayTaskFetch(c *gin.Context) {
	taskID := strings.TrimSpace(c.Param("task_id"))
	if taskID == "" {
		common.Fail(c, common.CodeParamError, "task_id 不能为空")
		return
	}

	if task, ok := findMidjourneyTask(taskID); ok {
		status := normalizeTaskStatus(task.Status)
		url := strings.TrimSpace(task.ImageUrl)
		payload := gin.H{
			"id":     taskID,
			"status": status,
		}
		if url != "" {
			payload["result_url"] = url
		}
		common.OK(c, payload)
		return
	}

	if task, ok := findSunoTask(taskID); ok {
		status := normalizeTaskStatus(task.Status)
		url := extractResultURL(task.Result)
		payload := gin.H{
			"id":     taskID,
			"status": status,
		}
		if url != "" {
			payload["result_url"] = url
		}
		common.OK(c, payload)
		return
	}

	common.Fail(c, common.CodeNotFound, "任务不存在")
}

// VideoProxy 兼容 /v1/videos/:task_id/content，按任务结果 URL 反向代理内容。
func VideoProxy(c *gin.Context) {
	taskID := strings.TrimSpace(c.Param("task_id"))
	if taskID == "" {
		common.Fail(c, common.CodeParamError, "task_id 不能为空")
		return
	}

	resultURL := ""
	if task, ok := findMidjourneyTask(taskID); ok {
		resultURL = strings.TrimSpace(task.ImageUrl)
	}
	if resultURL == "" {
		if task, ok := findSunoTask(taskID); ok {
			resultURL = extractResultURL(task.Result)
		}
	}
	if resultURL == "" {
		common.Fail(c, common.CodeNotFound, "任务结果不存在")
		return
	}

	if strings.HasPrefix(resultURL, "data:") {
		writeDataURL(c, resultURL)
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, resultURL, nil)
	if err != nil {
		common.Fail(c, common.CodeServerError, "创建代理请求失败")
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		common.Fail(c, common.CodeServerError, "拉取任务结果失败")
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, v := range values {
			c.Writer.Header().Add(key, v)
		}
	}
	c.Status(resp.StatusCode)
	_, _ = io.Copy(c.Writer, resp.Body)
}

func findMidjourneyTask(taskID string) (*model.MidjourneyTask, bool) {
	var t model.MidjourneyTask
	if err := common.DB.Where("mj_id = ?", taskID).First(&t).Error; err == nil {
		return &t, true
	}
	return nil, false
}

func findSunoTask(taskID string) (*model.SunoTask, bool) {
	var t model.SunoTask
	if err := common.DB.Where("task_id = ?", taskID).First(&t).Error; err == nil {
		return &t, true
	}
	return nil, false
}

func writeDataURL(c *gin.Context, dataURL string) {
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		common.Fail(c, common.CodeServerError, "无效的数据 URL")
		return
	}
	header := parts[0]
	content := parts[1]
	mime := "application/octet-stream"
	if strings.HasPrefix(header, "data:") {
		meta := strings.TrimPrefix(header, "data:")
		if idx := strings.Index(meta, ";"); idx > 0 {
			mime = meta[:idx]
		}
	}
	bs, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		bs, err = base64.RawStdEncoding.DecodeString(content)
		if err != nil {
			common.Fail(c, common.CodeServerError, "解码数据 URL 失败")
			return
		}
	}
	c.Header("Content-Type", mime)
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write(bs)
}
