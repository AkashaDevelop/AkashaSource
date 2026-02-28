package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func FetchChannelBalance(c *gin.Context) {
	id := c.Param("id")
	var channel model.Channel
	if err := common.DB.First(&channel, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}

	baseUrl := strings.TrimSuffix(channel.BaseURL, "/")
	key := service.GetNextKey(channel.Key)
	client := common.NewHTTPClient(channel.Proxy)

	var balance float64
	var fetchErr error

	switch channel.Type {
	case model.ChannelTypeSiliconFlow:
		if baseUrl == "" {
			baseUrl = "https://api.siliconflow.cn"
		}
		req, _ := http.NewRequest("GET", baseUrl+"/v1/user/info", nil)
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := client.Do(req)
		if err != nil {
			fetchErr = err
			break
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
		// OpenAI-compatible billing endpoint
		if baseUrl == "" {
			baseUrl = "https://api.openai.com"
		}
		req, _ := http.NewRequest("GET", baseUrl+"/v1/dashboard/billing/credit_grants", nil)
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := client.Do(req)
		if err != nil {
			fetchErr = err
			break
		}
		defer resp.Body.Close()
		var result struct {
			TotalAvailable float64 `json:"total_available"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			balance = result.TotalAvailable
		}
	}

	if fetchErr != nil {
		common.Fail(c, common.CodeServerError, "查询余额失败: "+fetchErr.Error())
		return
	}

	common.DB.Model(&channel).Update("balance", balance)
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
