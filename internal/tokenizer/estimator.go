// internal/tokenizer/estimator.go
package tokenizer

import (
	"fmt"
	"unicode/utf8"
)

// EstimateResult Token 估算结果
type EstimateResult struct {
	TextLength int // 文本长度（字符数）
	TokenCount int // Token 数量
}

// Estimator Token 估算器
type Estimator struct{}

// NewEstimator 创建估算器
func NewEstimator() *Estimator {
	return &Estimator{}
}

// Estimate 估算文本的 Token 数量
func (e *Estimator) Estimate(text string) EstimateResult {
	result := EstimateResult{
		TextLength: utf8.RuneCountInString(text),
	}

	// 使用启发式估算
	result.TokenCount = e.estimateHeuristic(text)

	return result
}

// estimateHeuristic 启发式估算 Token 数量
// 根据 OpenAI 的经验公式：
// - 英文：约 4 字符 = 1 token
// - 中文：约 1.5 字符 = 1 token
// - 代码（混合）：约 3.5 字符 = 1 token
func (e *Estimator) estimateHeuristic(text string) int {
	if len(text) == 0 {
		return 0
	}

	var chineseCount int
	var englishCount int
	var digitCount int
	var symbolCount int
	var spaceCount int

	for _, r := range text {
		switch {
		case r >= 0x4E00 && r <= 0x9FFF:
			// 中文字符（CJK Unified Ideographs）
			chineseCount++
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			// 英文字母
			englishCount++
		case r >= '0' && r <= '9':
			// 数字
			digitCount++
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			// 空白字符
			spaceCount++
		default:
			// 其他符号（标点、特殊字符等）
			symbolCount++
		}
	}

	// Token 估算公式（基于实际测试调整）
	// - 英文单词：约 4 个字符 = 1 token
	// - 中文：约 1.5 个字符 = 1 token
	// - 数字：约 3 个字符 = 1 token
	// - 符号：约 2 个字符 = 1 token
	// - 空白：约 10 个 = 1 token
	tokens := float64(englishCount)/4.0 +
		float64(chineseCount)/1.5 +
		float64(digitCount)/3.0 +
		float64(symbolCount)/2.0 +
		float64(spaceCount)/10.0

	// 向上取整，加上少量余量
	return int(tokens) + 1
}

// FormatEstimate 格式化估算结果
func FormatEstimate(result EstimateResult) string {
	//var pages = float64(result.TokenCount) / 500.0 // 假设每页约 500 token
	return fmt.Sprintf("📊 ~%d tokens (based on %d characters)",
		result.TokenCount, result.TextLength)
}

// FormatEstimateDetailed 格式化详细估算结果
func FormatEstimateDetailed(result EstimateResult) string {
	pages := float64(result.TokenCount) / 500.0
	cost := GetTokenCost(result.TokenCount)

	return fmt.Sprintf(`📊 Token Estimate:
  Characters: %d
  Tokens: ~%d
  Pages: ~%.1f pages
  Cost: %s`,
		result.TextLength,
		result.TokenCount,
		pages,
		cost,
	)
}

// GetTokenCost 获取 Token 对应的费用估算（基于 GPT-4 价格）
func GetTokenCost(tokenCount int) string {
	// GPT-4 价格：输入 $0.03 / 1K tokens
	costPer1K := 0.03
	cost := float64(tokenCount) / 1000.0 * costPer1K
	return fmt.Sprintf("$%.4f (GPT-4)", cost)
}
