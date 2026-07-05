package billingexpr

import (
	"STfreApi/common"
	"fmt"
)

// Settle 使用分层表达式结算
// exprStr: 表达式字符串
// inputTokens: 输入 token 数
// outputTokens: 输出 token 数
// groupRatio: 分组倍率
// 返回: quota, error
func Settle(exprStr string, inputTokens, outputTokens int, groupRatio float64) (int64, error) {
	expr, err := ParseExpr(exprStr)
	if err != nil {
		return 0, fmt.Errorf("parse expr failed: %w", err)
	}
	if expr == nil {
		return 0, nil
	}

	output, err := expr.Evaluate(inputTokens, outputTokens)
	if err != nil {
		return 0, fmt.Errorf("eval expr failed: %w", err)
	}

	// quota = exprOutput / 1_000_000 * QuotaPerUnit * groupRatio
	quota := int64(output / 1_000_000 * common.QuotaPerUnit * groupRatio)
	if quota < 0 {
		quota = 0
	}
	return quota, nil
}
