package controller

import (
	"STfreApi/common"
	"STfreApi/common/email"
	"STfreApi/model"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
	Turnstile string `json:"turnstile"`
}

func UserLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if common.TurnstileCheckEnabled && !common.VerifyTurnstile(req.Turnstile) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Turnstile verification failed"})
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
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
	Email     string `json:"email"`
	Turnstile string `json:"turnstile"`
}

func UserRegister(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if common.TurnstileCheckEnabled && !common.VerifyTurnstile(req.Turnstile) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Turnstile verification failed"})
		return
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

	// Check if this is the first user
	var count int64
	common.DB.Model(&model.User{}).Count(&count)
	if count == 0 {
		user.Role = model.RoleRoot // First user is Root
		user.Status = model.UserStatusActive
	} else {
		// Email Verification Logic if enabled
		if common.EmailVerificationEnabled {
			user.Status = model.UserStatusBanned // Pending verification
			// Send verification email
			code := fmt.Sprintf("%06d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1000000))
			// Save code to Redis or DB (omitted for brevity, assume valid for 5 mins)
			// For now, we just log it or skip verification implementation details

			go func() {
				err := email.SendEmail(req.Email, "Email Verification", "Your verification code is: "+code)
				if err != nil {
					fmt.Println("Failed to send email:", err)
				}
			}()
		}
	}

	if err := user.Insert(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
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
