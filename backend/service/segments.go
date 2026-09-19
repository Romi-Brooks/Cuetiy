package service

import (
	"regexp"
	"strconv"
	"strings"
)

// ReplySegment 完整回复中的一个显式小段
type ReplySegment struct {
	Index   int    `json:"index"`
	SegID   string `json:"seg_id"`
	Content string `json:"content"`
	// Start/End 相对完整原文的 rune 下标（半开区间），便于后续流式打断引用
	Start int `json:"start"`
	End   int `json:"end"`
}

var (
	blankLineRe = regexp.MustCompile(`\n\s*\n+`)
	// 单独成行的显式分段符
	explicitSegRe = regexp.MustCompile(`(?m)^\s*(?:-{3,}|<<<SEG>>>|【段】)\s*$`)
)

const (
	segShortLineMax = 40
	segShortMinLine = 2
)

// SplitReplySegments 把一整段 AI 回复切分为显式小段（确定性规则，不依赖流式打断）。
//
// 规则（v1）：
//  1. 若存在显式分段符（单独一行的 --- / <<<SEG>>> / 【段】），优先按其切开
//  2. 否则按空行（一个及以上）切段
//  3. 若结果仍只有一段且含多行「短句」（≥2 行且每行 ≤40 rune），再按单个 \n 切开
//  4. 丢弃切完后的空白段；空原文返回空切片
//  5. 无显式边界时至少返回 1 段（整段原文）
//
// SegID 使用消息内稳定序号字符串（"0","1",...）；完整引用 id 为 message_id:seg_id。
func SplitReplySegments(content string) []ReplySegment {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	if strings.TrimSpace(normalized) == "" {
		return nil
	}

	var raws []string
	if explicitSegRe.MatchString(normalized) {
		raws = explicitSegRe.Split(normalized, -1)
	} else {
		raws = blankLineRe.Split(normalized, -1)
	}

	raws = compactSegments(raws)
	if len(raws) <= 1 && shouldSplitShortLines(normalized) {
		raws = compactSegments(strings.Split(normalized, "\n"))
	}
	if len(raws) == 0 {
		return nil
	}

	return buildSegments(normalized, raws)
}

func compactSegments(parts []string) []string {
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func shouldSplitShortLines(normalized string) bool {
	lines := strings.Split(normalized, "\n")
	n := 0
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		// 显式分段符行不算内容行
		if explicitSegRe.MatchString(t) {
			continue
		}
		if utf8Len(t) > segShortLineMax {
			return false
		}
		n++
	}
	return n >= segShortMinLine
}

func buildSegments(full string, raws []string) []ReplySegment {
	fullRunes := []rune(full)
	cursor := 0
	out := make([]ReplySegment, 0, len(raws))
	for i, raw := range raws {
		start := indexRunes(fullRunes, []rune(raw), cursor)
		if start < 0 {
			start = cursor
		}
		end := start + utf8Len(raw)
		if end > len(fullRunes) {
			end = len(fullRunes)
		}
		cursor = end
		out = append(out, ReplySegment{
			Index:   i,
			SegID:   strconv.Itoa(i),
			Content: raw,
			Start:   start,
			End:     end,
		})
	}
	return out
}

func indexRunes(haystack, needle []rune, from int) int {
	if len(needle) == 0 {
		return from
	}
	if from < 0 {
		from = 0
	}
	for i := from; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
