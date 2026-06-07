package service

import (
	"STfreApi/adapter"
	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
	"fmt"
	"log"
	"strings"
	"time"
)

// CheckChannel 测试渠道可用性，返回响应时间(ms)和错误
func CheckChannel(channel *model.Channel) (int, error) {
	return CheckChannelWithPrompt(channel, "", "")
}

// CheckChannelWithPrompt 使用自定义问题测试渠道可用性
func CheckChannelWithPrompt(channel *model.Channel, modelName string, prompt string) (int, error) {
	startTime := time.Now()

	// Pick a model
	if modelName == "" {
		models := strings.Split(channel.Models, ",")
		if len(models) > 0 && models[0] != "" {
			modelName = models[0]
		} else {
			// Default models per type if not specified
			switch channel.Type {
			case model.ChannelTypeOpenAI:
				modelName = "gpt-3.5-turbo"
			case model.ChannelTypeAnthropic:
				modelName = "claude-3-haiku-20240307"
			case model.ChannelTypeGemini:
				modelName = "gemini-pro"
			case model.ChannelTypeMidjourney:
				// MJ check is complex, skip for now
				return 0, nil
			default:
				modelName = "gpt-3.5-turbo"
			}
		}
	}

	if prompt == "" {
		prompt = "现在几点了"
	}

	req := &dto.OpenAIRequest{
		Model: modelName,
		Messages: []interface{}{
			map[string]string{
				"role":    "user",
				"content": prompt,
			},
		},
		MaxTokens: 1,
	}

	adaptor := adapter.GetAdaptor(channel.Type, channel)
	// Passing nil context as these adaptors don't strictly require it for conversion/request
	convertedReq, err := adaptor.ConvertRequest(nil, req)
	if err != nil {
		return 0, err
	}

	resp, err := adaptor.DoRequest(nil, channel, convertedReq)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("status code %d", resp.StatusCode)
	}

	elapsed := time.Since(startTime)
	return int(elapsed.Milliseconds()), nil
}

// StartChannelCheck 启动定时任务
func StartChannelCheck() {
	go func() {
		for {
			time.Sleep(1 * time.Hour) // Check every hour
			CheckAllChannels()
		}
	}()
}

func CheckAllChannels() {
	var channels []model.Channel
	common.DB.Find(&channels)

	for _, channel := range channels {
		// If disabled (auto-ban), try to recover
		if channel.Status == model.ChannelStatusAutoDisabled {
			responseTime, err := CheckChannel(&channel)
			if err == nil {
				// Recover
				log.Printf("[渠道检查] 渠道 #%d (%s) 已恢复", channel.Id, channel.Name)
				common.DB.Model(&channel).Updates(map[string]interface{}{
					"status":        model.ChannelStatusActive,
					"response_time": responseTime,
					"test_time":     common.GetTimestamp(),
				})
			} else {
				// Update test time even if failed
				common.DB.Model(&channel).Update("test_time", common.GetTimestamp())
			}
		} else if channel.Status == model.ChannelStatusActive {
			// Update response time for active channels
			responseTime, err := CheckChannel(&channel)
			if err == nil {
				common.DB.Model(&channel).Updates(map[string]interface{}{
					"response_time": responseTime,
					"test_time":     common.GetTimestamp(),
				})
			} else {
				log.Printf("[渠道检查] 渠道 #%d (%s) 检查失败: %v", channel.Id, channel.Name, err)
				common.DB.Model(&channel).Update("test_time", common.GetTimestamp())
			}
		}
	}
}
