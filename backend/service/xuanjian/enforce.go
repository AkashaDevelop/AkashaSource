package xuanjian

// 宸汐玄鉴·处置执行器
// 检测到异常之后，该干啥就干啥——
// warn 只记录，notify 发通知，throttle 降限速，
// disable_token / ban_user 是大招，只在 strict 模式出手。
// M4（生化武器类）无论模式都直接出大招，这条没有商量余地。

import (
	"encoding/json"
	"log"
	"strconv"

	"STfreApi/common"
	"STfreApi/dto"
	"STfreApi/model"
	"STfreApi/service"
	"STfreApi/service/sanction"
)

// Enforce 根据 findings、规则的 Action 字段和当前模式执行处置动作。
//
// 处置模型（2026 重构）：
//   - Action 决定"处置什么"（throttle/rpm_limit/suspend/billing_penalty/disable/ban_ip/ban_user/notify/warn）
//   - mode 是"破坏性动作的阀门"：
//     monitor — 全部只记录 + 通知，绝不动手
//     protect — 放行非破坏性、可自愈的处置（throttle/rpm_limit/suspend/billing_penalty/notify）
//     strict  — 额外允许破坏性处置（disable_token/ban_ip/ban_user）落地
//   - M4（bioweapon_chem）在 protect/strict 下无视规则 Action 直接出大招；
//     monitor 下仍然只告警——见下方说明。
func Enforce(findings []Finding, rec RequestRecord, cfg XJConfig, mode string) {
	if len(findings) == 0 {
		return
	}
	for _, f := range findings {
		// 异步记录所有 Finding 事件
		RecordEventAsync(rec.TokenID, rec.UserID, f.Score, rec.TokenName, rec.IP, rec.Model, f)

		// M4 bioweapon_chem：最高优先级，protect 及以上直接 disable_token + notify
		//
		// ～2026.8.4 修正：这条以前是"无论模式"直接封号，连 monitor 都不放过。
		// 可 monitor 的语义就是"我只想先看看，别动我的用户"——管理员刚接入、
		// 正在观察误报率的时候被系统自作主张封了客户的 token，那是要出事故的 (๑•́ ₃ •̀๑)
		// 现在 monitor 下改为"最高优先级告警但不处置"，通知里会写清楚为什么没动手，
		// 管理员看到后可以自己决定是手动封禁还是切到 protect 模式。
		if f.Type == "bioweapon_chem" {
			if mode == ModeMonitor {
				go notifyAdmin(rec, f, "CRITICAL: 检测到生化武器内容，但当前为监控模式未自动处置，请人工确认")
			} else {
				applyDisableToken(rec, f, "CRITICAL: 生化武器/化学武器内容")
				go notifyAdmin(rec, f, "CRITICAL: bioweapon/chemical weapon content generation detected")
			}
			continue
		}

		action := f.Action
		if action == "" {
			action = "warn"
		}

		switch action {
		case "warn":
			// 仅记录（已在上方 RecordEventAsync 完成）

		case "notify":
			if mode != ModeMonitor && cfg.NotifyAdmin {
				go notifyAdmin(rec, f, "")
			}

		case model.SanctionThrottle:
			// 非破坏性：protect 及以上生效
			if mode != ModeMonitor && rec.TokenID > 0 {
				applySanction(rec, f, model.SanctionTargetToken, strconv.Itoa(rec.TokenID),
					model.SanctionThrottle, throttleFactor(cfg), cfg.ThrottleDurationMinutes)
			}

		case model.SanctionRPMLimit:
			if mode != ModeMonitor && rec.TokenID > 0 {
				applySanction(rec, f, model.SanctionTargetToken, strconv.Itoa(rec.TokenID),
					model.SanctionRPMLimit, float64(penaltyRPM(cfg)), cfg.ThrottleDurationMinutes)
			}

		case model.SanctionSuspendToken:
			if mode != ModeMonitor && rec.TokenID > 0 {
				applySanction(rec, f, model.SanctionTargetToken, strconv.Itoa(rec.TokenID),
					model.SanctionSuspendToken, 0, suspendDuration(cfg))
			}

		case model.SanctionBillingPenalty:
			// 非破坏性（只是多收费）：protect 及以上生效，默认作用于整个账号
			if mode != ModeMonitor && rec.UserID > 0 {
				applySanction(rec, f, model.SanctionTargetUser, strconv.Itoa(rec.UserID),
					model.SanctionBillingPenalty, billingPenaltyFactor(cfg), cfg.BillingPenaltyDurationMinutes)
			}

		case model.SanctionDisableToken:
			// 破坏性：仅 strict，且分数达标
			if mode == ModeStrict && rec.TokenID > 0 && f.Score >= cfg.AutoDisableScore {
				applyDisableToken(rec, f, f.Evidence)
				go notifyAdmin(rec, f, "auto disable token: risk score exceeded threshold")
			} else if mode != ModeMonitor && cfg.NotifyAdmin {
				go notifyAdmin(rec, f, "")
			}

		case model.SanctionBanIP:
			// 破坏性：仅 strict
			if mode == ModeStrict && rec.IP != "" {
				applySanction(rec, f, model.SanctionTargetIP, rec.IP,
					model.SanctionBanIP, 0, cfg.BanIPDurationMinutes)
				go notifyAdmin(rec, f, "auto ban ip: "+rec.IP)
			} else if mode != ModeMonitor && cfg.NotifyAdmin {
				go notifyAdmin(rec, f, "")
			}

		case model.SanctionBanUser:
			// 破坏性：仅 strict，且分数达标
			if mode == ModeStrict && rec.UserID > 0 && f.Score >= cfg.AutoBanScore {
				applyBanUser(rec, f)
				go notifyAdmin(rec, f, "auto ban: risk score exceeded threshold")
			} else if mode != ModeMonitor && cfg.NotifyAdmin {
				go notifyAdmin(rec, f, "")
			}

		default:
			// 未知 Action 回退为仅通知，避免误伤
			if mode != ModeMonitor && cfg.NotifyAdmin {
				go notifyAdmin(rec, f, "unknown action: "+action)
			}
		}
	}
}

// ── 处置力度取值 ──────────────────────────────────────────────────────
//
// 兜底逻辑统一收敛到 XJConfig.normalize()（policy.go），配置一加载就补齐，
// 这里只做取值。之前每个函数各自兜一遍默认值，同一个"默认 30 分钟"
// 在两个文件里各写一份，改的时候极容易漏掉一处喵～

func throttleFactor(cfg XJConfig) float64 {
	return cfg.ThrottleFactor
}

func penaltyRPM(cfg XJConfig) int {
	return cfg.PenaltyRPM
}

func suspendDuration(cfg XJConfig) int {
	return cfg.SuspendDurationMinutes
}

func billingPenaltyFactor(cfg XJConfig) float64 {
	return cfg.BillingPenaltyFactor
}

// applySanction 落一条制裁记录（玄鉴自动来源），失败只记录日志不影响主流程
func applySanction(rec RequestRecord, f Finding, targetType, targetKey, action string, factor float64, durationMinutes int) {
	reason := f.Type
	if f.Evidence != "" {
		reason = f.Type + ": " + f.Evidence
	}
	reason = truncateForLog(reason, 240)
	go func() {
		if err := sanction.Apply(targetType, targetKey, action, factor, reason, "xuanjian_auto", durationMinutes); err != nil {
			log.Printf("[xuanjian] 施加处置失败 action=%s target=%s/%s: %v", action, targetType, targetKey, err)
		}
	}()
}

// applyDisableToken 停用 Token：同时写制裁表（永久）+ 直接更新 Token 状态（即时生效）
func applyDisableToken(rec RequestRecord, f Finding, reason string) {
	if rec.TokenID <= 0 {
		return
	}
	applySanction(rec, f, model.SanctionTargetToken, strconv.Itoa(rec.TokenID), model.SanctionDisableToken, 0, 0)
	go executeDisableToken(rec.TokenID)
}

// applyBanUser 封禁用户：同时写制裁表（永久）+ 直接更新用户状态
func applyBanUser(rec RequestRecord, f Finding) {
	if rec.UserID <= 0 {
		return
	}
	applySanction(rec, f, model.SanctionTargetUser, strconv.Itoa(rec.UserID), model.SanctionBanUser, 0, 0)
	go executeBanUser(rec.UserID)
}

// truncateForLog 截断处置原因，防止超过 Sanction.Reason 字段长度
func truncateForLog(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes])
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
	// 同一 (token, 类型) 组合在冷却期内只发一次，别把管理员的收件箱冲垮了（见 notify_throttle.go）
	if !allowNotify(rec.TokenID, f.Type) {
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
