package controller

import (
	"net/http"
	"strings"
	"time"

	"STfreApi/common"
	"STfreApi/model"

	"github.com/gin-gonic/gin"
)

type ModelObject struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type ModelListResponse struct {
	Object string        `json:"object"`
	Data   []ModelObject `json:"data"`
}

func ListModels(c *gin.Context) {
	var channels []model.Channel
	common.DB.Where("status = ?", model.ChannelStatusActive).Find(&channels)

	modelSet := make(map[string]bool)
	var models []ModelObject

	for _, ch := range channels {
		for _, m := range strings.Split(ch.Models, ",") {
			m = strings.TrimSpace(m)
			if m != "" && !modelSet[m] {
				modelSet[m] = true
				models = append(models, ModelObject{
					Id:      m,
					Object:  "model",
					Created: time.Now().Unix(),
					OwnedBy: "system",
				})
			}
		}
	}

	c.JSON(http.StatusOK, ModelListResponse{
		Object: "list",
		Data:   models,
	})
}

func RetrieveModel(c *gin.Context) {
	modelId := c.Param("model")

	var channels []model.Channel
	common.DB.Where("status = ?", model.ChannelStatusActive).Find(&channels)

	for _, ch := range channels {
		for _, m := range strings.Split(ch.Models, ",") {
			if strings.TrimSpace(m) == modelId {
				c.JSON(http.StatusOK, ModelObject{
					Id:      modelId,
					Object:  "model",
					Created: time.Now().Unix(),
					OwnedBy: "system",
				})
				return
			}
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"error": gin.H{
			"message": "model not found",
			"type":    "invalid_request_error",
			"code":    "model_not_found",
		},
	})
}
