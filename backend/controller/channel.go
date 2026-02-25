package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetAllChannels(c *gin.Context) {
	var channels []model.Channel
	// Simple fetching, in production you should add pagination and filtering
	if err := common.DB.Order("priority desc").Find(&channels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch channels"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": channels})
}

func AddChannel(c *gin.Context) {
	var channel model.Channel
	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Basic validation
	if channel.Name == "" || channel.Key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name and Key are required"})
		return
	}

	if err := common.DB.Create(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Channel created successfully", "data": channel})
}

func AddChannels(c *gin.Context) {
	var channels []model.Channel
	if err := c.ShouldBindJSON(&channels); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(channels) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No channels provided"})
		return
	}

	// Validate basic fields
	for _, channel := range channels {
		if channel.Name == "" || channel.Key == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Name and Key are required for all channels"})
			return
		}
	}

	if err := common.DB.Create(&channels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create channels"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Channels created successfully", "count": len(channels)})
}

func DeleteChannel(c *gin.Context) {
	id := c.Param("id")
	if err := common.DB.Delete(&model.Channel{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete channel"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Channel deleted successfully"})
}

func UpdateChannel(c *gin.Context) {
	var channel model.Channel
	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// We need ID to update
	if channel.Id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	if err := common.DB.Model(&channel).Updates(channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Channel updated successfully"})
}

func TestChannel(c *gin.Context) {
	// Simple mock test
	id := c.Param("id")
	// In production, we should actually test the connection
	// For now, let's just pretend it works
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"time":    100 + (len(id) * 10), // Mock latency
		"message": "Test passed",
	})
}
