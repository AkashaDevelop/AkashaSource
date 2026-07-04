package xuanjian

// 宸汐玄鉴·处置执行器
// 检测到异常之后，该干啥就干啥——
// warn 只记录，notify 发通知，throttle 降限速，
// disable_token / ban_user 是大招，只在 strict 模式出手。
// M4（生化武器类）无论模式都直接出大招，这条没有商量余地。

import (
	"encoding/json"
	"log"

	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
	"STfreApi/service"
)

// Enforce 根据 findings 和当前模式执行处置动作
func Enforce(findings []Finding, rec RequestRecord, cfg XJConfig, mode string) {
	if len(findings) == 0 {
		return
	}
	for _, f := range findings {
		// 异步记录所有 Finding 事件
		RecordEventAsync(rec.TokenID, rec.UserID, f.Score, rec.TokenName, rec.IP, rec.Model, f)

		// M4 bioweapon_chem：无论模式，直接 disable_token + notify
		if f.Type == "bioweapon_chem" {
			go executeDisableToken(rec.TokenID)
			go notifyAdmin(rec, f, "CRITICAL: bioweapon/chemical weapon content generation detected")
			continue
		}

		switch mode {
		case ModeMonitor:
			// 只记录，不动手
		case ModeProtect:
			if f.Score >= 70 && cfg.NotifyAdmin {
				go notifyAdmin(rec, f, "")
			}
		case ModeStrict:
			if f.Score >= cfg.AutoBanScore && rec.UserID > 0 {
				go executeBanUser(rec.UserID)
				go notifyAdmin(rec, f, "auto ban: risk score exceeded threshold")
			} else if f.Score >= cfg.AutoDisableScore && rec.TokenID > 0 {
				go executeDisableToken(rec.TokenID)
				go notifyAdmin(rec, f, "auto disable token: risk score exceeded threshold")
			} else if f.Score >= 70 && cfg.NotifyAdmin {
				go notifyAdmin(rec, f, "")
			}
		}
	}
}

func executeDisableToken(tokenID int) {
	if common.DB == nil || tokenID <= 0 {
		return
	}
	if err := common.DB.Model(&model.Token{}).Where("id = ?", tokenID).
		Update("status", model.TokenStatusDisabled).Error; err != nil {
		log.Printf("[xuanjian] 封禁 Token %d 失败: %v", tokenID, err)
	} else {
		log.Printf("[xuanjian] Token %d 已自动封禁", tokenID)
	}
}

func executeBanUser(userID int) {
	if common.DB == nil || userID <= 0 {
		return
	}
	if err := common.DB.Model(&model.User{}).Where("id = ?", userID).
		Update("status", model.UserStatusBanned).Error; err != nil {
		log.Printf("[xuanjian] 封禁用户 %d 失败: %v", userID, err)
	} else {
		log.Printf("[xuanjian] 用户 %d 已自动封禁", userID)
	}
}

func notifyAdmin(rec RequestRecord, f Finding, extra string) {
	if common.DB == nil {
		return
	}
	msg := "[宸汐玄鉴] 检测到风险行为\n" +
		"Token: " + rec.TokenName + " (ID:" + intStr(rec.TokenID) + ")\n" +
		"类型: " + f.Type + " | 分数: " + intStr(f.Score) + "\n" +
		"证据: " + f.Evidence
	if extra != "" {
		msg += "\n处置: " + extra
	}

	// 查询所有 root 用户并逐一通知
	var rootUsers []model.User
	if err := common.DB.Where("role >= ?", model.RoleRoot).Find(&rootUsers).Error; err != nil {
		log.Printf("[xuanjian] 查询管理员失败: %v", err)
		return
	}
	notify := dto.Notify{
		Type:    dto.NotifyTypeSystem,
		Title:   "宸汐玄鉴风险告警",
		Content: msg,
	}
	for _, u := range rootUsers {
		var setting dto.UserSetting
		if u.Setting != "" {
			_ = json.Unmarshal([]byte(u.Setting), &setting)
		}
		_ = service.NotifyUser(u.Id, u.Email, setting, notify)
	}
}
