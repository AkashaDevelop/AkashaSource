package controller

// 🌸 模型广场公开统计～从真实日志聚合模型的性能表现
// （new-api 这部分是前端 mock 的假数据，我们直接上真材实料哦）

import (
	"strings"
	"time"

	"STfreApi/common"
	"STfreApi/model"

	"github.com/gin-gonic/gin"
)

// GetPublicModelStats GET /api/pricing/stats?model=xxx
// 返回该模型近 7 天的聚合表现：请求数、成功率、平均延迟、吞吐（TPS）
func GetPublicModelStats(c *gin.Context) {
	modelName := strings.TrimSpace(c.Query("model"))
	if modelName == "" {
		common.Fail(c, common.CodeParamError, "缺少 model 参数")
		return
	}

	since := time.Now().AddDate(0, 0, -7).Unix()
	sinceDay := time.Now().AddDate(0, 0, -1).Unix()

	// ～消费日志 = 成功请求；失败日志单独统计～
	type aggRow struct {
		Cnt              int64
		SumUseTime       int64
		SumCompletionTok int64
		SumPromptTok     int64
	}

	var ok aggRow
	_ = common.DB.Model(&model.Log{}).
		Select("COUNT(*) as cnt, COALESCE(SUM(use_time),0) as sum_use_time, COALESCE(SUM(completion_tokens),0) as sum_completion_tok, COALESCE(SUM(prompt_tokens),0) as sum_prompt_tok").
		Where("model_name = ? AND type = ? AND created_at >= ?", modelName, model.LogTypeConsume, since).
		Scan(&ok).Error

	var failCnt int64
	_ = common.DB.Model(&model.Log{}).
		Where("model_name = ? AND type = ? AND created_at >= ?", modelName, model.LogTypeFail, since).
		Count(&failCnt).Error

	var cnt24h int64
	_ = common.DB.Model(&model.Log{}).
		Where("model_name = ? AND type = ? AND created_at >= ?", modelName, model.LogTypeConsume, sinceDay).
		Count(&cnt24h).Error

	// ～算平均：没数据就都给 0，前端显示"暂无"～
	total := ok.Cnt + failCnt
	var successRate float64
	if total > 0 {
		successRate = float64(ok.Cnt) / float64(total) * 100
	}
	var avgLatencyMs int64
	var tps float64
	if ok.Cnt > 0 && ok.SumUseTime > 0 {
		avgLatencyMs = ok.SumUseTime * 1000 / ok.Cnt // use_time 单位是秒
		tps = float64(ok.SumCompletionTok) / float64(ok.SumUseTime)
	}

	// ～近 7 天按天的请求量，做个迷你趋势～
	// 用 created_at/86400 分桶，MySQL/SQLite/PG 通吃，不碰方言函数～
	type bucketRow struct {
		Bucket int64
		Cnt    int64
	}
	var buckets []bucketRow
	_ = common.DB.Model(&model.Log{}).
		Select("created_at / 86400 as bucket, COUNT(*) as cnt").
		Where("model_name = ? AND type = ? AND created_at >= ?", modelName, model.LogTypeConsume, since).
		Group("bucket").Order("bucket").
		Scan(&buckets).Error

	type dayRow struct {
		Day string `json:"day"`
		Cnt int64  `json:"count"`
	}
	days := make([]dayRow, 0, len(buckets))
	for _, b := range buckets {
		days = append(days, dayRow{
			Day: time.Unix(b.Bucket*86400, 0).UTC().Format("01-02"),
			Cnt: b.Cnt,
		})
	}

	common.OK(c, gin.H{
		"model":          modelName,
		"window_days":    7,
		"request_count":  total,
		"success_count":  ok.Cnt,
		"fail_count":     failCnt,
		"success_rate":   successRate,
		"avg_latency_ms": avgLatencyMs,
		"tps":            tps,
		"requests_24h":   cnt24h,
		"total_tokens":   ok.SumPromptTok + ok.SumCompletionTok,
		"daily":          days,
	})
}
