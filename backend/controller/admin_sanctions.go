package controller

import (
	"strconv"
	"strings"

	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service/sanction"

	"github.com/gin-gonic/gin"
)

// AdminListSanctions 列出制裁记录（可按 target_type 过滤）
// GET /api/admin/sanctions?target_type=token|user|ip
func AdminListSanctions(c *gin.Context) {
	targetType := c.Query("target_type")
	list, err := sanction.List(targetType)
	if err != nil {
		common.Fail(c, common.CodeServerError, "查询制裁记录失败")
		return
	}
	common.OK(c, gin.H{"list": list, "total": len(list)})
}

// AdminCreateSanction 管理员手动施加一条制裁
// POST /api/admin/sanctions
func AdminCreateSanction(c *gin.Context) {
	var req struct {
		TargetType      string  `json:"target_type"`
		TargetKey       string  `json:"target_key"`
		Action          string  `json:"action"`
		Factor          float64 `json:"factor"`
		Reason          string  `json:"reason"`
		DurationMinutes int     `json:"duration_minutes"` // 0 = 永久
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数解析失败")
		return
	}

	req.TargetType = strings.TrimSpace(req.TargetType)
	req.TargetKey = strings.TrimSpace(req.TargetKey)
	req.Action = strings.TrimSpace(req.Action)

	if !validSanctionTarget(req.TargetType) {
		common.Fail(c, common.CodeParamError, "无效的处置目标类型")
		return
	}
	if req.TargetKey == "" {
		common.Fail(c, common.CodeParamError, "处置目标不能为空")
		return
	}
	if !validSanctionAction(req.Action) {
		common.Fail(c, common.CodeParamError, "无效的处置动作")
		return
	}
	// target/action 组合合法性：ban_ip 只能作用于 ip，ban_user/billing_penalty(user) 作用于 user 等
	if !targetActionMatch(req.TargetType, req.Action) {
		common.Fail(c, common.CodeParamError, "处置动作与目标类型不匹配")
		return
	}
	// token/user 目标必须是数字 ID
	if req.TargetType != model.SanctionTargetIP {
		if _, err := strconv.Atoi(req.TargetKey); err != nil {
			common.Fail(c, common.CodeParamError, "token/user 目标必须是数字 ID")
			return
		}
	}

	reason := req.Reason
	if reason == "" {
		reason = "管理员手动处置"
	}
	if err := sanction.Apply(req.TargetType, req.TargetKey, req.Action, req.Factor, reason, "admin_manual", req.DurationMinutes); err != nil {
		common.Fail(c, common.CodeServerError, "施加处置失败: "+err.Error())
		return
	}

	// 破坏性处置需要即时同步更新 Token/User 状态（制裁表只是登记，实际状态另存）
	syncDestructiveState(req.TargetType, req.TargetKey, req.Action)

	common.OKMsg(c, "处置已生效", nil)
}

// AdminRevokeSanction 解除一条制裁
// DELETE /api/admin/sanctions/:id
func AdminRevokeSanction(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.Fail(c, common.CodeParamError, "无效的制裁 ID")
		return
	}

	// 解除前先查出记录，破坏性处置解除时要顺带恢复 Token/User 状态
	list, _ := sanction.List("")
	var target *model.Sanction
	for i := range list {
		if list[i].Id == id {
			target = &list[i]
			break
		}
	}

	if err := sanction.Revoke(id); err != nil {
		common.Fail(c, common.CodeServerError, "解除处置失败: "+err.Error())
		return
	}

	if target != nil {
		restoreDestructiveState(target.TargetType, target.TargetKey, target.Action)
	}

	common.OKMsg(c, "处置已解除", nil)
}

func validSanctionTarget(t string) bool {
	switch t {
	case model.SanctionTargetToken, model.SanctionTargetUser, model.SanctionTargetIP:
		return true
	}
	return false
}

func validSanctionAction(a string) bool {
	switch a {
	case model.SanctionThrottle, model.SanctionRPMLimit, model.SanctionBillingPenalty,
		model.SanctionSuspendToken, model.SanctionDisableToken, model.SanctionBanIP, model.SanctionBanUser:
		return true
	}
	return false
}

// targetActionMatch 校验处置动作与目标类型的合法组合
func targetActionMatch(target, action string) bool {
	switch action {
	case model.SanctionBanIP:
		return target == model.SanctionTargetIP
	case model.SanctionBanUser:
		return target == model.SanctionTargetUser
	case model.SanctionThrottle, model.SanctionRPMLimit, model.SanctionSuspendToken, model.SanctionDisableToken:
		return target == model.SanctionTargetToken
	case model.SanctionBillingPenalty:
		// 计费惩罚支持 user（整个账号）和 token（单个令牌）两级
		return target == model.SanctionTargetUser || target == model.SanctionTargetToken
	}
	return false
}

// syncDestructiveState 破坏性处置即时同步 Token/User 状态
func syncDestructiveState(target, key, action string) {
	id, err := strconv.Atoi(key)
	if err != nil {
		return
	}
	switch action {
	case model.SanctionDisableToken, model.SanctionSuspendToken:
		common.DB.Model(&model.Token{}).Where("id = ?", id).Update("status", model.TokenStatusDisabled)
	case model.SanctionBanUser:
		common.DB.Model(&model.User{}).Where("id = ?", id).Update("status", model.UserStatusBanned)
	}
}

// restoreDestructiveState 解除破坏性处置时恢复 Token/User 状态
func restoreDestructiveState(target, key, action string) {
	id, err := strconv.Atoi(key)
	if err != nil {
		return
	}
	switch action {
	case model.SanctionDisableToken, model.SanctionSuspendToken:
		common.DB.Model(&model.Token{}).Where("id = ? AND status = ?", id, model.TokenStatusDisabled).
			Update("status", model.TokenStatusActive)
	case model.SanctionBanUser:
		common.DB.Model(&model.User{}).Where("id = ? AND status = ?", id, model.UserStatusBanned).
			Update("status", model.UserStatusActive)
	}
}
