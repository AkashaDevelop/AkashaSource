package adapter

import (
	"STfreApi/model"
)

func GetAccountAdaptor(channelType int) AccountAdaptor {
	switch channelType {
	case model.ChannelTypeOpenAI, model.ChannelTypeCustom, model.ChannelTypeAzure:
		return NewNewApiAccountAdaptor()
	case model.ChannelTypeQwen, model.ChannelTypeDeepseek, model.ChannelTypeZhipu,
		model.ChannelTypeMoonshot, model.ChannelTypeSiliconFlow:
		return NewNewApiAccountAdaptor()
	default:
		return NewNewApiAccountAdaptor()
	}
}

func SupportsAccountFeatures(channelType int) bool {
	switch channelType {
	case model.ChannelTypeOpenAI, model.ChannelTypeCustom, model.ChannelTypeAzure,
		model.ChannelTypeQwen, model.ChannelTypeDeepseek, model.ChannelTypeZhipu,
		model.ChannelTypeMoonshot, model.ChannelTypeSiliconFlow:
		return true
	default:
		return false
	}
}
