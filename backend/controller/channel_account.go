package controller

import (
	"strconv"

	"STfreApi/adapter"
	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service"

	"github.com/gin-gonic/gin"
)

func TriggerChannelCheckin(c *gin.Context) {
	result, err := service.TriggerCheckin()
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}
	common.OK(c, result)
}

func TriggerChannelBalanceRefresh(c *gin.Context) {
	result, err := service.TriggerBalanceRefresh()
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}
	common.OK(c, result)
}

func CheckinSingleChannel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.CodeParamError, "无效的渠道ID")
		return
	}

	var channel model.Channel
	if err := common.DB.First(&channel, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}

	result, err := service.CheckinChannelWithLog(&channel)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	common.OK(c, result)
}

func RefreshSingleChannelBalance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.CodeParamError, "无效的渠道ID")
		return
	}

	var channel model.Channel
	if err := common.DB.First(&channel, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}

	balanceInfo, err := service.RefreshChannelBalance(&channel)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	common.OK(c, gin.H{
		"channel_id":   channel.Id,
		"channel_name": channel.Name,
		"balance":      balanceInfo.Balance,
		"used":         balanceInfo.Used,
		"quota":        balanceInfo.Quota,
	})
}

func GetChannelCheckinLogs(c *gin.Context) {
	channelIdStr := c.Query("channel_id")
	limitStr := c.DefaultQuery("limit", "50")

	limit, _ := strconv.Atoi(limitStr)
	channelId := 0
	if channelIdStr != "" {
		channelId, _ = strconv.Atoi(channelIdStr)
	}

	logs, err := model.GetCheckinLogs(channelId, limit)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	channelIds := make(map[int]string)
	for _, log := range logs {
		if _, ok := channelIds[log.ChannelId]; !ok {
			var channel model.Channel
			if err := common.DB.Select("name").First(&channel, log.ChannelId).Error; err == nil {
				channelIds[log.ChannelId] = channel.Name
			}
		}
	}

	result := make([]map[string]interface{}, len(logs))
	for i, log := range logs {
		result[i] = map[string]interface{}{
			"id":           log.Id,
			"channel_id":   log.ChannelId,
			"channel_name": channelIds[log.ChannelId],
			"status":       log.Status,
			"message":      log.Message,
			"reward":       log.Reward,
			"created_at":   log.CreatedAt,
		}
	}

	common.OK(c, result)
}

func GetChannelBalanceLogs(c *gin.Context) {
	channelIdStr := c.Query("channel_id")
	limitStr := c.DefaultQuery("limit", "50")

	limit, _ := strconv.Atoi(limitStr)
	channelId := 0
	if channelIdStr != "" {
		channelId, _ = strconv.Atoi(channelIdStr)
	}

	logs, err := model.GetBalanceLogs(channelId, limit)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	channelIds := make(map[int]string)
	for _, log := range logs {
		if _, ok := channelIds[log.ChannelId]; !ok {
			var channel model.Channel
			if err := common.DB.Select("name").First(&channel, log.ChannelId).Error; err == nil {
				channelIds[log.ChannelId] = channel.Name
			}
		}
	}

	result := make([]map[string]interface{}, len(logs))
	for i, log := range logs {
		result[i] = map[string]interface{}{
			"id":           log.Id,
			"channel_id":   log.ChannelId,
			"channel_name": channelIds[log.ChannelId],
			"balance":      log.Balance,
			"used":         log.Used,
			"quota":        log.Quota,
			"message":      log.Message,
			"created_at":   log.CreatedAt,
		}
	}

	common.OK(c, result)
}

func GetSchedulerStatus(c *gin.Context) {
	status := service.GetSchedulerStatus()
	common.OK(c, status)
}

func UpdateSchedulerConfig(c *gin.Context) {
	var req struct {
		CheckinCron          string `json:"checkin_cron"`
		BalanceRefreshCron   string `json:"balance_refresh_cron"`
		CheckinIntervalHours int    `json:"checkin_interval_hours"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "无效的请求参数")
		return
	}

	config := service.SchedulerConfig{
		CheckinCron:          req.CheckinCron,
		BalanceRefreshCron:   req.BalanceRefreshCron,
		CheckinIntervalHours: req.CheckinIntervalHours,
	}

	service.UpdateSchedulerConfig(config)

	common.OKMsg(c, "配置已更新", nil)
}

func StartScheduler(c *gin.Context) {
	service.StartScheduler()
	common.OKMsg(c, "调度器已启动", nil)
}

func StopScheduler(c *gin.Context) {
	service.StopScheduler()
	common.OKMsg(c, "调度器已停止", nil)
}

func UpdateChannelAccount(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.CodeParamError, "无效的渠道ID")
		return
	}

	var req struct {
		CheckinEnabled  int    `json:"checkin_enabled"`
		AccountUsername string `json:"account_username"`
		AccountPassword string `json:"account_password"`
		AccessToken     string `json:"access_token"`
		PlatformUserId  int    `json:"platform_user_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "无效的请求参数")
		return
	}

	var channel model.Channel
	if err := common.DB.First(&channel, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}

	updates := map[string]interface{}{}
	if req.CheckinEnabled >= 0 {
		updates["checkin_enabled"] = req.CheckinEnabled
	}
	if req.AccountUsername != "" {
		updates["account_username"] = req.AccountUsername
	}
	if req.AccountPassword != "" {
		updates["account_password"] = req.AccountPassword
	}
	if req.AccessToken != "" {
		updates["access_token"] = req.AccessToken
	}
	if req.PlatformUserId > 0 {
		updates["platform_user_id"] = req.PlatformUserId
	}

	if len(updates) > 0 {
		if err := common.DB.Model(&channel).Updates(updates).Error; err != nil {
			common.Fail(c, common.CodeServerError, "更新失败")
			return
		}
	}

	common.OKMsg(c, "更新成功", nil)
}

func TestChannelLogin(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.CodeParamError, "无效的渠道ID")
		return
	}

	var channel model.Channel
	if err := common.DB.First(&channel, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "渠道不存在")
		return
	}

	if channel.AccountUsername == "" || channel.AccountPassword == "" {
		common.Fail(c, common.CodeParamError, "未配置账号密码")
		return
	}

	adaptor := adapter.GetAccountAdaptor(channel.Type)
	if adaptor == nil {
		common.Fail(c, common.CodeParamError, "不支持的渠道类型")
		return
	}

	token, err := adaptor.Login(channel.BaseURL, channel.AccountUsername, channel.AccountPassword)
	if err != nil {
		common.OK(c, gin.H{
			"success": false,
			"message": "登录失败: " + err.Error(),
		})
		return
	}

	common.DB.Model(&channel).Update("access_token", token)

	common.OK(c, gin.H{
		"success": true,
		"message": "登录成功",
		"token":   token,
	})
}

func GetChannelBalanceStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.Fail(c, common.CodeParamError, "无效的渠道ID")
		return
	}

	result, err := service.GetChannelBalanceStatus(id)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	common.OK(c, result)
}
