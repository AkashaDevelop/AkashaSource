package controller

import (
	"encoding/json"
	"strconv"
	"time"

	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service/qingyuan"

	"github.com/gin-gonic/gin"
)

func GetContextSanitizationPolicies(c *gin.Context) {
	db := common.DB.Model(&model.ContextSanitizationPolicy{})
	if scope := c.Query("scope"); scope != "" {
		db = db.Where("scope = ?", scope)
	}
	if enabled := c.Query("enabled"); enabled != "" {
		db = db.Where("enabled = ?", enabled == "true")
	}
	if channelId := c.Query("channel_id"); channelId != "" {
		db = db.Where("channel_id = ?", channelId)
	}
	if modelName := c.Query("model_name"); modelName != "" {
		db = db.Where("model_name LIKE ?", "%"+modelName+"%")
	}
	var policies []model.ContextSanitizationPolicy
	if err := db.Order("id desc").Find(&policies).Error; err != nil {
		common.Fail(c, common.CodeServerError, "查询策略失败")
		return
	}
	common.OK(c, policies)
}

func CreateContextSanitizationPolicy(c *gin.Context) {
	var policy model.ContextSanitizationPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	now := time.Now().Unix()
	policy.Id = 0
	policy.Version = 1
	policy.CreatedAt = now
	policy.UpdatedAt = now
	if err := validatePolicyForSave(&policy, 0); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if err := common.DB.Create(&policy).Error; err != nil {
		common.Fail(c, common.CodeServerError, "创建策略失败")
		return
	}
	_ = qingyuan.ReloadPolicyCache()
	common.OK(c, policy)
}

func UpdateContextSanitizationPolicy(c *gin.Context) {
	var req model.ContextSanitizationPolicy
	if err := c.ShouldBindJSON(&req); err != nil || req.Id == 0 {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	var old model.ContextSanitizationPolicy
	if err := common.DB.First(&old, req.Id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "策略不存在")
		return
	}
	req.CreatedAt = old.CreatedAt
	req.Version = old.Version + 1
	req.UpdatedAt = time.Now().Unix()
	if err := validatePolicyForSave(&req, req.Id); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if err := savePolicyRevision(c, &old, "update"); err != nil {
		common.Fail(c, common.CodeServerError, "保存策略版本失败")
		return
	}
	if err := common.DB.Save(&req).Error; err != nil {
		common.Fail(c, common.CodeServerError, "更新策略失败")
		return
	}
	_ = qingyuan.ReloadPolicyCache()
	common.OK(c, req)
}

func DeleteContextSanitizationPolicy(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id == 0 {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	var policy model.ContextSanitizationPolicy
	if err := common.DB.First(&policy, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "策略不存在")
		return
	}
	_ = savePolicyRevision(c, &policy, "delete")
	if err := common.DB.Delete(&policy).Error; err != nil {
		common.Fail(c, common.CodeServerError, "删除策略失败")
		return
	}
	_ = qingyuan.ReloadPolicyCache()
	common.OKMsg(c, "删除成功", nil)
}

func GetContextSanitizationPolicyRevisions(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var revisions []model.ContextSanitizationPolicyRevision
	if err := common.DB.Where("policy_id = ?", id).Order("id desc").Find(&revisions).Error; err != nil {
		common.Fail(c, common.CodeServerError, "查询版本失败")
		return
	}
	common.OK(c, revisions)
}

func RollbackContextSanitizationPolicy(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		RevisionId int `json:"revision_id"`
		Version    int `json:"version"`
	}
	_ = c.ShouldBindJSON(&req)
	var revision model.ContextSanitizationPolicyRevision
	db := common.DB.Where("policy_id = ?", id)
	if req.RevisionId > 0 {
		db = db.Where("id = ?", req.RevisionId)
	} else if req.Version > 0 {
		db = db.Where("version = ?", req.Version)
	} else {
		common.Fail(c, common.CodeParamError, "请选择要回滚的版本")
		return
	}
	if err := db.First(&revision).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "版本不存在")
		return
	}
	var current model.ContextSanitizationPolicy
	if err := common.DB.First(&current, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "策略不存在")
		return
	}
	var restored model.ContextSanitizationPolicy
	if err := json.Unmarshal([]byte(revision.Snapshot), &restored); err != nil {
		common.Fail(c, common.CodeServerError, "版本快照解析失败")
		return
	}
	_ = savePolicyRevision(c, &current, "rollback_before")
	restored.Id = current.Id
	restored.Version = current.Version + 1
	restored.UpdatedAt = time.Now().Unix()
	if err := common.DB.Save(&restored).Error; err != nil {
		common.Fail(c, common.CodeServerError, "回滚失败")
		return
	}
	_ = qingyuan.ReloadPolicyCache()
	common.OK(c, restored)
}

func ReloadContextSanitizationPolicyCache(c *gin.Context) {
	if err := qingyuan.ReloadPolicyCache(); err != nil {
		common.Fail(c, common.CodeServerError, "刷新缓存失败")
		return
	}
	common.OK(c, qingyuan.CacheStats())
}

func GetContextSanitizationEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	db := common.DB.Model(&model.ContextSanitizationEvent{})
	for _, key := range []string{"direction", "stage", "action", "mode", "channel_id", "policy_id", "user_id", "token_name"} {
		if v := c.Query(key); v != "" {
			db = db.Where(key+" = ?", v)
		}
	}
	if modelName := c.Query("model"); modelName != "" {
		db = db.Where("requested_model LIKE ? OR mapped_model LIKE ?", "%"+modelName+"%", "%"+modelName+"%")
	}
	var total int64
	db.Count(&total)
	var events []model.ContextSanitizationEvent
	if err := db.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&events).Error; err != nil {
		common.Fail(c, common.CodeServerError, "查询事件失败")
		return
	}
	common.OK(c, gin.H{"total": total, "items": events})
}

func GetContextSanitizationStats(c *gin.Context) {
	var policyCount int64
	var eventCount int64
	common.DB.Model(&model.ContextSanitizationPolicy{}).Count(&policyCount)
	common.DB.Model(&model.ContextSanitizationEvent{}).Count(&eventCount)
	common.OK(c, gin.H{"policy_count": policyCount, "event_count": eventCount, "cache": qingyuan.CacheStats(), "circuit": qingyuan.CircuitStats()})
}

func validatePolicyForSave(policy *model.ContextSanitizationPolicy, ignoreId int) error {
	if err := qingyuan.ValidatePolicy(policy); err != nil {
		return err
	}
	db := common.DB.Model(&model.ContextSanitizationPolicy{}).Where("scope = ?", policy.Scope)
	if ignoreId > 0 {
		db = db.Where("id <> ?", ignoreId)
	}
	switch policy.Scope {
	case model.ContextSanitizationScopeGlobal:
		db = db.Where("model_name = '' OR model_name IS NULL")
	case model.ContextSanitizationScopeModel:
		db = db.Where("model_name = ?", policy.ModelName)
	case model.ContextSanitizationScopeChannel:
		db = db.Where("channel_id = ? AND (model_name = '' OR model_name IS NULL)", policy.ChannelId)
	case model.ContextSanitizationScopeChannelModel:
		db = db.Where("channel_id = ? AND model_name = ?", policy.ChannelId, policy.ModelName)
	}
	var count int64
	db.Count(&count)
	if count > 0 {
		return commonError("相同范围的策略已存在")
	}
	return nil
}

func savePolicyRevision(c *gin.Context, policy *model.ContextSanitizationPolicy, note string) error {
	snapshot, _ := json.Marshal(policy)
	uid := 0
	if v, ok := c.Get("id"); ok {
		if n, ok := v.(int); ok {
			uid = n
		}
	}
	return common.DB.Create(&model.ContextSanitizationPolicyRevision{PolicyId: policy.Id, Version: policy.Version, Snapshot: string(snapshot), CreatedAt: time.Now().Unix(), CreatedBy: uid, ChangeNote: note}).Error
}

type commonError string

func (e commonError) Error() string { return string(e) }
