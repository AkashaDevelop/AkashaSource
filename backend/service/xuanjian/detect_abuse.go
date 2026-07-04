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
	if p.QuotaBurned > cfg.MaxQuotaPerWin {
		findings = append(findings, Finding{
			Type:     "quota_burn",
			Group:    string(GroupLLMJacking),
			Score:    scoreByRatio64(p.QuotaBurned, cfg.MaxQuotaPerWin, 65, 88),
			Evidence: fmt.Sprintf("quota=%d, limit=%d in %dmin", p.QuotaBurned, cfg.MaxQuotaPerWin, cfg.WindowMinutes),
			Action:   "notify",
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
	if len(p.IPCIDRSet) > cfg.MaxIPCIDRsPerWin {
		findings = append(findings, Finding{
			Type:     "high_ip_rotation",
			Group:    string(GroupAgentAbuse),
			Score:    70,
			Evidence: fmt.Sprintf("unique /24 cidrs=%d in %dmin", len(p.IPCIDRSet), cfg.WindowMinutes),
			Action:   "notify",
		})
	}

	// ── 跨 Token 轮换（修正后：8 个不同 token 才触发）─────────────────────
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
				Action:   "notify",
			})
		}
	}

	// ── LLMjacking：新 token + 算力暴燃组合 ───────────────────────────────
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
					Action: "notify",
				})
			}
		}
	}

	// ── LLMjacking：Scanner 阶段（极短 prompt 后突增）──────────────────────
	// 前3次请求都是极短 prompt，且第5次开始请求数暴增
	if p.RequestCount >= 5 && p.ShortPromptCount >= 3 {
		shortRatio := float64(p.ShortPromptCount) / float64(p.RequestCount)
		// Scanner 特征：短 prompt 比例极高但请求总量也很多（说明已进入主力阶段）
		if shortRatio < 0.1 && p.RequestCount > 50 {
			findings = append(findings, Finding{
				Type:     "llmjacking_scanner",
				Group:    string(GroupLLMJacking),
				Score:    70,
				Evidence: fmt.Sprintf("started with %d short probes, now %d total requests", p.ShortPromptCount, p.RequestCount),
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
