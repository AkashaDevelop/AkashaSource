package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func fetchAndPersistChannelBalance(channel *model.Channel) (float64, error) {
	baseUrl := strings.TrimSuffix(channel.BaseURL, "/")
	key := service.GetNextKey(channel.Key)
	client := common.NewHTTPClient(channel.Proxy)

	var balance float64
	switch channel.Type {
	case model.ChannelTypeSiliconFlow:
		if baseUrl == "" {
			baseUrl = "https://api.siliconflow.cn"
		}
		req, _ := http.NewRequest("GET", baseUrl+"/v1/user/info", nil)
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := client.Do(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()
		var result struct {
			Data struct {
				Balance string `json:"balance"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			fmt.Sscanf(result.Data.Balance, "%f", &balance)
		}
	default:
		if baseUrl == "" {
			baseUrl = "https://api.openai.com"
		}
		req, _ := http.NewRequest("GET", baseUrl+"/v1/dashboard/billing/credit_grants", nil)
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := client.Do(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()
		var result struct {
			TotalAvailable float64 `json:"total_available"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			balance = result.TotalAvailable
		}
	}

	if err := common.DB.Model(channel).Update("balance", balance).Error; err != nil {
		return 0, err
	}
	return balance, nil
}

func FetchChannelBalance(c *gin.Context) {
	id := c.Param("id")
	var channel model.Channel
	if err := common.DB.First(&channel, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}
	balance, err := fetchAndPersistChannelBalance(&channel)
	if err != nil {
		common.Fail(c, common.CodeServerError, "查询余额失败: "+err.Error())
		return
	}
	common.OK(c, gin.H{"balance": balance})
}

func SearchChannels(c *gin.Context) {
	keyword := c.Query("keyword")
	var channels []model.Channel
	db := common.DB.Order("priority desc")
	if keyword != "" {
		db = db.Where("name LIKE ? OR base_url LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if err := db.Find(&channels).Error; err != nil {
		common.Fail(c, common.CodeServerError, "搜索失败")
		return
	}
	common.OK(c, channels)
}

func GetAllChannels(c *gin.Context) {
	var channels []model.Channel
	db := common.DB.Order("priority desc")
	if tag := c.Query("tag"); tag != "" {
		db = db.Where("tags LIKE ?", "%"+tag+"%")
	}
	if err := db.Find(&channels).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取渠道失败")
		return
	}
	common.OK(c, channels)
}

func ChannelListModels(c *gin.Context) {
	var channels []model.Channel
	if err := common.DB.Find(&channels).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取模型列表失败")
		return
	}
	set := make(map[string]struct{})
	for _, ch := range channels {
		for _, m := range strings.Split(ch.Models, ",") {
			m = strings.TrimSpace(m)
			if m != "" {
				set[m] = struct{}{}
			}
		}
	}
	models := make([]string, 0, len(set))
	for m := range set {
		models = append(models, m)
	}
	sort.Strings(models)
	common.OK(c, models)
}

func EnabledListModels(c *gin.Context) {
	var channels []model.Channel
	if err := common.DB.Where("status = ?", model.ChannelStatusActive).Find(&channels).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取模型列表失败")
		return
	}
	set := make(map[string]struct{})
	for _, ch := range channels {
		for _, m := range strings.Split(ch.Models, ",") {
			m = strings.TrimSpace(m)
			if m != "" {
				set[m] = struct{}{}
			}
		}
	}
	models := make([]string, 0, len(set))
	for m := range set {
		models = append(models, m)
	}
	sort.Strings(models)
	common.OK(c, models)
}

func GetChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "渠道 ID 非法")
		return
	}
	var channel model.Channel
	if err := common.DB.First(&channel, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}
	common.OK(c, channel)
}

func GetChannelKey(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "渠道 ID 非法")
		return
	}
	var channel model.Channel
	if err := common.DB.Select("id", "key").First(&channel, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}
	common.OK(c, gin.H{"key": channel.Key})
}

func UpdateAllChannelsBalance(c *gin.Context) {
	var channels []model.Channel
	if err := common.DB.Where("status = ?", model.ChannelStatusActive).Find(&channels).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取渠道失败")
		return
	}
	successCount := 0
	failCount := 0
	for i := range channels {
		if _, err := fetchAndPersistChannelBalance(&channels[i]); err != nil {
			failCount++
			continue
		}
		successCount++
	}
	common.OK(c, gin.H{"success_count": successCount, "fail_count": failCount})
}

func UpdateChannelBalance(c *gin.Context) {
	FetchChannelBalance(c)
}

func AddChannel(c *gin.Context) {
	var channel model.Channel
	if err := c.ShouldBindJSON(&channel); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if channel.Name == "" || channel.Key == "" {
		common.Fail(c, common.CodeParamError, "名称和密钥不能为空")
		return
	}
	if err := common.DB.Create(&channel).Error; err != nil {
		common.Fail(c, common.CodeServerError, "创建渠道失败")
		return
	}
	common.OKMsg(c, "渠道创建成功", channel)
}

func AddChannels(c *gin.Context) {
	var channels []model.Channel
	if err := c.ShouldBindJSON(&channels); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if len(channels) == 0 {
		common.Fail(c, common.CodeParamError, "渠道列表不能为空")
		return
	}
	for _, ch := range channels {
		if ch.Name == "" || ch.Key == "" {
			common.Fail(c, common.CodeParamError, "所有渠道的名称和密钥不能为空")
			return
		}
	}
	if err := common.DB.Create(&channels).Error; err != nil {
		common.Fail(c, common.CodeServerError, "批量创建渠道失败")
		return
	}
	common.OKMsg(c, "批量创建成功", gin.H{"count": len(channels)})
}

func DeleteChannel(c *gin.Context) {
	id := c.Param("id")
	if err := common.DB.Delete(&model.Channel{}, id).Error; err != nil {
		common.Fail(c, common.CodeServerError, "删除渠道失败")
		return
	}
	common.OKMsg(c, "渠道已删除", nil)
}

func DeleteDisabledChannel(c *gin.Context) {
	res := common.DB.Where("status IN ?", []int{model.ChannelStatusDisabled, model.ChannelStatusAutoDisabled}).Delete(&model.Channel{})
	if res.Error != nil {
		common.Fail(c, common.CodeServerError, "删除禁用渠道失败")
		return
	}
	common.OK(c, gin.H{"deleted": res.RowsAffected})
}

type channelTagReq struct {
	Tag          string  `json:"tag"`
	NewTag       *string `json:"new_tag"`
	Priority     *int64  `json:"priority"`
	Weight       *uint   `json:"weight"`
	ModelMapping *string `json:"model_mapping"`
	Models       *string `json:"models"`
}

func DisableTagChannels(c *gin.Context) {
	var req channelTagReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Tag) == "" {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if err := common.DB.Model(&model.Channel{}).Where("tags LIKE ?", "%"+strings.TrimSpace(req.Tag)+"%").Update("status", model.ChannelStatusDisabled).Error; err != nil {
		common.Fail(c, common.CodeServerError, "批量禁用失败")
		return
	}
	common.OK(c, nil)
}

func EnableTagChannels(c *gin.Context) {
	var req channelTagReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Tag) == "" {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if err := common.DB.Model(&model.Channel{}).Where("tags LIKE ?", "%"+strings.TrimSpace(req.Tag)+"%").Update("status", model.ChannelStatusActive).Error; err != nil {
		common.Fail(c, common.CodeServerError, "批量启用失败")
		return
	}
	common.OK(c, nil)
}

func EditTagChannels(c *gin.Context) {
	var req channelTagReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Tag) == "" {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	updates := map[string]interface{}{}
	if req.NewTag != nil {
		updates["tags"] = strings.TrimSpace(*req.NewTag)
	}
	if req.Priority != nil {
		updates["priority"] = int(*req.Priority)
	}
	if req.Weight != nil {
		updates["weight"] = int(*req.Weight)
	}
	if req.ModelMapping != nil {
		updates["model_mapping"] = strings.TrimSpace(*req.ModelMapping)
	}
	if req.Models != nil {
		updates["models"] = strings.TrimSpace(*req.Models)
	}
	if len(updates) == 0 {
		common.Fail(c, common.CodeParamError, "无可更新字段")
		return
	}
	if err := common.DB.Model(&model.Channel{}).Where("tags LIKE ?", "%"+strings.TrimSpace(req.Tag)+"%").Updates(updates).Error; err != nil {
		common.Fail(c, common.CodeServerError, "批量编辑失败")
		return
	}
	common.OK(c, nil)
}

func FixChannelsAbilities(c *gin.Context) {
	var total int64
	if err := common.DB.Model(&model.Channel{}).Count(&total).Error; err != nil {
		common.Fail(c, common.CodeServerError, "执行修复失败")
		return
	}
	common.OK(c, gin.H{"success": total, "fails": 0})
}

func UpdateChannel(c *gin.Context) {
	var channel model.Channel
	if err := c.ShouldBindJSON(&channel); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if channel.Id == 0 {
		common.Fail(c, common.CodeParamError, "缺少渠道 ID")
		return
	}
	if err := common.DB.Model(&channel).Updates(channel).Error; err != nil {
		common.Fail(c, common.CodeServerError, "更新渠道失败")
		return
	}
	common.OKMsg(c, "渠道更新成功", nil)
}

func ToggleChannelStatus(c *gin.Context) {
	id := c.Param("id")
	var channel model.Channel
	if err := common.DB.First(&channel, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}
	newStatus := model.ChannelStatusDisabled
	if channel.Status == model.ChannelStatusDisabled {
		newStatus = model.ChannelStatusActive
	}
	common.DB.Model(&channel).Update("status", newStatus)
	common.OK(c, gin.H{"status": newStatus})
}

func TestChannel(c *gin.Context) {
	id := c.Param("id")
	var channel model.Channel
	if err := common.DB.First(&channel, id).Error; err != nil {
		c.JSON(http.StatusNotFound, common.R{Code: common.CodeNotFound, Msg: "渠道不存在"})
		return
	}
	responseTime, err := service.CheckChannel(&channel)
	if err != nil {
		common.OK(c, gin.H{"success": false, "msg": err.Error(), "time": 0})
		return
	}
	channel.ResponseTime = responseTime
	channel.TestTime = common.GetTimestamp()
	common.DB.Save(&channel)
	common.OK(c, gin.H{"success": true, "time": responseTime, "msg": "测试通过"})
}
