package adapter

import (
	"STfreApi/adapter/ali"
	"STfreApi/adapter/baidu"
	"STfreApi/adapter/claude"
	"STfreApi/adapter/deepseek"
	"STfreApi/adapter/dify"
	"STfreApi/adapter/gemini"
	"STfreApi/adapter/moonshot"
	"STfreApi/adapter/ollama"
	"STfreApi/adapter/openai"
	"STfreApi/adapter/tencent"
	"STfreApi/adapter/xunfei"
	"STfreApi/adapter/zhipu"
	"STfreApi/model"
)

func GetAdaptor(channelType int) Adaptor {
	switch channelType {
	case model.ChannelTypeAnthropic:
		return &claude.Adaptor{}
	case model.ChannelTypeGemini:
		return &gemini.Adaptor{}
	case model.ChannelTypeQwen:
		return &ali.Adaptor{}
	case model.ChannelTypeHunyuan:
		return &tencent.Adaptor{}
	case model.ChannelTypeWenxin:
		return &baidu.Adaptor{}
	case model.ChannelTypeSpark:
		return &xunfei.Adaptor{}
	case model.ChannelTypeDeepseek:
		return &deepseek.Adaptor{}
	case model.ChannelTypeZhipu:
		return &zhipu.Adaptor{}
	case model.ChannelTypeMoonshot:
		return &moonshot.Adaptor{}
	case model.ChannelTypeOllama:
		return &ollama.Adaptor{}
	case model.ChannelTypeDify:
		return &dify.Adaptor{}
	case model.ChannelTypeOpenAI, model.ChannelTypeCustom, model.ChannelTypeAzure:
		return &openai.Adaptor{}
	default:
		return &openai.Adaptor{}
	}
}
