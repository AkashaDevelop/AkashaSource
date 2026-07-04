package xuanjian

// ～宸汐玄鉴·LLM 滥用检测～ ╰(*°▽°*)╯
// 不是防止模型说坏话（那是宸汐清源的活儿），
// 而是防止有人把大模型当"坏事生产工具"——
// 写恶意代码、帮逆向分析、批量生成钓鱼话术……
// 阿卡夏的镜子照得到。

// DetectLLMAbuse 检测利用 LLM 生成危害性内容的行为
func DetectLLMAbuse(rec RequestRecord, cfg XJConfig) []Finding {
	if !cfg.EnableLLMAbuse {
		return nil
	}

	rules := GetActiveRules()

	// prompt 侧扫描（恶意请求词）
	promptFindings := MatchRules(rec.PromptSnippet, true, rec.CompletionTokens, rules)

	// completion 侧扫描（看生成物里是否真的产出了危险内容）
	var completionFindings []Finding
	if rec.CompletionSnippet != "" {
		completionFindings = MatchRules(rec.CompletionSnippet, false, rec.CompletionTokens, rules)
		// completion 命中的分数额外 +15（说明不是询问而是真的生成了）
		for i := range completionFindings {
			completionFindings[i].Score += 15
			if completionFindings[i].Score > 100 {
				completionFindings[i].Score = 100
			}
			completionFindings[i].Evidence = "[completion] " + completionFindings[i].Evidence
		}
	}

	return append(promptFindings, completionFindings...)
}
