package xuanjian

// ～宸汐玄鉴·事件记录员～ ✦
// 每次检测到异常，都要在这里留下一条记录，
// 供管理员后续复核和追溯用。

import (
	"encoding/json"
	"time"

	"STfreApi/common"
	"STfreApi/model"
)

// RecordEventAsync 异步写入玄鉴事件（不阻塞主链路）
func RecordEventAsync(tokenID, userID, riskScore int, tokenName, ip, modelName string, f Finding) {
	go func() { _ = recordEvent(tokenID, userID, riskScore, tokenName, ip, modelName, f) }()
}

func recordEvent(tokenID, userID, riskScore int, tokenName, ip, modelName string, f Finding) error {
	if common.DB == nil {
		return nil
	}
	evidenceJSON, _ := json.Marshal(map[string]string{
		"finding_type": f.Type,
		"evidence":     f.Evidence,
	})
	return common.DB.Create(&model.XuanJianEvent{
		CreatedAt:    time.Now().Unix(),
		TokenID:      tokenID,
		TokenName:    tokenName,
		UserID:       userID,
		RiskScore:    riskScore,
		FindingType:  f.Type,
		FindingGroup: f.Group,
		Action:       f.Action,
		Evidence:     string(evidenceJSON),
		IP:           ip,
		Model:        modelName,
	}).Error
}
