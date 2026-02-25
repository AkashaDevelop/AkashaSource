package adapter

import (
	"STfreApi/adapter/claude"
	"STfreApi/adapter/gemini"
	"STfreApi/adapter/openai"
	"STfreApi/model"
)

func GetAdaptor(channelType int) Adaptor {
	switch channelType {
	case model.ChannelTypeAnthropic:
		return &claude.Adaptor{}
	case model.ChannelTypeGemini:
		return &gemini.Adaptor{}
	case model.ChannelTypeOpenAI, model.ChannelTypeCustom, model.ChannelTypeAzure:
		return &openai.Adaptor{}
	default:
		return &openai.Adaptor{}
	}
}
