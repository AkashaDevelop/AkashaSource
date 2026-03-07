package controller

import (
	"encoding/json"

	"STfreApi/common"

	"github.com/gin-gonic/gin"
)

const channelAffinityRulesOptionKey = "ChannelAffinityRules"

func GetChannelAffinityRules(c *gin.Context) {
	rulesJSON := common.OptionMap[channelAffinityRulesOptionKey]
	if rulesJSON == "" {
		common.OK(c, gin.H{"rules": []interface{}{}})
		return
	}

	var rules interface{}
	if err := json.Unmarshal([]byte(rulesJSON), &rules); err != nil {
		common.OK(c, gin.H{"rules": []interface{}{}})
		return
	}

	common.OK(c, gin.H{"rules": rules})
}

func SaveChannelAffinityRules(c *gin.Context) {
	var req struct {
		Rules interface{} `json:"rules"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}

	rulesJSON, err := json.Marshal(req.Rules)
	if err != nil {
		common.Fail(c, common.CodeServerError, "序列化失败")
		return
	}

	common.UpdateOptionMap(channelAffinityRulesOptionKey, string(rulesJSON))
	common.OKMsg(c, "保存成功", nil)
}

func GetChannelAffinityStats(c *gin.Context) {
	stats, err := getChannelAffinityCacheStats()
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}
	common.OK(c, stats)
}

func ClearChannelAffinityCacheAPI(c *gin.Context) {
	count, err := clearChannelAffinityCacheAll()
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}
	common.OK(c, gin.H{"count": count})
}
