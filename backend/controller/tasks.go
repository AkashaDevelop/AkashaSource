package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// UnifiedTask 统一任务响应结构
type UnifiedTask struct {
	Id         string `json:"id"`
	Type       string `json:"type"` // midjourney, suno
	UserId     int    `json:"user_id"`
	Action     string `json:"action"`
	Status     string `json:"status"`
	Input      string `json:"input"`
	ResultUrl  string `json:"result_url,omitempty"`
	Progress   string `json:"progress,omitempty"`
	FailReason string `json:"fail_reason,omitempty"`
	Quota      int64  `json:"quota"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at,omitempty"`
	FinishTime int64  `json:"finish_time,omitempty"`
}

func convertMJTaskToUnified(task *model.MidjourneyTask) UnifiedTask {
	return UnifiedTask{
		Id:         task.MjId,
		Type:       "midjourney",
		UserId:     task.UserId,
		Action:     task.Action,
		Status:     normalizeTaskStatus(task.Status),
		Input:      task.Prompt,
		ResultUrl:  strings.TrimSpace(task.ImageUrl),
		Progress:   task.Progress,
		FailReason: task.FailReason,
		Quota:      task.Quota,
		CreatedAt:  task.CreatedAt,
		FinishTime: task.FinishTime,
	}
}

func convertSunoTaskToUnified(task *model.SunoTask) UnifiedTask {
	return UnifiedTask{
		Id:        task.TaskId,
		Type:      "suno",
		UserId:    task.UserId,
		Action:    task.Action,
		Status:    normalizeTaskStatus(task.Status),
		Input:     task.Input,
		ResultUrl: extractResultURL(task.Result),
		Quota:     task.Quota,
		CreatedAt: task.CreatedAt,
		UpdatedAt: task.UpdatedAt,
	}
}

func normalizeTaskStatus(status string) string {
	s := strings.ToLower(strings.TrimSpace(status))
	switch s {
	case "success", "completed", "succeeded":
		return "succeeded"
	case "failed", "failure", "error":
		return "failed"
	case "processing", "submitted":
		return "processing"
	default:
		return "pending"
	}
}

func extractResultURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") || strings.HasPrefix(raw, "data:") {
		return raw
	}
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return ""
	}
	return findURL(payload)
}

func findURL(v any) string {
	switch x := v.(type) {
	case map[string]any:
		for _, vv := range x {
			if u := findURL(vv); u != "" {
				return u
			}
		}
	case []any:
		for _, vv := range x {
			if u := findURL(vv); u != "" {
				return u
			}
		}
	case string:
		s := strings.TrimSpace(x)
		if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "data:") {
			return s
		}
	}
	return ""
}

// Admin: list all MJ tasks
func AdminGetMJTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	var tasks []model.MidjourneyTask
	var total int64
	query := common.DB.Model(&model.MidjourneyTask{})

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)
	query.Order("id desc").Offset(offset).Limit(size).Find(&tasks)

	common.OK(c, gin.H{"tasks": tasks, "total": total})
}

// GetAllTasks 获取所有任务（MJ + Suno 聚合）- 管理员接口
func GetAllTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	taskType := c.Query("type")   // midjourney, suno, all
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	var unifiedTasks []UnifiedTask
	var total int64

	// 根据类型过滤
	if taskType == "" || taskType == "all" {
		// 查询两种任务
		var mjTasks []model.MidjourneyTask
		var sunoTasks []model.SunoTask

		mjQuery := common.DB.Model(&model.MidjourneyTask{})
		sunoQuery := common.DB.Model(&model.SunoTask{})

		if status != "" {
			mjQuery = mjQuery.Where("status = ?", status)
			sunoQuery = sunoQuery.Where("status = ?", status)
		}

		var mjTotal, sunoTotal int64
		mjQuery.Count(&mjTotal)
		sunoQuery.Count(&sunoTotal)
		total = mjTotal + sunoTotal

		// 简单实现：分别获取并合并
		mjQuery.Order("created_at desc").Limit(size).Find(&mjTasks)
		sunoQuery.Order("created_at desc").Limit(size).Find(&sunoTasks)

		for _, t := range mjTasks {
			unifiedTasks = append(unifiedTasks, convertMJTaskToUnified(&t))
		}
		for _, t := range sunoTasks {
			unifiedTasks = append(unifiedTasks, convertSunoTaskToUnified(&t))
		}
	} else if taskType == "midjourney" {
		var mjTasks []model.MidjourneyTask
		query := common.DB.Model(&model.MidjourneyTask{})

		if status != "" {
			query = query.Where("status = ?", status)
		}

		query.Count(&total)
		offset := (page - 1) * size
		query.Order("created_at desc").Offset(offset).Limit(size).Find(&mjTasks)

		for _, t := range mjTasks {
			unifiedTasks = append(unifiedTasks, convertMJTaskToUnified(&t))
		}
	} else if taskType == "suno" {
		var sunoTasks []model.SunoTask
		query := common.DB.Model(&model.SunoTask{})

		if status != "" {
			query = query.Where("status = ?", status)
		}

		query.Count(&total)
		offset := (page - 1) * size
		query.Order("created_at desc").Offset(offset).Limit(size).Find(&sunoTasks)

		for _, t := range sunoTasks {
			unifiedTasks = append(unifiedTasks, convertSunoTaskToUnified(&t))
		}
	}

	common.OK(c, gin.H{"tasks": unifiedTasks, "total": total})
}

// GetUserTasks 获取用户自己的所有任务（MJ + Suno 聚合）
func GetUserTasks(c *gin.Context) {
	userId := c.GetInt("id")
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	taskType := c.Query("type")   // midjourney, suno, all
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	var unifiedTasks []UnifiedTask
	var total int64

	// 根据类型过滤
	if taskType == "" || taskType == "all" {
		// 查询两种任务
		var mjTasks []model.MidjourneyTask
		var sunoTasks []model.SunoTask

		mjQuery := common.DB.Model(&model.MidjourneyTask{}).Where("user_id = ?", userId)
		sunoQuery := common.DB.Model(&model.SunoTask{}).Where("user_id = ?", userId)

		if status != "" {
			mjQuery = mjQuery.Where("status = ?", status)
			sunoQuery = sunoQuery.Where("status = ?", status)
		}

		var mjTotal, sunoTotal int64
		mjQuery.Count(&mjTotal)
		sunoQuery.Count(&sunoTotal)
		total = mjTotal + sunoTotal

		// 简单实现：分别获取并合并
		mjQuery.Order("created_at desc").Limit(size).Find(&mjTasks)
		sunoQuery.Order("created_at desc").Limit(size).Find(&sunoTasks)

		for _, t := range mjTasks {
			unifiedTasks = append(unifiedTasks, convertMJTaskToUnified(&t))
		}
		for _, t := range sunoTasks {
			unifiedTasks = append(unifiedTasks, convertSunoTaskToUnified(&t))
		}
	} else if taskType == "midjourney" {
		var mjTasks []model.MidjourneyTask
		query := common.DB.Model(&model.MidjourneyTask{}).Where("user_id = ?", userId)

		if status != "" {
			query = query.Where("status = ?", status)
		}

		query.Count(&total)
		offset := (page - 1) * size
		query.Order("created_at desc").Offset(offset).Limit(size).Find(&mjTasks)

		for _, t := range mjTasks {
			unifiedTasks = append(unifiedTasks, convertMJTaskToUnified(&t))
		}
	} else if taskType == "suno" {
		var sunoTasks []model.SunoTask
		query := common.DB.Model(&model.SunoTask{}).Where("user_id = ?", userId)

		if status != "" {
			query = query.Where("status = ?", status)
		}

		query.Count(&total)
		offset := (page - 1) * size
		query.Order("created_at desc").Offset(offset).Limit(size).Find(&sunoTasks)

		for _, t := range sunoTasks {
			unifiedTasks = append(unifiedTasks, convertSunoTaskToUnified(&t))
		}
	}

	common.OK(c, gin.H{"tasks": unifiedTasks, "total": total})
}

// Admin: list all Suno tasks
func AdminGetSunoTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	var tasks []model.SunoTask
	var total int64
	query := common.DB.Model(&model.SunoTask{})

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)
	query.Order("id desc").Offset(offset).Limit(size).Find(&tasks)

	common.OK(c, gin.H{"tasks": tasks, "total": total})
}

// User: list own MJ tasks
func UserGetMJTasks(c *gin.Context) {
	userId := c.GetInt("id")
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	var tasks []model.MidjourneyTask
	var total int64
	common.DB.Model(&model.MidjourneyTask{}).Where("user_id = ?", userId).Count(&total)
	common.DB.Where("user_id = ?", userId).Order("id desc").Offset(offset).Limit(size).Find(&tasks)

	common.OK(c, gin.H{"tasks": tasks, "total": total})
}

// User: list own Suno tasks
func UserGetSunoTasks(c *gin.Context) {
	userId := c.GetInt("id")
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	var tasks []model.SunoTask
	var total int64
	common.DB.Model(&model.SunoTask{}).Where("user_id = ?", userId).Count(&total)
	common.DB.Where("user_id = ?", userId).Order("id desc").Offset(offset).Limit(size).Find(&tasks)

	common.OK(c, gin.H{"tasks": tasks, "total": total})
}
