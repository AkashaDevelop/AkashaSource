package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	passkeysvc "STfreApi/service/passkey"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	webauthnlib "github.com/go-webauthn/webauthn/webauthn"
)

type passkeyFinishRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

func PasskeyStatus(c *gin.Context) {
	userID := c.GetInt("id")
	credential, err := model.GetPasskeyByUserID(userID)
	if errors.Is(err, model.ErrPasskeyNotFound) {
		common.OK(c, gin.H{"enabled": false})
		return
	}
	if err != nil {
		common.Fail(c, common.CodeServerError, "查询 Passkey 状态失败")
		return
	}
	common.OK(c, gin.H{"enabled": true, "last_used_at": credential.LastUsedAt})
}

func PasskeyRegisterBegin(c *gin.Context) {
	userID := c.GetInt("id")
	if userID == 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}

	var user model.User
	if err := common.DB.First(&user, userID).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}

	credential, err := model.GetPasskeyByUserID(userID)
	if err != nil && !errors.Is(err, model.ErrPasskeyNotFound) {
		common.Fail(c, common.CodeServerError, "读取 Passkey 信息失败")
		return
	}
	if errors.Is(err, model.ErrPasskeyNotFound) {
		credential = nil
	}

	wa, err := passkeysvc.BuildWebAuthn(c.Request)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	waUser := passkeysvc.NewWebAuthnUser(&user, credential)
	creation, sessionData, err := wa.BeginRegistration(waUser)
	if err != nil {
		common.Fail(c, common.CodeServerError, fmt.Sprintf("生成 Passkey 注册参数失败: %v", err))
		return
	}

	sessionID, err := passkeysvc.SaveRegisterSession(sessionData)
	if err != nil {
		common.Fail(c, common.CodeServerError, "保存 Passkey 会话失败")
		return
	}

	common.OK(c, gin.H{"session_id": sessionID, "options": creation})
}

func PasskeyRegisterFinish(c *gin.Context) {
	userID := c.GetInt("id")
	if userID == 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}

	var req passkeyFinishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	var user model.User
	if err := common.DB.First(&user, userID).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}

	record, err := model.GetPasskeyByUserID(userID)
	if err != nil && !errors.Is(err, model.ErrPasskeyNotFound) {
		common.Fail(c, common.CodeServerError, "读取 Passkey 信息失败")
		return
	}
	if errors.Is(err, model.ErrPasskeyNotFound) {
		record = nil
	}

	sessionData, err := passkeysvc.PopRegisterSession(req.SessionID)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	wa, err := passkeysvc.BuildWebAuthn(c.Request)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	waUser := passkeysvc.NewWebAuthnUser(&user, record)
	credential, err := wa.FinishRegistration(waUser, *sessionData, c.Request)
	if err != nil {
		common.Fail(c, common.CodeServerError, fmt.Sprintf("Passkey 注册失败: %v", err))
		return
	}

	passkeyCredential := model.NewPasskeyCredentialFromWebAuthn(userID, credential)
	if passkeyCredential == nil {
		common.Fail(c, common.CodeServerError, "创建 Passkey 凭证失败")
		return
	}
	if err := model.UpsertPasskeyCredential(passkeyCredential); err != nil {
		common.Fail(c, common.CodeServerError, "保存 Passkey 凭证失败")
		return
	}

	common.OKMsg(c, "Passkey 注册成功", nil)
}

func PasskeyDelete(c *gin.Context) {
	userID := c.GetInt("id")
	if userID == 0 {
		common.Fail(c, common.CodeUnauthorized, "未登录")
		return
	}
	if err := model.DeletePasskeyByUserID(userID); err != nil {
		common.Fail(c, common.CodeServerError, "删除 Passkey 失败")
		return
	}
	common.OKMsg(c, "Passkey 已解绑", nil)
}

func PasskeyLoginBegin(c *gin.Context) {
	wa, err := passkeysvc.BuildWebAuthn(c.Request)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	assertion, sessionData, err := wa.BeginDiscoverableLogin()
	if err != nil {
		common.Fail(c, common.CodeServerError, fmt.Sprintf("生成 Passkey 登录参数失败: %v", err))
		return
	}

	sessionID, err := passkeysvc.SaveLoginSession(sessionData)
	if err != nil {
		common.Fail(c, common.CodeServerError, "保存 Passkey 会话失败")
		return
	}

	common.OK(c, gin.H{"session_id": sessionID, "options": assertion})
}

func PasskeyLoginFinish(c *gin.Context) {
	var req passkeyFinishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	sessionData, err := passkeysvc.PopLoginSession(req.SessionID)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	wa, err := passkeysvc.BuildWebAuthn(c.Request)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	handler := func(rawID, userHandle []byte) (webauthnlib.User, error) {
		credential, err := model.GetPasskeyByCredentialID(rawID)
		if err != nil {
			return nil, err
		}
		var user model.User
		if err := common.DB.First(&user, credential.UserID).Error; err != nil {
			return nil, err
		}
		if user.Status != model.UserStatusActive {
			return nil, fmt.Errorf("用户已禁用")
		}
		if len(userHandle) > 0 {
			if uid, parseErr := strconv.Atoi(string(userHandle)); parseErr == nil && uid != user.Id {
				return nil, fmt.Errorf("凭证用户不匹配")
			}
		}
		return passkeysvc.NewWebAuthnUser(&user, credential), nil
	}

	waUser, credential, err := wa.FinishPasskeyLogin(handler, *sessionData, c.Request)
	if err != nil {
		common.Fail(c, common.CodeUnauthorized, fmt.Sprintf("Passkey 登录失败: %v", err))
		return
	}

	userWrapper, ok := waUser.(*passkeysvc.WebAuthnUser)
	if !ok || userWrapper.ModelUser() == nil {
		common.Fail(c, common.CodeServerError, "Passkey 登录状态异常")
		return
	}
	user := userWrapper.ModelUser()

	updatedCredential := model.NewPasskeyCredentialFromWebAuthn(user.Id, credential)
	if updatedCredential != nil {
		now := time.Now()
		updatedCredential.LastUsedAt = &now
		_ = model.UpsertPasskeyCredential(updatedCredential)
	}

	token, err := common.GenerateToken(user.Id, user.Username, user.Role)
	if err != nil {
		common.Fail(c, common.CodeServerError, "生成令牌失败")
		return
	}

	common.OKMsg(c, "登录成功", gin.H{
		"token": token,
		"user":  gin.H{"id": user.Id, "username": user.Username, "role": user.Role, "quota": user.Quota},
	})
}
