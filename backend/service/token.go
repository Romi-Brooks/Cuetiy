package service

import (
	"unicode/utf8"
)

// EstimateTokens 粗估 token 数：中文约 1.6 字/token，ASCII 约 4 字符/token。
// 用于预算与压缩触发，故意偏保守（略高估）。
func EstimateTokens(s string) int {
	if s == "" {
		return 0
	}
	var cjk, other int
	for _, r := range s {
		if r > 0x2E80 {
			cjk++
		} else {
			other++
		}
	}
	t := int(float64(cjk)*1.6 + float64(other)/4.0)
	if t < 1 {
		t = 1
	}
	return t
}

// EstimateRunes 兼容辅助
func EstimateRunes(s string) int {
	return utf8.RuneCountInString(s)
}

// TruncateRunes 按字符截断，保留前 n 个 rune
func TruncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…[已截断]"
}
