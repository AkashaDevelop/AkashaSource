package oauth

import (
	"STfreApi/common"
	"STfreApi/model"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// stateEntry holds an in-memory OAuth state with expiry (used when Redis is unavailable)
type stateEntry struct {
	expires time.Time
}

var stateStore sync.Map

// generateState creates a random CSRF state token and persists it (Redis or memory)
func generateState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	state := hex.EncodeToString(b)

	if common.RedisClient != nil {
		common.SetCache("oauth_state:"+state, true, 5*time.Minute)
	} else {
		stateStore.Store(state, stateEntry{expires: time.Now().Add(5 * time.Minute)})
	}
	return state
}

// GenerateStateToken exports OAuth state generation for unified API endpoint.
func GenerateStateToken() string {
	return generateState()
}

// verifyState validates and consumes a state token (one-time use)
func verifyState(state string) bool {
	if state == "" {
		return false
	}
	if common.RedisClient != nil {
		// Redis: use DEL to atomically consume (one-time use)
		var ok bool
		if !common.GetCache("oauth_state:"+state, &ok) {
			return false
		}
		common.RedisClient.Del(context.Background(), "oauth_state:"+state)
		return ok
	}
	val, loaded := stateStore.LoadAndDelete(state)
	if !loaded {
		return false
	}
	return time.Now().Before(val.(stateEntry).expires)
}

// pendingOAuthTTL is how long a deferred OAuth registration may wait for an
// invitation code before the session expires.
const pendingOAuthTTL = 10 * time.Minute

// PendingOAuthData holds temporary OAuth data between login and invitation-code
// entry.
type PendingOAuthData struct {
	IDField  string
	IDValue  string
	User     model.User // skeleton user from buildUser()
	Provider string
	Expires  time.Time
}

var (
	pendingOAuthSessions = make(map[string]PendingOAuthData)
	pendingOAuthMu       sync.RWMutex
)

func savePendingOAuthSession(data PendingOAuthData) string {
	id := uuid.New().String()
	pendingOAuthMu.Lock()
	defer pendingOAuthMu.Unlock()
	// Clean expired
	for k, v := range pendingOAuthSessions {
		if time.Now().After(v.Expires) {
			delete(pendingOAuthSessions, k)
		}
	}
	pendingOAuthSessions[id] = data
	return id
}

func getPendingOAuthSession(id string) (PendingOAuthData, bool) {
	pendingOAuthMu.RLock()
	defer pendingOAuthMu.RUnlock()
	data, ok := pendingOAuthSessions[id]
	if !ok || time.Now().After(data.Expires) {
		return PendingOAuthData{}, false
	}
	return data, true
}

func deletePendingOAuthSession(id string) {
	pendingOAuthMu.Lock()
	defer pendingOAuthMu.Unlock()
	delete(pendingOAuthSessions, id)
}

// createOAuthUser finds an existing user by OAuth ID or creates a new one.
// If the user is new and invitation codes are enabled, it does NOT create the
// user yet; instead it stores the pending OAuth data and returns a pending
// session ID. The caller should then redirect to the invitation-code entry page.
// idField: DB column name (e.g. "github_id")
// idValue: provider user ID string
// buildUser: returns a skeleton User (Quota will be overwritten by new-user reward)
// provider: OAuth provider name (e.g. "github")
func createOAuthUser(idField, idValue string, buildUser func() model.User, provider string) (*model.User, string, error) {
	if !common.RegisterEnabled {
		return nil, "", fmt.Errorf("注册已关闭")
	}
	var user model.User
	if err := common.DB.Where(idField+" = ?", idValue).First(&user).Error; err == nil {
		if user.Status == model.UserStatusBanned {
			return nil, "", fmt.Errorf("账号已被封禁")
		}
		return &user, "", nil
	}

	// New user
	invitationEnabled := common.OptionMap[model.OptionKeyInvitationEnabled] == "true"
	if !invitationEnabled {
		// Auto-register without invitation
		newUser := buildUser()
		newUserReward, _ := strconv.ParseFloat(common.OptionMap[model.OptionKeyNewUserReward], 64)
		newUser.Quota = int64(newUserReward)
		if err := common.DB.Create(&newUser).Error; err != nil {
			return nil, "", err
		}
		return &newUser, "", nil
	}

	// Invitation required — defer registration until the user enters a code
	sessionID := savePendingOAuthSession(PendingOAuthData{
		IDField:  idField,
		IDValue:  idValue,
		User:     buildUser(),
		Provider: provider,
		Expires:  time.Now().Add(pendingOAuthTTL),
	})
	return nil, sessionID, nil
}

// completeOAuthRegistration validates the invitation code and finishes creating
// the user that was deferred during the OAuth callback.
func completeOAuthRegistration(sessionID, invitationCode string) (*model.User, error) {
	data, ok := getPendingOAuthSession(sessionID)
	if !ok {
		return nil, fmt.Errorf("OAuth 会话不存在或已过期，请重新登录")
	}

	if invitationCode == "" {
		return nil, fmt.Errorf("注册需要邀请码")
	}
	var invitation model.Invitation
	if err := common.DB.Where("code = ? AND status = ?", invitationCode, model.InvitationStatusUnused).First(&invitation).Error; err != nil {
		return nil, fmt.Errorf("邀请码无效或已使用")
	}
	if invitation.MaxUses > 0 && invitation.UsedCount >= invitation.MaxUses {
		return nil, fmt.Errorf("邀请码已达使用上限")
	}

	newUser := data.User
	newUserReward, _ := strconv.ParseFloat(common.OptionMap[model.OptionKeyNewUserReward], 64)
	newUser.Quota = int64(newUserReward)

	err := common.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&newUser).Error; err != nil {
			return err
		}
		invitation.UsedCount++
		invitation.InviteeId = newUser.Id
		invitation.UsedAt = time.Now().Unix()
		if invitation.MaxUses > 0 && invitation.UsedCount >= invitation.MaxUses {
			invitation.Status = model.InvitationStatusUsed
		}
		if err := tx.Save(&invitation).Error; err != nil {
			return err
		}
		if err := tx.Model(&newUser).Update("inviter_id", invitation.InviterId).Error; err != nil {
			return err
		}
		invitationReward, _ := strconv.ParseFloat(common.OptionMap[model.OptionKeyInvitationReward], 64)
		if invitationReward > 0 {
			var inviter model.User
			if err := tx.First(&inviter, invitation.InviterId).Error; err == nil {
				tx.Model(&inviter).Update("aff_quota", gorm.Expr("aff_quota + ?", int64(invitationReward)))
				tx.Create(&model.Log{
					UserId:    inviter.Id,
					Username:  inviter.Username,
					CreatedAt: time.Now().Unix(),
					Type:      model.LogTypeSystem,
					Content:   fmt.Sprintf("邀请返利（待转账）: %s", newUser.Username),
					Quota:     int64(invitationReward),
					ModelName: "system",
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	deletePendingOAuthSession(sessionID)
	return &newUser, nil
}

// CompleteOAuthRegistration handles the invitation-code submission that finishes
// a deferred OAuth registration.
func CompleteOAuthRegistration(c *gin.Context) {
	var req struct {
		SessionID      string `json:"session_id" binding:"required"`
		InvitationCode string `json:"invitation_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}

	user, err := completeOAuthRegistration(req.SessionID, req.InvitationCode)
	if err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	token, err := common.GenerateToken(user.Id, user.Username, user.Role)
	if err != nil {
		common.Fail(c, common.CodeServerError, "生成令牌失败")
		return
	}

	common.OKMsg(c, "注册成功", gin.H{
		"token": token,
		"user":  gin.H{"id": user.Id, "username": user.Username, "role": user.Role, "quota": user.Quota},
	})
}

// CheckOAuthPending validates that a pending OAuth session still exists.
func CheckOAuthPending(c *gin.Context) {
	sessionID := c.Query("session_id")
	_, ok := getPendingOAuthSession(sessionID)
	if !ok {
		common.Fail(c, common.CodeParamError, "会话不存在或已过期")
		return
	}
	common.OK(c, gin.H{"valid": true})
}

// oauthRedirect issues a JWT and redirects to the frontend
func oauthRedirect(c *gin.Context, user *model.User) {
	token, err := common.GenerateToken(user.Id, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Failed to generate token"})
		return
	}
	c.Redirect(http.StatusFound, fmt.Sprintf("/?token=%s", token))
}
