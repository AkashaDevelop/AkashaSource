package controller

import (
	"STfreApi/common"
	"STfreApi/middleware"
	"STfreApi/model"
	"STfreApi/service"
	"STfreApi/service/xuanjian"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

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
	usingGroup, err := service.ResolveUsingGroup(user.Group, token.Group)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if !middleware.GroupRateLimitMiddleware(service.ResolveBillingGroup(user.Group, token.Group)) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": fmt.Sprintf("分组 %s 请求过于频繁", usingGroup)})
		return
	}
	channels, mappedModels, _, err := SelectChannelWithAffinity(modelName, user.Group, usingGroup, token.CrossGroupRetry, tokenKey, defaultChannelAffinityRule)
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
	go upsertChannelAffinity(defaultChannelAffinityRule, user.Group, getChannelAffinityKeyFP(tokenKey), channel.Id, mappedModel, 0, 0, 0)

	// 7. Bidirectional pipe
	done := make(chan struct{})
	var clientBytes, upstreamBytes int64

	// Client -> Upstream
	go func() {
		defer close(done)
		for {
			msgType, msg, err := clientConn.ReadMessage()
			if err != nil {
				return
			}
			atomic.AddInt64(&clientBytes, int64(len(msg)))
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
			atomic.AddInt64(&upstreamBytes, int64(len(msg)))
			if err := clientConn.WriteMessage(msgType, msg); err != nil {
				return
			}
		}
	}()

	<-done

	// 宸汐玄鉴：Realtime 是双向事件流，没有单一 prompt/completion 的概念，做不了内容级检测，
	// 但至少把这条连接的请求量记进行为画像，速率/规模类异常（比如疯狂建连）还是能查得到～
	if xuanjian.IsEnabled() {
		// 粗略按 4 字节/token 估算，够画像统计用，不追求精确
		promptTokens := int(atomic.LoadInt64(&clientBytes) / 4)
		completionTokens := int(atomic.LoadInt64(&upstreamBytes) / 4)
		go xuanjian.RecordRequest(xuanjian.RequestRecord{
			TokenID:          token.Id,
			TokenName:        token.Name,
			UserID:           user.Id,
			TokenCreatedAt:   time.Unix(token.CreatedTime, 0),
			IP:               c.ClientIP(),
			UserAgent:        c.GetHeader("User-Agent"),
			Model:            mappedModel,
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			StatusCode:       http.StatusOK,
		})
	}
}
