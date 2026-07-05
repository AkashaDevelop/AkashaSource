package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"STfreApi/adapter"
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/middleware"
	"STfreApi/model"
	"STfreApi/service"
	"STfreApi/service/moderation"
	"STfreApi/service/qingyuan"
	"STfreApi/service/xuanjian"

	"github.com/gin-gonic/gin"
)

// Relay 处理核心转发逻辑
func Relay(c *gin.Context) {
	startTime := time.Now()
	// 1. 获取 Authorization Token
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "未提供 API Key，请在 Authorization 头中使用 Bearer 方式传入",
			Type:    "invalid_request_error",
			Code:    "invalid_api_key",
		}})
		return
	}
	tokenKey := strings.TrimPrefix(authHeader, "Bearer ")

	// 2. 验证 Token
	token, err := GetTokenByKey(tokenKey)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "API Key 无效",
			Type:    "invalid_request_error",
			Code:    "invalid_api_key",
		}})
		return
	}

	if err := ValidateToken(token); err != nil {
		c.JSON(http.StatusUnauthorized, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: err.Error(),
			Type:    "invalid_request_error",
		}})
		return
	}
	if !CheckTokenIP(token, c.ClientIP()) {
		c.JSON(http.StatusForbidden, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "令牌不允许的IP",
			Type:    "invalid_request_error",
		}})
		return
	}

	// 3. 解析请求体以获取 Model
	var openAIReq dto.OpenAIRequest
	var bodyBytes []byte

	contentType := c.Request.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		bodyBytes, err = io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
				Message: "请求体解析失败",
				Type:    "invalid_request_error",
			}})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
			c.JSON(http.StatusBadRequest, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
				Message: "表单解析失败",
				Type:    "invalid_request_error",
			}})
			return
		}
		openAIReq.Model = c.Request.FormValue("model")
		openAIReq.ContentType = contentType
		openAIReq.RawBody = bodyBytes
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	} else {
		bodyBytes, err = io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
				Message: "请求体解析失败",
				Type:    "invalid_request_error",
			}})
			return
		}
		if len(bodyBytes) > 0 {
			err = json.Unmarshal(bodyBytes, &openAIReq)
			if err != nil {
				c.JSON(http.StatusBadRequest, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
					Message: "请求体 JSON 解析失败",
					Type:    "invalid_request_error",
				}})
				return
			}
		} else {
			if c.Request.Method == "GET" {
				c.JSON(http.StatusOK, gin.H{
					"message": "Akasha 正常运行，请使用 POST /v1/chat/completions",
				})
				return
			}
		}
		openAIReq.RawBody = bodyBytes
	}

	// 未指定模型则使用默认模型
	if openAIReq.Model == "" {
		path := c.FullPath()
		if path == "/v1/audio/transcriptions" {
			openAIReq.Model = "whisper-1"
		} else if path == "/v1/audio/speech" {
			openAIReq.Model = "tts-1"
		} else if path == "/v1/images/generations" {
			openAIReq.Model = "dall-e-3"
		} else if path == "/v1/completions" {
			openAIReq.Model = "gpt-3.5-turbo-instruct"
		} else if path == "/v1/moderations" {
			openAIReq.Model = "text-moderation-latest"
		} else if path == "/v1/rerank" {
			openAIReq.Model = "rerank-1"
		} else {
			openAIReq.Model = common.DefaultModel
		}
	}

	var user model.User
	err = common.DB.Where("id = ?", token.UserId).First(&user).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "服务器内部错误",
			Type:    "server_error",
		}})
		return
	}

	common.OptionLock.RLock()
	moderationEnabled := common.OptionMap[model.OptionKeyContentModerationEnabled] == "true"
	keywords := common.OptionMap[model.OptionKeyContentModerationKeywords]
	whitelistUsers := common.OptionMap[model.OptionKeyContentModerationWhitelistUsers]
	whitelistModels := common.OptionMap[model.OptionKeyContentModerationWhitelistModels]
	whitelistIPs := common.OptionMap[model.OptionKeyContentModerationWhitelistIPs]
	common.OptionLock.RUnlock()
	whitelisted := isWhitelisted(user.Id, openAIReq.Model, c.ClientIP(), whitelistUsers, whitelistModels, whitelistIPs)
	if moderationEnabled && !whitelisted {
		if blocked, kw := isContentBlocked(&openAIReq, keywords); blocked {
			msg := fmt.Sprintf("内容包含敏感关键词: %s", kw)
			go RecordFailLog(c, token, openAIReq.Model, msg)
			c.JSON(http.StatusBadRequest, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
				Message: msg,
				Type:    "invalid_request_error",
			}})
			return
		}
	}

	if !CheckTokenModel(token, openAIReq.Model) {
		c.JSON(http.StatusForbidden, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "令牌不允许的模型",
			Type:    "invalid_request_error",
		}})
		return
	}

	// 按模型限流
	if !middleware.ModelRateLimitMiddleware(openAIReq.Model) {
		c.JSON(http.StatusTooManyRequests, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: fmt.Sprintf("模型 %s 请求过于频繁", openAIReq.Model),
			Type:    "rate_limit_error",
		}})
		return
	}

	common.OptionLock.RLock()
	moderationTimeout := common.ContentModerationTimeout
	common.OptionLock.RUnlock()
	if moderationEnabled && !whitelisted {
		allowed, reason := checkTencentModeration(moderationTimeout, buildModerationText(&openAIReq))
		if !allowed {
			msg := "内容审查未通过"
			if reason != "" {
				msg = "内容审查未通过: " + reason
			}
			go RecordFailLog(c, token, openAIReq.Model, msg)
			c.JSON(http.StatusBadRequest, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
				Message: msg,
				Type:    "invalid_request_error",
			}})
			return
		}
	}

	// 3.5 解析 reasoning 后缀
	reasoningParams := common.ParseReasoningSuffix(openAIReq.Model)
	openAIReq.Model = reasoningParams.CleanModel
	if reasoningParams.ReasoningEffort != "" {
		openAIReq.ReasoningEffort = reasoningParams.ReasoningEffort
	}
	if reasoningParams.ThinkingMode {
		openAIReq.ThinkingMode = true
	}

	// 4. 选择渠道
	channels, mappedModels, err := SelectChannelWithAffinity(openAIReq.Model, user.Group, tokenKey, defaultChannelAffinityRule)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: fmt.Sprintf("没有可用渠道，模型: %s", openAIReq.Model),
			Type:    "server_error",
		}})
		return
	}

	// 4.5 宸汐玄鉴 AI 审核（预审 / 规则初审+AI复审 / 两者）
	// 在渠道选择后、qingyuan 净化前执行，审核不通过直接拦截
	if xuanjian.IsEnabled() {
		allowed, blockMsg := xuanjian.AIReviewCheck(token.Id, user.Id, openAIReq.Messages, openAIReq.Prompt)
		if !allowed {
			go RecordFailLog(c, token, openAIReq.Model, blockMsg)
			c.JSON(http.StatusBadRequest, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
				Message: blockMsg,
				Type:    "invalid_request_error",
				Code:    "ai_review_blocked",
			}})
			return
		}
	}

	requestedModel := openAIReq.Model
	baseReq := openAIReq

	// 预扣额度
	preConsumedQuota, preConsumeErr := service.PreConsumeQuota(token,
		openAIReq.MaxTokens, // 预估 prompt tokens（简化处理）
		openAIReq.MaxTokens,
		openAIReq.Model, 0, 0)
	if preConsumeErr != nil {
		c.JSON(http.StatusPaymentRequired, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
			Message: "额度不足: " + preConsumeErr.Error(),
			Type:    "insufficient_quota",
			Code:    "insufficient_quota",
		}})
		return
	}

	// 预扣退款标志位：成功结算后置 true，函数返回时未结算则退还预扣
	billingSettled := false
	defer func() {
		if !billingSettled {
			service.RefundQuota(token, preConsumedQuota)
		}
	}()

	// 重试循环
	var lastError error
	for i, channel := range channels {
		mappedModel := mappedModels[i]
		attemptReq, copyErr := qingyuan.DeepCopyOpenAIRequest(&baseReq)
		if copyErr != nil {
			lastError = copyErr
			continue
		}

		// 更新本次请求模型
		attemptReq.Model = mappedModel

		// 多 Key 轮换
		channel.Key = service.GetNextKey(channel.Key)

		policy := qingyuan.ResolvePolicy(channel.Id, requestedModel, mappedModel)
		reqCtx := qingyuan.RequestContext{
			RequestId:      c.GetString("request_id"),
			UserId:         user.Id,
			TokenId:        token.Id,
			TokenName:      token.Name,
			ChannelId:      channel.Id,
			ChannelName:    channel.Name,
			ChannelType:    channel.Type,
			RequestedModel: requestedModel,
			MappedModel:    mappedModel,
			UserGroup:      user.Group,
			ClientIP:       c.ClientIP(),
		}
		sanitizeResult, sanitizeErr := qingyuan.ApplyRequest(c.Request.Context(), attemptReq, reqCtx, policy)
		if sanitizeErr != nil {
			lastError = sanitizeErr
			continue
		}
		if sanitizeResult.Blocked {
			msg := sanitizeResult.Message
			if msg == "" {
				msg = "请求被安全策略拒绝，如有疑问请联系管理员"
			}
			go RecordFailLog(c, token, mappedModel, msg)
			c.JSON(http.StatusBadRequest, dto.OpenAIErrorResponse{Error: dto.OpenAIError{Message: msg, Type: "invalid_request_error", Code: "context_sanitization_blocked"}})
			return
		}
		if sanitizeResult.Changed {
			attemptReq.RawBody = nil
		}
		c.Set("context_sanitization_policy", policy)
		c.Set("context_sanitization_request_context", reqCtx)
		c.Set("context_sanitization_user_requested_ads", qingyuan.IsUserRequestingAds(attemptReq))
		c.Set("context_sanitization_request_tools", attemptReq.Tools)

		// 获取适配器
		adaptor := adapter.GetAdaptor(channel.Type, channel)

		// 转换请求
		var convertedReq any
		convertedReq, err = adaptor.ConvertRequest(c, attemptReq)
		if err != nil {
			lastError = err
			continue
		}

		// 发起请求
		var resp *http.Response
		resp, err = adaptor.DoRequest(c, channel, convertedReq)
		if err != nil {
			lastError = err
			continue
		}

		// 错误处理与自动禁用
		if resp.StatusCode >= 400 {
			// 读取响应体获取错误信息
			responseBody, err := io.ReadAll(resp.Body)
			resp.Body.Close()

			if err != nil {
				lastError = fmt.Errorf("渠道 %s 返回状态码 %d，读取响应体失败", channel.Name, resp.StatusCode)
				continue
			}

			var errorResponse dto.OpenAIErrorResponse
			// 尝试解析错误体，失败也不影响状态码处理
			json.Unmarshal(responseBody, &errorResponse)

			// 判断是否需要禁用渠道
			if service.ShouldDisableChannel(&errorResponse.Error, resp.StatusCode) {
				// 使用协程避免阻塞
				go service.DisableChannel(channel.Id, fmt.Sprintf("%d - %s", resp.StatusCode, errorResponse.Error.Message))
			}

			// 判断是否重试
			shouldRetry := resp.StatusCode == 429 || resp.StatusCode >= 500 || service.ShouldDisableChannel(&errorResponse.Error, resp.StatusCode)

			if shouldRetry {
				lastError = fmt.Errorf("渠道 %s 返回错误: %d %s", channel.Name, resp.StatusCode, errorResponse.Error.Message)
				continue
			}

			// 不重试则直接返回
			c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), responseBody)
			return
		}

		// 成功响应
		defer resp.Body.Close()

		// 处理响应并计算用量
		usage, err := adaptor.DoResponse(c, resp, token)
		if err != nil {
			// 响应解析失败时不再重试
		}

		// 记录日志
		promptTokens := 0
		completionTokens := 0
		cachedTokens := 0

		// 按次计费的模型，CalculateQuota 会自动处理
		// 不再需要手动设置 promptTokens
		if usage != nil {
			promptTokens = usage.PromptTokens
			completionTokens = usage.CompletionTokens
			cachedTokens = usage.CachedTokens
		} else {
			// 适配器未返回用量时估算
			if attemptReq.Input != nil {
				promptTokens = common.CountToken(fmt.Sprint(attemptReq.Input))
			} else {
				promptTokens = common.CountToken(fmt.Sprint(attemptReq.Messages))
			}
			// 流式返回无法准确统计输出用量
		}

		c.Set("channel_id", channel.Id)
		c.Set("is_stream", openAIReq.Stream)
		c.Set("use_time", int(time.Since(startTime).Milliseconds()))

		if usage != nil {
			RecordConsumeWithBilling(c, token, mappedModel, preConsumedQuota, usage)
			billingSettled = true
		}

		// 宸汐玄鉴：异步行为分析，不阻塞主链路
		if xuanjian.IsEnabled() {
			qyTypes := make([]string, 0, len(sanitizeResult.Findings))
			for _, f := range sanitizeResult.Findings {
				qyTypes = append(qyTypes, f.Type)
			}
			promptSnippet := xuanjian.CollectPromptSnippet(attemptReq.Messages, attemptReq.Prompt, 2000)
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
				PromptSnippet:    promptSnippet,
				Messages:         attemptReq.Messages,
				StatusCode:       resp.StatusCode,
				QYFindings:       qyTypes,
				QYRiskScore:      sanitizeResult.RiskScore,
			})
		}
		go upsertChannelAffinity(
			defaultChannelAffinityRule,
			user.Group,
			getChannelAffinityKeyFP(tokenKey),
			channel.Id,
			openAIReq.Model,
			promptTokens,
			completionTokens,
			cachedTokens,
		)

		return
	}

	// 所有渠道失败
	errorMsg := fmt.Sprintf("所有渠道调用失败。最后一次错误: %v", lastError)
	go RecordFailLog(c, token, openAIReq.Model, errorMsg)

	c.JSON(http.StatusServiceUnavailable, dto.OpenAIErrorResponse{Error: dto.OpenAIError{
		Message: errorMsg,
		Type:    "server_error",
	}})
}

func isContentBlocked(req *dto.OpenAIRequest, keywords string) (bool, string) {
	if keywords == "" {
		return false, ""
	}
	parts := strings.FieldsFunc(keywords, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；' || r == '\n' || r == '\t' || r == ' ' || r == '|'
	})
	if len(parts) == 0 {
		return false, ""
	}
	var texts []string
	if req.Prompt != "" {
		texts = append(texts, req.Prompt)
	}
	if req.Input != nil {
		texts = append(texts, fmt.Sprint(req.Input))
	}
	for _, msg := range req.Messages {
		if m, ok := msg.(map[string]interface{}); ok {
			if content, ok := m["content"]; ok {
				texts = append(texts, fmt.Sprint(content))
			}
		}
	}
	for _, t := range texts {
		lt := strings.ToLower(t)
		for _, kw := range parts {
			kw = strings.TrimSpace(kw)
			if kw == "" {
				continue
			}
			if strings.Contains(lt, strings.ToLower(kw)) {
				return true, kw
			}
		}
	}
	return false, ""
}

func isWhitelisted(userId int, modelName string, ip string, userList string, modelList string, ipList string) bool {
	if isInList(strconv.Itoa(userId), userList) {
		return true
	}
	if isInList(modelName, modelList) {
		return true
	}
	if isInList(ip, ipList) {
		return true
	}
	return false
}

func isInList(value string, list string) bool {
	if value == "" || list == "" {
		return false
	}
	items := strings.FieldsFunc(list, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；' || r == '\n' || r == '\t' || r == ' ' || r == '|'
	})
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.EqualFold(item, value) {
			return true
		}
	}
	return false
}

func buildModerationText(req *dto.OpenAIRequest) string {
	var texts []string
	if req.Prompt != "" {
		texts = append(texts, req.Prompt)
	}
	if req.Input != nil {
		texts = append(texts, fmt.Sprint(req.Input))
	}
	for _, msg := range req.Messages {
		if m, ok := msg.(map[string]interface{}); ok {
			if content, ok := m["content"]; ok {
				texts = append(texts, fmt.Sprint(content))
			}
		}
	}
	return strings.Join(texts, "\n")
}

// checkTencentModeration ～请腾讯云天御大人来给这段文字把把关，判定是否放行～
func checkTencentModeration(timeout int, text string) (bool, string) {
	common.OptionLock.RLock()
	cfg := moderation.TMSConfig{
		SecretId:  common.OptionMap[model.OptionKeyTencentModerationSecretId],
		SecretKey: common.OptionMap[model.OptionKeyTencentModerationSecretKey],
		Region:    common.OptionMap[model.OptionKeyTencentModerationRegion],
		BizType:   common.OptionMap[model.OptionKeyTencentModerationBizType],
		Timeout:   timeout,
	}
	common.OptionLock.RUnlock()

	result, err := moderation.ModerateText(cfg, text)
	if err != nil {
		// 调用失败不能拖住正常业务，放行本次请求并把原因打进日志方便排查
		log.Printf("[内容审查] 腾讯云天御调用失败，本次请求放行: %v", err)
		return true, ""
	}
	if result.Suggestion != "Block" {
		return true, ""
	}
	reason := result.Label
	if result.SubLabel != "" {
		reason += "/" + result.SubLabel
	}
	return false, reason
}
