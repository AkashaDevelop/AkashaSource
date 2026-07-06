package realname

import (
	"STfreApi/common"
	"STfreApi/model"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// IsRealnameEnabled 检查实名认证模块是否启用
func IsRealnameEnabled() bool {
	return common.OptionMap[model.OptionKeyRealnameEnabled] == "true"
}

// GetRealnameScenarios 获取需要实名认证的场景列表
func GetRealnameScenarios() []string {
	scenariosStr := common.OptionMap[model.OptionKeyRealnameScenarios]
	if scenariosStr == "" {
		return []string{}
	}
	var scenarios []string
	if err := json.Unmarshal([]byte(scenariosStr), &scenarios); err != nil {
		return []string{}
	}
	return scenarios
}

// IsScenarioRequired 检查指定场景是否需要实名认证
func IsScenarioRequired(scenario string) bool {
	if !IsRealnameEnabled() {
		return false
	}
	scenarios := GetRealnameScenarios()
	for _, s := range scenarios {
		if s == scenario {
			return true
		}
	}
	return false
}

// IsDoubleBlindMode 检查是否启用了双盲模式
// 双盲模式：只存认证结果和流水号，不存任何实名信息（姓名、身份证号），做信息合规处理
func IsDoubleBlindMode() bool {
	return IsScenarioRequired(model.RealnameScenarioDoubleBlind)
}

// IsUserRealnameVerified 检查用户是否已通过实名认证
func IsUserRealnameVerified(userId int) bool {
	var user model.User
	if err := common.DB.Select("realname_status").First(&user, userId).Error; err != nil {
		return false
	}
	return user.RealnameStatus == model.RealnameStatusPassed
}

// CheckRealnameRequirement 检查用户在指定场景下是否需要实名认证
// 返回 nil 表示通过，返回 error 表示需要认证但未通过
func CheckRealnameRequirement(userId int, scenario string) error {
	if !IsScenarioRequired(scenario) {
		return nil
	}
	if IsUserRealnameVerified(userId) {
		return nil
	}
	return fmt.Errorf("当前操作需要实名认证，请先完成实名认证")
}

// HashIdCard 对身份证号进行 SHA-256 哈希，避免明文存储
func HashIdCard(idCard string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(idCard)))
	return hex.EncodeToString(h[:])
}

// GetAliyunClient 从配置创建阿里云实人认证客户端
func GetAliyunClient() *AliyunClient {
	return NewAliyunClient(
		common.OptionMap[model.OptionKeyRealnameAliyunAccessKeyId],
		common.OptionMap[model.OptionKeyRealnameAliyunAccessKeySecret],
		common.OptionMap[model.OptionKeyRealnameAliyunRegion],
		common.OptionMap[model.OptionKeyRealnameAliyunSceneId],
	)
}

// IsConfigured 检查阿里云实人认证是否已正确配置
func IsConfigured() bool {
	return common.OptionMap[model.OptionKeyRealnameAliyunAccessKeyId] != "" &&
		common.OptionMap[model.OptionKeyRealnameAliyunAccessKeySecret] != ""
}

// GetRecordByCertifyId 根据认证流水号获取记录
func GetRecordByCertifyId(certifyId string) (*model.RealnameAuth, error) {
	return model.GetRealnameAuthByCertifyId(certifyId)
}
