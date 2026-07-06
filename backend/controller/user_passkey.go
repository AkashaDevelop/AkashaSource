package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"STfreApi/common"
	"STfreApi/model"
	passkeysvc "STfreApi/service/passkey"

	"github.com/gin-gonic/gin"
)

// PasskeyVerifyBegin 已登录用户用 Passkey 做二次验证（assertion flow），
// 与登录的 discoverable login 不同，这里针对特定用户的凭证发起 BeginLogin。
func PasskeyVerifyBegin(c *gin.Context) {
	userID := c.GetInt("id")
	if userID == 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}

	credential, err := model.GetPasskeyByUserID(userID)
	if err != nil {
		common.Fail(c, common.CodeNotFound, "该用户尚未绑定 Passkey")
		return
	}

	var user model.User
	if err := common.DB.First(&user, userID).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}

	wa, err := passkeysvc.BuildWebAuthn(c.Request)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	waUser := passkeysvc.NewWebAuthnUser(&user, credential)
	assertion, sessionData, err := wa.BeginLogin(waUser)
	if err != nil {
		common.Fail(c, common.CodeServerError, fmt.Sprintf("生成 Passkey 验证参数失败: %v", err))
		return
	}

	sessionID, err := passkeysvc.SaveVerifySession(sessionData)
	if err != nil {
		common.Fail(c, common.CodeServerError, "保存 Passkey 会话失败")
		return
	}

	common.OK(c, gin.H{"session_id": sessionID, "options": assertion})
}

// PasskeyVerifyFinish 完成 Passkey 二次验证，校验通过后仅返回成功，不重新发 token。
func PasskeyVerifyFinish(c *gin.Context) {
	userID := c.GetInt("id")
	if userID == 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}

	// Read body once, then restore it for FinishLogin to parse
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		common.Fail(c, common.CodeParamError, "读取请求失败")
		return
	}
	var req struct {
		SessionID string `json:"session_id" binding:"required"`
	}
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		common.Fail(c, common.CodeParamError, "缺少 session_id")
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	sessionData, err := passkeysvc.PopVerifySession(req.SessionID)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	credential, err := model.GetPasskeyByUserID(userID)
	if err != nil {
		common.Fail(c, common.CodeNotFound, "该用户尚未绑定 Passkey")
		return
	}

	var user model.User
	if err := common.DB.First(&user, userID).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}

	wa, err := passkeysvc.BuildWebAuthn(c.Request)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	waUser := passkeysvc.NewWebAuthnUser(&user, credential)
	updatedCred, err := wa.FinishLogin(waUser, *sessionData, c.Request)
	if err != nil {
		common.Fail(c, common.CodeUnauthorized, fmt.Sprintf("Passkey 验证失败: %v", err))
		return
	}

	// 更新凭证最后使用时间
	updated := model.NewPasskeyCredentialFromWebAuthn(userID, updatedCred)
	if updated != nil {
		now := time.Now()
		updated.LastUsedAt = &now
		_ = model.UpsertPasskeyCredential(updated)
	}

	common.OKMsg(c, "Passkey 验证成功", nil)
}
