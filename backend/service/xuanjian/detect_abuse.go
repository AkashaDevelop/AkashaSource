package xuanjian

// ～宸汐玄鉴·行为滥用检测～
// 盯着速率、算力、IP轮换、LLMjacking 这些纯平台层威胁 (｡•ᴗ•｡)
// 跟内容无关，只看"用了多少、用了多快、从哪里用"。

import (
	"fmt"
	"time"
)

// DetectAbuse 基于画像检测行为滥用
func DetectAbuse(p *TokenProfile, up *UserProfile, rec RequestRecord, cfg XJConfig) []Finding {
	if !cfg.EnableAbuseDetection {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	var findings []Finding

	// ── 速率暴增 ──────────────────────────────────────────────────────────
	if p.RequestCount > cfg.MaxRequestsPerWin {
		findings = append(findings, Finding{
			Type:     "rate_spike",
			Group:    string(GroupLLMJacking),
			Score:    scoreByRatio(p.RequestCount, cfg.MaxRequestsPerWin, 60, 85),
			Evidence: fmt.Sprintf("requests=%d, limit=%d in %dmin window", p.RequestCount, cfg.MaxRequestsPerWin, cfg.WindowMinutes),
			Action:   "throttle",
		})
	}

	// ── Quota 暴燃 ────────────────────────────────────────────────────────
	// 处置用 billing_penalty：烧算力的人就让他多付钱，
	// 既直击动机又完全不影响正常业务，是这类滥用最贴切的回应喵～
	if p.QuotaBurned > cfg.MaxQuotaPerWin {
		findings = append(findings, Finding{
			Type:     "quota_burn",
			Group:    string(GroupLLMJacking),
			Score:    scoreByRatio64(p.QuotaBurned, cfg.MaxQuotaPerWin, 65, 88),
			Evidence: fmt.Sprintf("quota=%d, limit=%d in %dmin", p.QuotaBurned, cfg.MaxQuotaPerWin, cfg.WindowMinutes),
			Action:   "billing_penalty",
		})
	}

	// ── 多模型扫描 ────────────────────────────────────────────────────────
	if len(p.ModelSet) > cfg.MaxModelsPerWin {
		findings = append(findings, Finding{
			Type:     "model_scanning",
			Group:    string(GroupReverseEng),
			Score:    60,
			Evidence: fmt.Sprintf("unique models=%d in %dmin", len(p.ModelSet), cfg.WindowMinutes),
			Action:   "warn",
		})
	}

	// ── IP CIDR 轮换（修正后：15 个 /24 CIDR 才触发）──────────────────────
	// 处置用 rpm_limit 而不是 ban_ip：出口 IP 很可能是 NAT/CGNAT 网关，
	// 封 IP 会连坐一整栋楼的无辜用户，限速才是对症下药～
	if len(p.IPCIDRSet) > cfg.MaxIPCIDRsPerWin {
		findings = append(findings, Finding{
			Type:     "high_ip_rotation",
			Group:    string(GroupAgentAbuse),
			Score:    70,
			Evidence: fmt.Sprintf("unique /24 cidrs=%d in %dmin", len(p.IPCIDRSet), cfg.WindowMinutes),
			Action:   "rpm_limit",
		})
	}

	// ── 跨 Token 轮换（修正后：8 个不同 token 才触发）─────────────────────
	// 这是 LLMjacking 最典型的特征（拿到一批泄漏的 key 轮着用），
	// suspend_token 到点自动恢复，误判了也不会造成永久损失
	if up != nil {
		up.mu.Lock()
		tokenCount := len(up.TokenIDSet)
		up.mu.Unlock()
		if tokenCount > cfg.MaxTokensPerUser {
			findings = append(findings, Finding{
				Type:     "token_rotation",
				Group:    string(GroupLLMJacking),
				Score:    75,
				Evidence: fmt.Sprintf("user %d used %d different tokens in %dmin", rec.UserID, tokenCount, cfg.WindowMinutes),
				Action:   "suspend_token",
			})
		}
	}

	// ── LLMjacking：新 token + 算力暴燃组合 ───────────────────────────────
	// 全新的 token 一上来就狂烧算力，几乎只有一种解释：key 泄漏被人捡去用了。
	// suspend_token 先把出血口按住，管理员收到通知后再决定是否恢复～
	if cfg.LLMJackingNewTokenHours > 0 {
		tokenAge := time.Since(rec.TokenCreatedAt)
		newTokenThreshold := time.Duration(cfg.LLMJackingNewTokenHours) * time.Hour
		if tokenAge < newTokenThreshold {
			// 计算同期平均消耗（粗略：假设平均用户每天消耗约 100k token，新token 用10倍认定异常）
			expectedQuota := int64(100_000) * int64(tokenAge.Hours()) / 24
			if expectedQuota < 10_000 {
				expectedQuota = 10_000 // 最低基准
			}
			if cfg.LLMJackingQuotaMultiple > 0 &&
				float64(p.TotalLifetimeQuota) > float64(expectedQuota)*cfg.LLMJackingQuotaMultiple {
				findings = append(findings, Finding{
					Type:  "llmjacking_compute_burst",
					Group: string(GroupLLMJacking),
					Score: 85,
					Evidence: fmt.Sprintf("new token (age=%s), quota=%d >> expected=%d",
						tokenAge.Round(time.Minute), p.TotalLifetimeQuota, expectedQuota),
					Action: "suspend_token",
				})
			}
		}
	}

	// ── LLMjacking：Scanner 阶段（先用极短 prompt 探路，再切换到主力消耗）──
	//
	// ～2026.8.4 修正：原条件写的是 shortRatio < 0.1（短 prompt 占比**低于** 10%），
	// 这跟"scanner 特征"完全相反——真正的探路阶段短 prompt 占比应该很**高**才对。
	// 按原逻辑，一个短 prompt 占比极低的正常用户只要请求量上了 50 就会被扣帽子，
	// 而真正的扫描行为反倒被条件挡在门外，抓错了人还漏了贼 (￣▽￣;)
	//
	// 现在改成正确的语义：探路期短 prompt 扎堆（占比 ≥ 30%）+ 总量已经起来了。
	if p.RequestCount >= 20 && p.ShortPromptCount >= 5 {
		shortRatio := float64(p.ShortPromptCount) / float64(p.RequestCount)
		if shortRatio >= 0.3 {
			findings = append(findings, Finding{
				Type:     "llmjacking_scanner",
				Group:    string(GroupLLMJacking),
				Score:    70,
				Evidence: fmt.Sprintf("%d short probes out of %d requests (%.0f%%)", p.ShortPromptCount, p.RequestCount, shortRatio*100),
				Action:   "notify",
			})
		}
	}

	// ── 高错误率（疑似在探测边界）──────────────────────────────────────────
	if p.RequestCount > 20 {
		errorRate := float64(p.ErrorCount) / float64(p.RequestCount)
		if errorRate > 0.5 {
			findings = append(findings, Finding{
				Type:     "high_error_rate",
				Group:    string(GroupReverseEng),
				Score:    50,
				Evidence: fmt.Sprintf("error_rate=%.0f%% (%d/%d)", errorRate*100, p.ErrorCount, p.RequestCount),
				Action:   "warn",
			})
		}
	}

	return findings
}

// scoreByRatio 按超出比例线性插值计算风险分
func scoreByRatio(actual, limit, minScore, maxScore int) int {
	if limit <= 0 {
		return maxScore
	}
	ratio := float64(actual) / float64(limit)
	score := int(float64(minScore) + (float64(maxScore-minScore) * (ratio - 1.0)))
	if score > maxScore {
		score = maxScore
	}
	return score
}

func scoreByRatio64(actual, limit int64, minScore, maxScore int) int {
	if limit <= 0 {
		return maxScore
	}
	ratio := float64(actual) / float64(limit)
	score := int(float64(minScore) + (float64(maxScore-minScore) * (ratio - 1.0)))
	if score > maxScore {
		score = maxScore
	}
	return score
}
