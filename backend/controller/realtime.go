package controller

import (
	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/service"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// RelayRealtime handles WebSocket proxy for OpenAI Realtime API
func RelayRealtime(c *gin.Context) {
	// 1. Auth
	tokenKey := c.Query("api_key")
	if tokenKey == "" {
		authHeader := c.GetHeader("Authorization")
		tokenKey = strings.TrimPrefix(authHeader, "Bearer ")
	}
	if tokenKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing api key"})
		return
	}

	token, err := GetTokenByKey(tokenKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
		return
	}
	if err := ValidateToken(token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 2. Get user
	var user model.User
	if err := common.DB.Where("id = ?", token.UserId).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	// 3. Determine model
	modelName := c.Query("model")
	if modelName == "" {
		modelName = "gpt-4o-realtime-preview"
	}

	// 4. Select channel
	channels, mappedModels, err := SelectChannel(modelName, user.Group)
	if err != nil || len(channels) == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": fmt.Sprintf("no channel for model: %s", modelName),
		})
		return
	}

	channel := channels[0]
	mappedModel := mappedModels[0]
	channel.Key = service.GetNextKey(channel.Key)

	// 5. Upgrade client connection
	clientConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer clientConn.Close()

	// 6. Connect to upstream
	baseURL := channel.BaseURL
	if baseURL == "" {
		baseURL = "wss://api.openai.com"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	baseURL = strings.Replace(baseURL, "https://", "wss://", 1)
	baseURL = strings.Replace(baseURL, "http://", "ws://", 1)

	upstreamURL := fmt.Sprintf(
		"%s/v1/realtime?model=%s",
		baseURL, mappedModel,
	)

	header := http.Header{}
	header.Set("Authorization", "Bearer "+channel.Key)
	header.Set("OpenAI-Beta", "realtime=v1")

	upstreamConn, _, err := websocket.DefaultDialer.Dial(upstreamURL, header)
	if err != nil {
		clientConn.WriteMessage(websocket.TextMessage,
			[]byte(fmt.Sprintf(`{"error":"upstream connect failed: %s"}`, err.Error())))
		return
	}
	defer upstreamConn.Close()

	// 7. Bidirectional pipe
	done := make(chan struct{})

	// Client -> Upstream
	go func() {
		defer close(done)
		for {
			msgType, msg, err := clientConn.ReadMessage()
			if err != nil {
				return
			}
			if err := upstreamConn.WriteMessage(msgType, msg); err != nil {
				return
			}
		}
	}()

	// Upstream -> Client
	go func() {
		for {
			msgType, msg, err := upstreamConn.ReadMessage()
			if err != nil {
				clientConn.Close()
				return
			}
			if err := clientConn.WriteMessage(msgType, msg); err != nil {
				return
			}
		}
	}()

	<-done
}
