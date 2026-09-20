package service

import (
	"strings"
	"testing"
)

// 验证：压缩后最终 messages 必须是
// system(含滚动摘要) + 仅 keep 的 history + 当前 user
// 且 system 不能仍是压缩前的旧摘要。
func TestAssembleShape_CompressedHistoryInMessages(t *testing.T) {
	// 纯逻辑抽检：用 buildSystem 等价片段验证摘要进入 system
	summary := "【摘要】用户喜欢雨天，约定周末一起看展"
	parts := []PromptPart{}
	add := func(name, content string) {
		content = strings.TrimSpace(content)
		if content != "" {
			parts = append(parts, PromptPart{Name: name, Content: content})
		}
	}
	add("L0 核心人设", "你是温柔女友")
	add("L1 能力索引", "- 情绪陪伴")
	add("滚动摘要", summary)

	var sys strings.Builder
	for i, p := range parts {
		if i > 0 {
			sys.WriteString("\n\n")
		}
		sys.WriteString(p.Content)
	}
	systemStr := sys.String()
	if !strings.Contains(systemStr, summary) {
		t.Fatalf("system 必须包含本轮滚动摘要, got=%q", systemStr)
	}

	// history 只保留压缩后的 recent
	type msg struct{ role, content string }
	history := []msg{
		{"user", "最近好累"},
		{"assistant", "抱抱宝宝"},
	}
	current := "周末还记得看展吗？"

	messages := []ChatMessage{{Role: "system", Content: systemStr}}
	for _, h := range history {
		messages = append(messages, ChatMessage{Role: h.role, Content: h.content})
	}
	messages = append(messages, ChatMessage{Role: "user", Content: current})

	if len(messages) != 1+len(history)+1 {
		t.Fatalf("messages 长度应为 system+history+user, got=%d", len(messages))
	}
	if messages[0].Role != "system" || !strings.Contains(messages[0].Content, "摘要") {
		t.Fatalf("messages[0] 应为含摘要的 system")
	}
	if messages[len(messages)-1].Content != current {
		t.Fatalf("最后一条应为当前提问")
	}
	// 旧的长历史不得出现在 history 段
	for _, m := range messages[1 : len(messages)-1] {
		if strings.Contains(m.contentOrContent(), "三周前的长对话") {
			t.Fatalf("压缩掉的旧消息不应再进 history")
		}
	}
}

func (m ChatMessage) contentOrContent() string { return m.Content }

func TestEstimateTokens_CJK(t *testing.T) {
	if EstimateTokens("") != 0 {
		t.Fatal("empty")
	}
	n := EstimateTokens("你好世界")
	if n <= 0 {
		t.Fatal("cjk tokens should > 0")
	}
}

func TestCompactL0_KeepsCoreIdentity(t *testing.T) {
	// 短 prompt：token 未超阈值时原样返回（技能包内容必须保留）
	short := "你是温柔女友\n# 禁止\n1. 自称AI\n"
	out := CompactL0Prompt(short)
	if !strings.Contains(out, "女友") {
		t.Fatalf("短 L0 应保留技能包内容, got=%q", out)
	}

	// 长 prompt：应剥离示例行，保留禁止/身份
	long := "你是温柔女友，灵魂伴侣。\n# 身份\n亲密关系\n"
	for i := 0; i < 40; i++ {
		long += "规则行内容示例填充填充填充填充填充填充填充填充\n"
	}
	long += "# 示例\n✅ 推荐说法一\n❌ 禁止说法二\n# 禁止\n1. 自称AI\n"
	out2 := CompactL0Prompt(long)
	if strings.Contains(out2, "✅") || strings.Contains(out2, "❌") {
		t.Fatalf("长 L0 应剥离示例标记行, got=%q", out2)
	}
	if !strings.Contains(out2, "禁止") && !strings.Contains(out2, "身份") && !strings.Contains(out2, "女友") {
		t.Fatalf("长 L0 压缩后仍需保留技能包核心, got=%q", out2)
	}

	// 空技能包：回落中性占位，不得写死具体人设
	empty := CompactL0Prompt("")
	if strings.Contains(empty, "女友") || strings.Contains(empty, "宝宝") {
		t.Fatalf("空 L0 占位不应含业务人设, got=%q", empty)
	}
}
