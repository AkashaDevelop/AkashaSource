package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	ratioDefaultTimeoutSeconds = 10
	ratioDefaultEndpoint       = "/api/ratio_config"
)

type upstreamDTO struct {
	ID       int    `json:"id,omitempty"`
	Name     string `json:"name"`
	BaseURL  string `json:"base_url"`
	Endpoint string `json:"endpoint"`
}

type upstreamRequest struct {
	ChannelIDs []int64       `json:"channel_ids"`
	Upstreams  []upstreamDTO `json:"upstreams"`
	Timeout    int           `json:"timeout"`
}

type testResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func GetAllQuotaDates(c *gin.Context) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	username := c.Query("username")

	dates, err := model.GetAllQuotaDates(startTimestamp, endTimestamp, username)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": dates})
}

func GetUserQuotaDates(c *gin.Context) {
	userId := c.GetInt("id")
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)

	if endTimestamp-startTimestamp > 2592000 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "时间跨度不能超过 1 个月"})
		return
	}

	dates, err := model.GetQuotaDataByUserId(userId, startTimestamp, endTimestamp)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": dates})
}

func GetPrefillGroups(c *gin.Context) {
	groupType := c.Query("type")
	groups, err := model.GetAllPrefillGroups(groupType)
	if err != nil {
		common.Fail(c, common.CodeServerError, "获取预填组失败")
		return
	}
	common.OK(c, groups)
}

func CreatePrefillGroup(c *gin.Context) {
	var g model.PrefillGroup
	if err := c.ShouldBindJSON(&g); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if strings.TrimSpace(g.Name) == "" || strings.TrimSpace(g.Type) == "" {
		common.Fail(c, common.CodeParamError, "组名称和类型不能为空")
		return
	}
	if dup, err := model.IsPrefillGroupNameDuplicated(0, g.Name); err != nil {
		common.Fail(c, common.CodeServerError, "校验组名称失败")
		return
	} else if dup {
		common.Fail(c, common.CodeConflict, "组名称已存在")
		return
	}
	if err := g.Insert(); err != nil {
		common.Fail(c, common.CodeServerError, "创建预填组失败")
		return
	}
	common.OK(c, g)
}

func UpdatePrefillGroup(c *gin.Context) {
	var g model.PrefillGroup
	if err := c.ShouldBindJSON(&g); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if g.Id == 0 {
		common.Fail(c, common.CodeParamError, "缺少组 ID")
		return
	}
	if dup, err := model.IsPrefillGroupNameDuplicated(g.Id, g.Name); err != nil {
		common.Fail(c, common.CodeServerError, "校验组名称失败")
		return
	} else if dup {
		common.Fail(c, common.CodeConflict, "组名称已存在")
		return
	}
	if err := g.Update(); err != nil {
		common.Fail(c, common.CodeServerError, "更新预填组失败")
		return
	}
	common.OKMsg(c, "更新成功", g)
}

func DeletePrefillGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "组 ID 非法")
		return
	}
	if err = model.DeletePrefillGroupByID(id); err != nil {
		common.Fail(c, common.CodeServerError, "删除预填组失败")
		return
	}
	common.OKMsg(c, "删除成功", nil)
}

func GetSyncableChannels(c *gin.Context) {
	var channels []model.Channel
	if err := common.DB.Order("id desc").Find(&channels).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	type syncableChannel struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		BaseURL string `json:"base_url"`
		Status  int    `json:"status"`
		Type    int    `json:"type"`
	}
	results := make([]syncableChannel, 0, len(channels)+2)
	for _, ch := range channels {
		base := strings.TrimSpace(ch.BaseURL)
		if base == "" {
			continue
		}
		results = append(results, syncableChannel{ID: ch.Id, Name: ch.Name, BaseURL: base, Status: ch.Status, Type: ch.Type})
	}
	results = append(results,
		syncableChannel{ID: -100, Name: "官方倍率预设", BaseURL: "https://basellm.github.io", Status: 1},
		syncableChannel{ID: -101, Name: "models.dev 价格预设", BaseURL: "https://models.dev", Status: 1},
	)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": results})
}

func FetchUpstreamRatios(c *gin.Context) {
	var req upstreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数格式错误"})
		return
	}

	if req.Timeout <= 0 {
		req.Timeout = ratioDefaultTimeoutSeconds
	}

	upstreams := make([]upstreamDTO, 0)
	if len(req.Upstreams) > 0 {
		for _, u := range req.Upstreams {
			u.BaseURL = strings.TrimRight(strings.TrimSpace(u.BaseURL), "/")
			if !strings.HasPrefix(u.BaseURL, "http") {
				continue
			}
			if strings.TrimSpace(u.Endpoint) == "" {
				u.Endpoint = ratioDefaultEndpoint
			}
			upstreams = append(upstreams, u)
		}
	} else if len(req.ChannelIDs) > 0 {
		ids := make([]int, 0, len(req.ChannelIDs))
		for _, id := range req.ChannelIDs {
			ids = append(ids, int(id))
		}
		var channels []model.Channel
		if err := common.DB.Where("id IN ?", ids).Find(&channels).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "查询渠道失败"})
			return
		}
		for _, ch := range channels {
			base := strings.TrimRight(strings.TrimSpace(ch.BaseURL), "/")
			if !strings.HasPrefix(base, "http") {
				continue
			}
			upstreams = append(upstreams, upstreamDTO{ID: ch.Id, Name: ch.Name, BaseURL: base, Endpoint: ratioDefaultEndpoint})
		}
	}

	if len(upstreams) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无有效上游渠道"})
		return
	}

	testResults := make([]testResult, 0, len(upstreams))
	differences := map[string]map[string]any{}
	httpClient := &http.Client{Timeout: time.Duration(req.Timeout) * time.Second}

	for _, u := range upstreams {
		endpoint := strings.TrimSpace(u.Endpoint)
		if endpoint == "" {
			endpoint = ratioDefaultEndpoint
		}

		fullURL := endpoint
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			if !strings.HasPrefix(endpoint, "/") {
				endpoint = "/" + endpoint
			}
			fullURL = u.BaseURL + endpoint
		}

		resp, err := httpClient.Get(fullURL)
		if err != nil {
			testResults = append(testResults, testResult{Name: u.Name, Status: "error", Error: err.Error()})
			continue
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
		resp.Body.Close()
		if readErr != nil {
			testResults = append(testResults, testResult{Name: u.Name, Status: "error", Error: readErr.Error()})
			continue
		}
		if resp.StatusCode != http.StatusOK {
			testResults = append(testResults, testResult{Name: u.Name, Status: "error", Error: resp.Status})
			continue
		}

		if !extractRatioPayload(body) {
			testResults = append(testResults, testResult{Name: u.Name, Status: "error", Error: "无法解析上游返回数据"})
			continue
		}

		testResults = append(testResults, testResult{Name: u.Name, Status: "success"})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"differences":  differences,
			"test_results": testResults,
		},
	})
}

func extractRatioPayload(body []byte) bool {
	var wrapped struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
		Message string          `json:"message"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil && len(wrapped.Data) > 0 {
		if !wrapped.Success {
			return false
		}
		var m map[string]any
		if err2 := json.Unmarshal(wrapped.Data, &m); err2 == nil {
			if _, ok := m["model_ratio"]; ok {
				return true
			}
			if _, ok := m["completion_ratio"]; ok {
				return true
			}
			if _, ok := m["model_price"]; ok {
				return true
			}
		}
		var arr []map[string]any
		if err2 := json.Unmarshal(wrapped.Data, &arr); err2 == nil && len(arr) >= 0 {
			return true
		}
	}

	var direct map[string]any
	if err := json.Unmarshal(body, &direct); err == nil {
		if _, ok := direct["model_ratio"]; ok {
			return true
		}
		if _, ok := direct["completion_ratio"]; ok {
			return true
		}
		if _, ok := direct["model_price"]; ok {
			return true
		}
	}
	return false
}
