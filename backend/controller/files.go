package controller

import (
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func getApiToken(c *gin.Context) (*model.Token, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, http.ErrNoCookie
	}
	tokenKey := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := GetTokenByKey(tokenKey)
	if err != nil {
		return nil, err
	}
	if err := ValidateToken(token); err != nil {
		return nil, err
	}
	if !CheckTokenIP(token, c.ClientIP()) {
		return nil, http.ErrNotSupported
	}
	return token, nil
}

func FilesList(c *gin.Context) {
	token, err := getApiToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "未授权",
			Type:    "invalid_request_error",
		}})
		return
	}

	var files []model.StoredFile
	if err := common.DB.Where("user_id = ?", token.UserId).Order("id desc").Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取文件列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": files})
}

func FilesCreate(c *gin.Context) {
	token, err := getApiToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "未授权",
			Type:    "invalid_request_error",
		}})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少文件"})
		return
	}
	purpose := c.PostForm("purpose")
	if purpose == "" {
		purpose = "fine-tune"
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件读取失败"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件读取失败"})
		return
	}

	item := model.StoredFile{
		UserId:    token.UserId,
		Purpose:   purpose,
		FileName:  fileHeader.Filename,
		Bytes:     int64(len(data)),
		Content:   data,
		CreatedAt: time.Now().Unix(),
	}
	if err := common.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件保存失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func FilesRetrieve(c *gin.Context) {
	token, err := getApiToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "未授权",
			Type:    "invalid_request_error",
		}})
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	var item model.StoredFile
	if err := common.DB.Where("id = ? AND user_id = ?", id, token.UserId).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	item.Content = nil
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func FilesContent(c *gin.Context) {
	token, err := getApiToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "未授权",
			Type:    "invalid_request_error",
		}})
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	var item model.StoredFile
	if err := common.DB.Where("id = ? AND user_id = ?", id, token.UserId).First(&item).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	c.Data(http.StatusOK, "application/octet-stream", item.Content)
}

func FilesDelete(c *gin.Context) {
	token, err := getApiToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "未授权",
			Type:    "invalid_request_error",
		}})
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	if err := common.DB.Where("id = ? AND user_id = ?", id, token.UserId).Delete(&model.StoredFile{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除文件失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
