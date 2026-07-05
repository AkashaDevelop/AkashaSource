package billingexpr

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// TieredExpr 分层表达式
type TieredExpr struct {
	Expr  string `json:"expr"`            // 表达式字符串
	Tiers []Tier `json:"tiers,omitempty"` // 分层定价
}

// Tier 分层定价区间
type Tier struct {
	Min   float64 `json:"min"`   // 最小 token 数
	Max   float64 `json:"max"`   // 最大 token 数（-1 表示无限）
	Price float64 `json:"price"` // 价格系数（$/1M tokens）
}

// ParseExpr 解析表达式字符串
func ParseExpr(exprStr string) (*TieredExpr, error) {
	exprStr = strings.TrimSpace(exprStr)
	if exprStr == "" {
		return nil, nil
	}

	// 尝试解析为分层 JSON
	if strings.HasPrefix(exprStr, "[") || strings.HasPrefix(exprStr, "{") {
		var tiers []Tier
		if err := json.Unmarshal([]byte(exprStr), &tiers); err != nil {
			return nil, fmt.Errorf("invalid tiered expr: %w", err)
		}
		return &TieredExpr{Expr: exprStr, Tiers: tiers}, nil
	}

	// 简单表达式模式：如 "2.5" 或 "2.5*input+10*output"
	return &TieredExpr{Expr: exprStr}, nil
}

// Evaluate 计算表达式输出
// inputTokens: 输入 token 数
// outputTokens: 输出 token 数
// 返回值需要除以 1_000_000 再乘以 QuotaPerUnit
func (e *TieredExpr) Evaluate(inputTokens, outputTokens int) (float64, error) {
	if e == nil || e.Expr == "" {
		return 0, nil
	}

	// 分层定价模式
	if len(e.Tiers) > 0 {
		totalTokens := float64(inputTokens + outputTokens)
		var result float64
		for _, tier := range e.Tiers {
			if totalTokens >= tier.Min && (tier.Max < 0 || totalTokens < tier.Max) {
				result = tier.Price * totalTokens
				return result, nil
			}
		}
		// 没有匹配的分层，使用最后一个
		if len(e.Tiers) > 0 {
			last := e.Tiers[len(e.Tiers)-1]
			result = last.Price * totalTokens
			return result, nil
		}
		return 0, nil
	}

	// 简单表达式模式
	// 支持 input 和 output 变量，以及基本的四则运算
	result, err := evalSimpleExpr(e.Expr, float64(inputTokens), float64(outputTokens))
	if err != nil {
		return 0, err
	}
	return result, nil
}

// evalSimpleExpr 简单表达式求值
// 支持: input, output 变量, 数字, +, -, *, /
func evalSimpleExpr(expr string, input, output float64) (float64, error) {
	// 替换变量
	expr = strings.ReplaceAll(expr, "input", fmt.Sprintf("%g", input))
	expr = strings.ReplaceAll(expr, "output", fmt.Sprintf("%g", output))

	// 验证表达式只包含数字、运算符和小数点
	matched, _ := regexp.MatchString(`^[\d\s+\-*/().]+$`, expr)
	if !matched {
		return 0, fmt.Errorf("invalid expression: %s", expr)
	}

	// 简单的递归下降解析器
	tokens := tokenize(expr)
	parser := &exprParser{tokens: tokens, pos: 0}
	result, err := parser.parseExpr()
	if err != nil {
		return 0, err
	}
	return result, nil
}

func tokenize(s string) []string {
	var tokens []string
	i := 0
	for i < len(s) {
		c := s[i]
		if c == ' ' || c == '\t' {
			i++
			continue
		}
		if c == '+' || c == '-' || c == '*' || c == '/' || c == '(' || c == ')' {
			tokens = append(tokens, string(c))
			i++
		} else if (c >= '0' && c <= '9') || c == '.' {
			j := i + 1
			for j < len(s) && ((s[j] >= '0' && s[j] <= '9') || s[j] == '.') {
				j++
			}
			tokens = append(tokens, s[i:j])
			i = j
		} else {
			i++
		}
	}
	return tokens
}

type exprParser struct {
	tokens []string
	pos    int
}

func (p *exprParser) peek() string {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return ""
}

func (p *exprParser) consume() string {
	tok := p.peek()
	p.pos++
	return tok
}

func (p *exprParser) parseExpr() (float64, error) {
	left, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for {
		op := p.peek()
		if op != "+" && op != "-" {
			break
		}
		p.consume()
		right, err := p.parseTerm()
		if err != nil {
			return 0, err
		}
		if op == "+" {
			left += right
		} else {
			left -= right
		}
	}
	return left, nil
}

func (p *exprParser) parseTerm() (float64, error) {
	left, err := p.parseFactor()
	if err != nil {
		return 0, err
	}
	for {
		op := p.peek()
		if op != "*" && op != "/" {
			break
		}
		p.consume()
		right, err := p.parseFactor()
		if err != nil {
			return 0, err
		}
		if op == "*" {
			left *= right
		} else {
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			left /= right
		}
	}
	return left, nil
}

func (p *exprParser) parseFactor() (float64, error) {
	tok := p.peek()
	if tok == "(" {
		p.consume()
		val, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		if p.peek() != ")" {
			return 0, fmt.Errorf("expected )")
		}
		p.consume()
		return val, nil
	}
	if tok == "-" {
		p.consume()
		val, err := p.parseFactor()
		if err != nil {
			return 0, err
		}
		return -val, nil
	}
	p.consume()
	return strconv.ParseFloat(tok, 64)
}
