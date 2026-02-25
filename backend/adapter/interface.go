package adapter

import (
	"STfreApi/dto"
	"STfreApi/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Adaptor interface {
	ConvertRequest(c *gin.Context, request *dto.OpenAIRequest) (any, error)
	DoRequest(c *gin.Context, channel *model.Channel, request any) (*http.Response, error)
	DoResponse(c *gin.Context, resp *http.Response, meta *model.Token) (usage *dto.Usage, err error)
	// GetModelList returns a list of models for this channel.
	// This list is used for UI display or validation.
	// Note: This does not restrict the models that can be used if manually configured.
	GetModelList() []string
	GetChannelName() string
}
