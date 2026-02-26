package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type LoginRequest struct {
	Username  string                          `json:"username" binding:"required"`
	Password  string                          `json:"password" binding:"required"`
	Turnstile string                          `json:"turnstile"`
	GeeTest   *common.GeeTestValidateRequest  `json:"geetest"`
}

func UserLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !common.VerifyCaptcha(req.Turnstile, req.GeeTest) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "人机验证失败"})
		return
	}

	var user model.User
	if err := common.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	if !common.ValidatePassword(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// Check if 2FA is enabled
	if user.TOTPEnabled {
		c.JSON(http.StatusOK, gin.H{
			"requires_2fa": true,
			"user_id":      user.Id,
			"message":      "请输入两步验证码",
		})
		return
	}

	// Generate JWT
	token, err := common.GenerateToken(user.Id, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
		"user": gin.H{
			"id":       user.Id,
			"username": user.Username,
			"role":     user.Role,
			"quota":    user.Quota,
		},
	})
}

type RegisterRequest struct {
	Username       string                          `json:"username" binding:"required"`
	Password       string                          `json:"password" binding:"required"`
	Email          string                          `json:"email"`
	Turnstile      string                          `json:"turnstile"`
	GeeTest        *common.GeeTestValidateRequest  `json:"geetest"`
	InvitationCode string                          `json:"invitation_code"`
}

func UserRegister(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !common.VerifyCaptcha(req.Turnstile, req.GeeTest) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "人机验证失败"})
		return
	}

	// Check Invitation Code if enabled
	invitationEnabled := common.OptionMap[model.OptionKeyInvitationEnabled] == "true"
	var invitation model.Invitation
	if invitationEnabled {
		if req.InvitationCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invitation code is required"})
			return
		}
		if err := common.DB.Where("code = ? AND status = ?", req.InvitationCode, model.InvitationStatusUnused).First(&invitation).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or used invitation code"})
			return
		}
	} else if req.InvitationCode != "" {
		// Optional invitation code logic
		common.DB.Where("code = ? AND status = ?", req.InvitationCode, model.InvitationStatusUnused).First(&invitation)
	}

	// Check if username exists
	var existingUser model.User
	if err := common.DB.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username already exists"})
		return
	}

	user := model.User{
		Username:    req.Username,
		Password:    req.Password,
		Email:       req.Email,
		DisplayName: req.Username,
		Role:        model.RoleUser,
		Status:      model.UserStatusActive,
		Quota:       0,
	}

	// New User Reward
	newUserRewardStr := common.OptionMap[model.OptionKeyNewUserReward]
	newUserReward, _ := strconv.ParseFloat(newUserRewardStr, 64)
	user.Quota = int64(newUserReward)

	// Check if this is the first user
	var count int64
	common.DB.Model(&model.User{}).Count(&count)
	if count == 0 {
		user.Role = model.RoleRoot // First user is Root
		user.Status = model.UserStatusActive
	}

	// Transaction for registration and invitation processing
	err := common.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		// Process Invitation
		if invitation.Id > 0 {
			// Mark as used
			invitation.Status = model.InvitationStatusUsed
			invitation.InviteeId = user.Id
			invitation.UsedAt = time.Now().Unix()
			if err := tx.Save(&invitation).Error; err != nil {
				return err
			}

			// Update User Inviter
			if err := tx.Model(&user).Update("inviter_id", invitation.InviterId).Error; err != nil {
				return err
			}

			// Reward Inviter
			invitationRewardStr := common.OptionMap[model.OptionKeyInvitationReward]
			invitationReward, _ := strconv.ParseFloat(invitationRewardStr, 64)
			if invitationReward > 0 {
				var inviter model.User
				if err := tx.First(&inviter, invitation.InviterId).Error; err == nil {
					if err := tx.Model(&inviter).Update("quota", gorm.Expr("quota + ?", int64(invitationReward))).Error; err != nil {
						return err
					}
					// Log reward
					log := model.Log{
						UserId:    inviter.Id,
						Username:  inviter.Username,
						CreatedAt: time.Now().Unix(),
						Type:      model.LogTypeSystem,
						Content:   fmt.Sprintf("邀请奖励: %s", user.Username),
						Quota:     int64(invitationReward),
						ModelName: "system",
					}
					tx.Create(&log)
				}
			}
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}

func GetAllUsers(c *gin.Context) {
	var users []model.User
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	offset := (page - 1) * size

	var total int64
	common.DB.Model(&model.User{}).Count(&total)

	if err := common.DB.Limit(size).Offset(offset).Order("id desc").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	// Hide passwords
	for i := range users {
		users[i].Password = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  users,
		"total": total,
	})
}

func AddUser(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if user.Status == 0 {
		user.Status = model.UserStatusActive
	}
	if err := user.Insert(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User created successfully", "data": user})
}

func UpdateUser(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existingUser model.User
	if err := common.DB.First(&existingUser, user.Id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update fields
	existingUser.Username = user.Username
	existingUser.DisplayName = user.DisplayName
	existingUser.Role = user.Role
	existingUser.Status = user.Status
	existingUser.Quota = user.Quota
	existingUser.Group = user.Group
	existingUser.Email = user.Email

	if user.Password != "" {
		hashed, err := common.Password2Hash(user.Password)
		if err == nil {
			existingUser.Password = hashed
		}
	}

	if err := common.DB.Save(&existingUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

func DeleteUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := common.DB.Delete(&model.User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func GetSelf(c *gin.Context) {
	userId, _ := c.Get("id")
	var user model.User
	if err := common.DB.First(&user, userId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	user.Password = ""
	c.JSON(http.StatusOK, gin.H{
		"data": user,
	})
}

func UpdateSelf(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId, _ := c.Get("id")
	var existingUser model.User
	if err := common.DB.First(&existingUser, userId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Only allow updating safe fields
	existingUser.DisplayName = user.DisplayName
	existingUser.Email = user.Email

	if user.Password != "" {
		hashed, err := common.Password2Hash(user.Password)
		if err == nil {
			existingUser.Password = hashed
		}
	}

	if err := common.DB.Save(&existingUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}
