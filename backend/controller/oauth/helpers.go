package oauth

import (
	"STfreApi/common"
	"STfreApi/model"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
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

// verifyState validates and consumes a state token (one-time use)
func verifyState(state string) bool {
	if state == "" {
		return false
	}
	if common.RedisClient != nil {
		var ok bool
		if !common.GetCache("oauth_state:"+state, &ok) {
			return false
		}
		return ok
	}
	val, loaded := stateStore.LoadAndDelete(state)
	if !loaded {
		return false
	}
	return time.Now().Before(val.(stateEntry).expires)
}

// createOAuthUser finds an existing user by OAuth ID or creates a new one.
// idField: DB column name (e.g. "github_id")
// idValue: provider user ID string
// buildUser: returns a skeleton User (Quota will be overwritten by new-user reward)
// invitationCode: optional invitation code from query param "aff"
func createOAuthUser(idField, idValue string, buildUser func() model.User, invitationCode string) (*model.User, error) {
	var user model.User
	if err := common.DB.Where(idField+" = ?", idValue).First(&user).Error; err == nil {
		if user.Status == model.UserStatusBanned {
			return nil, fmt.Errorf("账号已被封禁")
		}
		return &user, nil
	}

	// New user — check invitation gate
	invitationEnabled := common.OptionMap[model.OptionKeyInvitationEnabled] == "true"
	var invitation model.Invitation
	if invitationEnabled {
		if invitationCode == "" {
			return nil, fmt.Errorf("注册需要邀请码")
		}
		if err := common.DB.Where("code = ? AND status = ?", invitationCode, model.InvitationStatusUnused).First(&invitation).Error; err != nil {
			return nil, fmt.Errorf("邀请码无效或已使用")
		}
	} else if invitationCode != "" {
		common.DB.Where("code = ? AND status = ?", invitationCode, model.InvitationStatusUnused).First(&invitation)
	}
	if invitation.Id > 0 && invitation.MaxUses > 0 && invitation.UsedCount >= invitation.MaxUses {
		if invitationEnabled {
			return nil, fmt.Errorf("邀请码已达使用上限")
		}
		invitation = model.Invitation{}
	}

	newUser := buildUser()
	newUserReward, _ := strconv.ParseFloat(common.OptionMap[model.OptionKeyNewUserReward], 64)
	newUser.Quota = int64(newUserReward)

	err := common.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&newUser).Error; err != nil {
			return err
		}
		if invitation.Id > 0 {
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
					tx.Model(&inviter).Update("quota", gorm.Expr("quota + ?", int64(invitationReward)))
					tx.Create(&model.Log{
						UserId:    inviter.Id,
						Username:  inviter.Username,
						CreatedAt: time.Now().Unix(),
						Type:      model.LogTypeSystem,
						Content:   fmt.Sprintf("邀请奖励: %s", newUser.Username),
						Quota:     int64(invitationReward),
						ModelName: "system",
					})
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &newUser, nil
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
