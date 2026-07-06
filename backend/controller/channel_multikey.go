package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var multiKeyLockStore = struct {
	sync.Mutex
	m map[int]*sync.Mutex
}{m: map[int]*sync.Mutex{}}

func getMultiKeyLock(channelID int) *sync.Mutex {
	multiKeyLockStore.Lock()
	defer multiKeyLockStore.Unlock()
	if l, ok := multiKeyLockStore.m[channelID]; ok {
		return l
	}
	l := &sync.Mutex{}
	multiKeyLockStore.m[channelID] = l
	return l
}

func splitLinesNonEmpty(s string) []string {
	items := make([]string, 0)
	for _, k := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		k = strings.TrimSpace(k)
		if k != "" {
			items = append(items, k)
		}
	}
	return items
}

func parseIntMap(s string) map[int]int {
	if strings.TrimSpace(s) == "" {
		return map[int]int{}
	}
	raw := map[string]int{}
	if json.Unmarshal([]byte(s), &raw) != nil {
		return map[int]int{}
	}
	ret := map[int]int{}
	for k, v := range raw {
		if idx, err := strconv.Atoi(k); err == nil {
			ret[idx] = v
		}
	}
	return ret
}

func parseInt64Map(s string) map[int]int64 {
	if strings.TrimSpace(s) == "" {
		return map[int]int64{}
	}
	raw := map[string]int64{}
	if json.Unmarshal([]byte(s), &raw) != nil {
		return map[int]int64{}
	}
	ret := map[int]int64{}
	for k, v := range raw {
		if idx, err := strconv.Atoi(k); err == nil {
			ret[idx] = v
		}
	}
	return ret
}

func parseStringMap(s string) map[int]string {
	if strings.TrimSpace(s) == "" {
		return map[int]string{}
	}
	raw := map[string]string{}
	if json.Unmarshal([]byte(s), &raw) != nil {
		return map[int]string{}
	}
	ret := map[int]string{}
	for k, v := range raw {
		if idx, err := strconv.Atoi(k); err == nil {
			ret[idx] = v
		}
	}
	return ret
}

func marshalIntMap(m map[int]int) string {
	raw := map[string]int{}
	for k, v := range m {
		raw[strconv.Itoa(k)] = v
	}
	b, _ := json.Marshal(raw)
	return string(b)
}

func marshalInt64Map(m map[int]int64) string {
	raw := map[string]int64{}
	for k, v := range m {
		raw[strconv.Itoa(k)] = v
	}
	b, _ := json.Marshal(raw)
	return string(b)
}

func marshalStringMap(m map[int]string) string {
	raw := map[string]string{}
	for k, v := range m {
		raw[strconv.Itoa(k)] = v
	}
	b, _ := json.Marshal(raw)
	return string(b)
}

type MultiKeyManageRequest struct {
	ChannelID int      `json:"channel_id"`
	Action    string   `json:"action"`
	KeyIndex  *int     `json:"key_index,omitempty"`
	Page      int      `json:"page,omitempty"`
	PageSize  int      `json:"page_size,omitempty"`
	Status    *int     `json:"status,omitempty"`
	Keys      []string `json:"keys,omitempty"`
}

type KeyStatus struct {
	Index        int    `json:"index"`
	Status       int    `json:"status"`
	DisabledTime int64  `json:"disabled_time,omitempty"`
	Reason       string `json:"reason,omitempty"`
	KeyPreview   string `json:"key_preview"`
}

func ManageMultiKeys(c *gin.Context) {
	var req MultiKeyManageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	req.Action = strings.TrimSpace(req.Action)
	if req.ChannelID == 0 {
		common.Fail(c, common.CodeParamError, "channel_id 不能为空")
		return
	}
	if req.Action == "" {
		common.Fail(c, common.CodeParamError, "action 不能为空")
		return
	}

	var ch model.Channel
	if err := common.DB.First(&ch, req.ChannelID).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}

	lock := getMultiKeyLock(ch.Id)
	lock.Lock()
	defer lock.Unlock()

	keys := splitLinesNonEmpty(ch.Key)
	statusMap := parseIntMap(ch.MultiKeyStatus)
	disabledTimeMap := parseInt64Map(ch.MultiKeyDisabledTime)
	disabledReasonMap := parseStringMap(ch.MultiKeyDisabledReason)

	saveState := func() error {
		updates := map[string]interface{}{
			"key":                       strings.Join(keys, "\n"),
			"multi_key_status":          marshalIntMap(statusMap),
			"multi_key_disabled_time":   marshalInt64Map(disabledTimeMap),
			"multi_key_disabled_reason": marshalStringMap(disabledReasonMap),
		}
		return common.DB.Model(&model.Channel{}).Where("id = ?", ch.Id).Updates(updates).Error
	}

	switch req.Action {
	case "set":
		keys = make([]string, 0, len(req.Keys))
		for _, k := range req.Keys {
			k = strings.TrimSpace(k)
			if k != "" {
				keys = append(keys, k)
			}
		}
		if len(keys) == 0 {
			common.Fail(c, common.CodeParamError, "keys 不能为空")
			return
		}
		statusMap = map[int]int{}
		disabledTimeMap = map[int]int64{}
		disabledReasonMap = map[int]string{}
		if err := saveState(); err != nil {
			common.Fail(c, common.CodeServerError, "更新多 Key 失败")
			return
		}
		common.OK(c, gin.H{"success": true, "message": "", "data": gin.H{"count": len(keys)}})
		return

	case "get_key_status":
		page := req.Page
		pageSize := req.PageSize
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 50
		}
		if pageSize > 200 {
			pageSize = 200
		}

		all := make([]KeyStatus, 0, len(keys))
		enabledCount, manualDisabledCount, autoDisabledCount := 0, 0, 0
		for i, k := range keys {
			st := 1
			if v, ok := statusMap[i]; ok {
				st = v
			}
			switch st {
			case 1:
				enabledCount++
			case 2:
				manualDisabledCount++
			case 3:
				autoDisabledCount++
			}
			preview := k
			if len(preview) > 10 {
				preview = preview[:10] + "..."
			}
			item := KeyStatus{Index: i, Status: st, KeyPreview: preview}
			if st != 1 {
				if t, ok := disabledTimeMap[i]; ok {
					item.DisabledTime = t
				}
				if r, ok := disabledReasonMap[i]; ok {
					item.Reason = r
				}
			}
			all = append(all, item)
		}

		filtered := all
		if req.Status != nil {
			filtered = make([]KeyStatus, 0)
			for _, item := range all {
				if item.Status == *req.Status {
					filtered = append(filtered, item)
				}
			}
		}

		total := len(filtered)
		totalPages := (total + pageSize - 1) / pageSize
		if totalPages == 0 {
			totalPages = 1
		}
		if page > totalPages {
			page = totalPages
		}
		start := (page - 1) * pageSize
		end := start + pageSize
		if end > total {
			end = total
		}
		pageData := make([]KeyStatus, 0)
		if start < total {
			pageData = filtered[start:end]
		}
		common.OK(c, gin.H{"success": true, "message": "", "data": gin.H{
			"keys":  pageData,
			"total": total, "page": page, "page_size": pageSize, "total_pages": totalPages,
			"enabled_count": enabledCount, "manual_disabled_count": manualDisabledCount, "auto_disabled_count": autoDisabledCount,
		}})
		return

	case "disable_key", "enable_key", "delete_key":
		if req.KeyIndex == nil {
			common.Fail(c, common.CodeParamError, "未指定要操作的密钥索引")
			return
		}
		idx := *req.KeyIndex
		if idx < 0 || idx >= len(keys) {
			common.Fail(c, common.CodeParamError, "密钥索引超出范围")
			return
		}

		if req.Action == "disable_key" {
			statusMap[idx] = 2
			disabledTimeMap[idx] = time.Now().Unix()
			disabledReasonMap[idx] = "manual"
		} else if req.Action == "enable_key" {
			delete(statusMap, idx)
			delete(disabledTimeMap, idx)
			delete(disabledReasonMap, idx)
		} else {
			if len(keys) <= 1 {
				common.Fail(c, common.CodeParamError, "不能删除最后一个密钥")
				return
			}
			nextKeys := make([]string, 0, len(keys)-1)
			nextStatus := map[int]int{}
			nextTime := map[int]int64{}
			nextReason := map[int]string{}
			ni := 0
			for i, key := range keys {
				if i == idx {
					continue
				}
				nextKeys = append(nextKeys, key)
				if st, ok := statusMap[i]; ok && st != 1 {
					nextStatus[ni] = st
				}
				if t, ok := disabledTimeMap[i]; ok {
					nextTime[ni] = t
				}
				if r, ok := disabledReasonMap[i]; ok {
					nextReason[ni] = r
				}
				ni++
			}
			keys = nextKeys
			statusMap = nextStatus
			disabledTimeMap = nextTime
			disabledReasonMap = nextReason
		}
		if err := saveState(); err != nil {
			common.Fail(c, common.CodeServerError, "更新多 Key 失败")
			return
		}
		common.OK(c, gin.H{"success": true, "message": ""})
		return

	case "enable_all_keys":
		statusMap = map[int]int{}
		disabledTimeMap = map[int]int64{}
		disabledReasonMap = map[int]string{}
		if err := saveState(); err != nil {
			common.Fail(c, common.CodeServerError, "更新多 Key 失败")
			return
		}
		common.OK(c, gin.H{"success": true, "message": ""})
		return

	case "disable_all_keys":
		now := time.Now().Unix()
		disabledCount := 0
		for i := range keys {
			st := 1
			if v, ok := statusMap[i]; ok {
				st = v
			}
			if st == 1 {
				statusMap[i] = 2
				disabledTimeMap[i] = now
				disabledReasonMap[i] = "manual"
				disabledCount++
			}
		}
		if disabledCount == 0 {
			common.Fail(c, common.CodeParamError, "没有可禁用的密钥")
			return
		}
		if err := saveState(); err != nil {
			common.Fail(c, common.CodeServerError, "更新多 Key 失败")
			return
		}
		common.OK(c, gin.H{"success": true, "message": ""})
		return

	case "delete_disabled_keys":
		nextKeys := make([]string, 0, len(keys))
		nextStatus := map[int]int{}
		nextTime := map[int]int64{}
		nextReason := map[int]string{}
		deleted := 0
		ni := 0
		for i, k := range keys {
			st := 1
			if v, ok := statusMap[i]; ok {
				st = v
			}
			if st == 3 {
				deleted++
				continue
			}
			nextKeys = append(nextKeys, k)
			if st != 1 {
				nextStatus[ni] = st
				if t, ok := disabledTimeMap[i]; ok {
					nextTime[ni] = t
				}
				if r, ok := disabledReasonMap[i]; ok {
					nextReason[ni] = r
				}
			}
			ni++
		}
		if deleted == 0 {
			common.Fail(c, common.CodeParamError, "没有需要删除的自动禁用密钥")
			return
		}
		if len(nextKeys) == 0 {
			common.Fail(c, common.CodeParamError, "不能删除最后一个密钥")
			return
		}
		keys = nextKeys
		statusMap = nextStatus
		disabledTimeMap = nextTime
		disabledReasonMap = nextReason
		if err := saveState(); err != nil {
			common.Fail(c, common.CodeServerError, "更新多 Key 失败")
			return
		}
		common.OK(c, gin.H{"success": true, "message": "", "data": deleted})
		return

	default:
		common.Fail(c, common.CodeParamError, "不支持的操作")
		return
	}
}
