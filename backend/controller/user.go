package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type LoginRequest struct {
	Username   string                         `json:"username" binding:"required"`
	Password   string                         `json:"password" binding:"required"`
	Turnstile  string                         `json:"turnstile"`
	GeeTest    *common.GeeTestValidateRequest `json:"geetest"`
	HCaptcha   string                         `json:"hcaptcha"`
	ReCaptcha  string                         `json:"recaptcha"`
}

func UserLogin(c *gin.Context) {
	if !common.PasswordLoginEnabled {
		common.Fail(c, common.CodeForbidden, "密码登录已关闭")
		return
	}
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if !common.VerifyCaptcha(req.Turnstile, req.HCaptcha, req.ReCaptcha, req.GeeTest) {
		common.Fail(c, common.CodeParamError, "人机验证失败")
		return
	}
	var user model.User
	if err := common.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		common.Fail(c, common.CodeUnauthorized, "用户名或密码错误")
		return
	}
	if !common.ValidatePassword(req.Password, user.Password) {
		common.Fail(c, common.CodeUnauthorized, "用户名或密码错误")
		return
	}
	if user.TOTPEnabled {
		common.OK(c, gin.H{"requires_2fa": true, "user_id": user.Id, "msg": "请输入两步验证码"})
		return
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

type RegisterRequest struct {
	Username       string                         `json:"username" binding:"required"`
	Password       string                         `json:"password" binding:"required"`
	Email          string                         `json:"email"`
	EmailCode      string                         `json:"email_code"`
	Turnstile      string                         `json:"turnstile"`
	GeeTest        *common.GeeTestValidateRequest `json:"geetest"`
	HCaptcha       string                         `json:"hcaptcha"`
	ReCaptcha      string                         `json:"recaptcha"`
	InvitationCode string                         `json:"invitation_code"`
}

func UserRegister(c *gin.Context) {
	if !common.RegisterEnabled {
		common.Fail(c, common.CodeForbidden, "注册已关闭")
		return
	}
	if !common.PasswordRegisterEnabled {
		common.Fail(c, common.CodeForbidden, "密码注册已关闭，请使用第三方登录")
		return
	}
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if !common.VerifyCaptcha(req.Turnstile, req.HCaptcha, req.ReCaptcha, req.GeeTest) {
		common.Fail(c, common.CodeParamError, "人机验证失败")
		return
	}
	if common.EmailVerificationEnabled {
		if req.Email == "" {
			common.Fail(c, common.CodeParamError, "请填写邮箱")
			return
		}
		if !CheckEmailVerifyCode(req.Email, req.EmailCode) {
			common.Fail(c, common.CodeParamError, "邮箱验证码无效或已过期")
			return
		}
	}
	if common.EmailDomainRestrictionEnabled && req.Email != "" {
		allowed := false
		for _, domain := range strings.Split(common.EmailDomainWhitelist, ",") {
			domain = strings.TrimSpace(domain)
			if domain != "" && strings.HasSuffix(req.Email, "@"+domain) {
				allowed = true
				break
			}
		}
		if !allowed {
			common.Fail(c, common.CodeParamError, "该邮箱域名不在允许范围内")
			return
		}
	}
	invitationEnabled := common.OptionMap[model.OptionKeyInvitationEnabled] == "true"
	var invitation model.Invitation
	if invitationEnabled {
		if req.InvitationCode == "" {
			common.Fail(c, common.CodeParamError, "注册需要邀请码")
			return
		}
		if err := common.DB.Where("code = ? AND status = ?", req.InvitationCode, model.InvitationStatusUnused).First(&invitation).Error; err != nil {
			common.Fail(c, common.CodeParamError, "邀请码无效或已使用")
			return
		}
	} else if req.InvitationCode != "" {
		common.DB.Where("code = ? AND status = ?", req.InvitationCode, model.InvitationStatusUnused).First(&invitation)
	}
	// Extra check: max_uses
	if invitation.Id > 0 && invitation.MaxUses > 0 && invitation.UsedCount >= invitation.MaxUses {
		if invitationEnabled {
			common.Fail(c, common.CodeParamError, "邀请码已达使用上限")
			return
		}
		invitation = model.Invitation{}
	}
	var existingUser model.User
	if err := common.DB.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		common.Fail(c, common.CodeConflict, "用户名已存在")
		return
	}
	hashedPassword, err := common.Password2Hash(req.Password)
	if err != nil {
		common.Fail(c, common.CodeServerError, "密码加密失败")
		return
	}
	user := model.User{
		Username: req.Username, Password: hashedPassword, Email: req.Email,
		DisplayName: req.Username, Role: model.RoleUser, Status: model.UserStatusActive,
	}
	newUserReward, _ := strconv.ParseFloat(common.OptionMap[model.OptionKeyNewUserReward], 64)
	user.Quota = int64(newUserReward)
	var count int64
	common.DB.Model(&model.User{}).Count(&count)
	if count == 0 {
		user.Role = model.RoleRoot
	}
	err = common.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if invitation.Id > 0 {
			invitation.UsedCount++
			invitation.InviteeId = user.Id
			invitation.UsedAt = time.Now().Unix()
			if invitation.MaxUses > 0 && invitation.UsedCount >= invitation.MaxUses {
				invitation.Status = model.InvitationStatusUsed
			}
			if err := tx.Save(&invitation).Error; err != nil {
				return err
			}
			if err := tx.Model(&user).Update("inviter_id", invitation.InviterId).Error; err != nil {
				return err
			}
			invitationReward, _ := strconv.ParseFloat(common.OptionMap[model.OptionKeyInvitationReward], 64)
			if invitationReward > 0 {
				var inviter model.User
				if err := tx.First(&inviter, invitation.InviterId).Error; err == nil {
					tx.Model(&inviter).Update("aff_quota", gorm.Expr("aff_quota + ?", int64(invitationReward)))
					tx.Create(&model.Log{
						UserId: inviter.Id, Username: inviter.Username,
						CreatedAt: time.Now().Unix(), Type: model.LogTypeSystem,
						Content: fmt.Sprintf("邀请返利（待转账）: %s", user.Username),
						Quota: int64(invitationReward), ModelName: "system",
					})
				}
			}
		}
		return nil
	})
	if err != nil {
		common.Failf(c, common.CodeServerError, "注册失败: %s", err.Error())
		return
	}
	common.OKMsg(c, "注册成功", nil)
}

func SearchUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	var users []model.User
	db := common.DB.Order("id desc")
	if keyword != "" {
		db = db.Where("username LIKE ? OR display_name LIKE ? OR email LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if err := db.Limit(20).Find(&users).Error; err != nil {
		common.Fail(c, common.CodeServerError, "搜索失败")
		return
	}
	for i := range users { users[i].Password = "" }
	common.OK(c, users)
}

func GetAllUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	var total int64
	common.DB.Model(&model.User{}).Count(&total)
	var users []model.User
	if err := common.DB.Limit(size).Offset((page-1)*size).Order("id desc").Find(&users).Error; err != nil {
		common.Fail(c, common.CodeServerError, "获取用户列表失败")
		return
	}
	for i := range users {
		users[i].Password = ""
	}
	common.OK(c, gin.H{"data": users, "total": total})
}

func AddUser(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	if user.Status == 0 {
		user.Status = model.UserStatusActive
	}
	if err := user.Insert(); err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}
	common.OKMsg(c, "用户创建成功", user)
}

func UpdateUser(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	var existing model.User
	if err := common.DB.First(&existing, user.Id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}
	existing.Username = user.Username
	existing.DisplayName = user.DisplayName
	existing.Role = user.Role
	existing.Status = user.Status
	existing.Quota = user.Quota
	existing.Group = user.Group
	existing.Email = user.Email
	if user.Password != "" {
		if hashed, err := common.Password2Hash(user.Password); err == nil {
			existing.Password = hashed
		}
	}
	if err := common.DB.Save(&existing).Error; err != nil {
		common.Fail(c, common.CodeServerError, "更新用户失败")
		return
	}
	common.OKMsg(c, "用户更新成功", nil)
}

func DeleteUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := common.DB.Delete(&model.User{}, id).Error; err != nil {
		common.Fail(c, common.CodeServerError, "删除用户失败")
		return
	}
	common.OKMsg(c, "用户已删除", nil)
}

func ToggleUserStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var user model.User
	if err := common.DB.First(&user, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}
	newStatus := 1
	if user.Status == 1 {
		newStatus = 2
	}
	common.DB.Model(&user).Update("status", newStatus)
	common.OK(c, gin.H{"status": newStatus})
}

func AdjustUserQuota(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Delta int64 `json:"delta" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	var user model.User
	if err := common.DB.First(&user, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}
	newQuota := user.Quota + req.Delta
	if newQuota < 0 {
		newQuota = 0
	}
	common.DB.Model(&user).Update("quota", newQuota)
	common.OK(c, gin.H{"quota": newQuota})
}

func GetSelf(c *gin.Context) {
	userId, _ := c.Get("id")
	var user model.User
	if err := common.DB.First(&user, userId).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}
	user.Password = ""
	common.OK(c, user)
}

func UpdateSelf(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	userId, _ := c.Get("id")
	var existing model.User
	if err := common.DB.First(&existing, userId).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "用户不存在")
		return
	}
	existing.DisplayName = user.DisplayName
	existing.Email = user.Email
	if user.Password != "" {
		if hashed, err := common.Password2Hash(user.Password); err == nil {
			existing.Password = hashed
		}
	}
	if err := common.DB.Save(&existing).Error; err != nil {
		common.Fail(c, common.CodeServerError, "更新资料失败")
		return
	}
	common.OKMsg(c, "资料更新成功", nil)
}
