package service

import (
	"strings"
	"time"

	"STfreApi/common"
	"STfreApi/model"
)

const (
	ViolationFeeCodePrefix        = "violation_fee."
	ErrorCodeViolationFeeGrokCSAM = "violation_fee.grok_csam"
	CSAMViolationMarker           = "Failed check: SAFETY_CHECK_TYPE"
	ContentViolatesUsageMarker    = "Content violates usage guidelines"
)

func HasCSAMViolationMarker(errorMsg string) bool {
	if errorMsg == "" {
		return false
	}
	return strings.Contains(errorMsg, CSAMViolationMarker) ||
		strings.Contains(errorMsg, ContentViolatesUsageMarker)
}

func IsViolationFeeCode(code string) bool {
	return strings.HasPrefix(code, ViolationFeeCodePrefix)
}

func CalcViolationFeeQuota(amount, groupRatio float64) int {
	if amount <= 0 || groupRatio <= 0 {
		return 0
	}
	quota := int(amount * common.QuotaPerUnit * groupRatio)
	if quota <= 0 {
		return 0
	}
	return quota
}

type ViolationFeeParams struct {
	UserId         int
	ChannelId      int
	TokenId        int
	ModelName      string
	TokenName      string
	Group          string
	IsStream       bool
	Amount         float64
	GroupRatio     float64
	ErrorCode      string
	ErrorMessage   string
	StartTime      time.Time
}

func ChargeViolationFee(params ViolationFeeParams) bool {
	feeQuota := CalcViolationFeeQuota(params.Amount, params.GroupRatio)
	if feeQuota <= 0 {
		return false
	}

	model.UpdateUserUsedQuotaAndRequestCount(params.UserId, feeQuota)
	if params.ChannelId > 0 {
		model.UpdateChannelUsedQuota(params.ChannelId, feeQuota)
	}

	return true
}
