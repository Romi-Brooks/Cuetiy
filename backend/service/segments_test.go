package service

import (
	"strconv"
	"strings"
	"testing"
)

func TestSplitReplySegments_Empty(t *testing.T) {
	if got := SplitReplySegments(""); got != nil {
		t.Fatalf("%v", got)
	}
	if got := SplitReplySegments("   \n\n  "); got != nil {
		t.Fatalf("%v", got)
	}
}

func TestSplitReplySegments_Single(t *testing.T) {
	got := SplitReplySegments("宝宝好呀～今天也要开心喔")
	if len(got) != 1 {
		t.Fatalf("len=%d %v", len(got), got)
	}
	if got[0].Index != 0 || got[0].SegID != "0" {
		t.Fatalf("%+v", got[0])
	}
	if got[0].Content != "宝宝好呀～今天也要开心喔" {
		t.Fatalf("content=%q", got[0].Content)
	}
}

func TestSplitReplySegments_BlankLines(t *testing.T) {
	src := "第一段呀\n\n第二段喔\n\n\n第三段啦"
	got := SplitReplySegments(src)
	if len(got) != 3 {
		t.Fatalf("len=%d %v", len(got), got)
	}
	want := []string{"第一段呀", "第二段喔", "第三段啦"}
	for i, w := range want {
		if got[i].Content != w {
			t.Fatalf("seg%d=%q want %q", i, got[i].Content, w)
		}
		if got[i].SegID != strconv.Itoa(i) {
			t.Fatalf("seg id %q", got[i].SegID)
		}
	}
}

func TestSplitReplySegments_ShortLines(t *testing.T) {
	src := "宝宝在吗\n我跟你说件事\n今天超开心的"
	got := SplitReplySegments(src)
	if len(got) != 3 {
		t.Fatalf("len=%d %v", len(got), got)
	}
}

func TestSplitReplySegments_LongLineNoSubsplit(t *testing.T) {
	long := strings.Repeat("这是一段比较长的说明内容", 10)
	src := long + "\n但第二段是正常换行短句"
	got := SplitReplySegments(src)
	// 空行不存在；第二行短但第一行超长 → 整段保持
	if len(got) != 1 {
		t.Fatalf("len=%d %v", len(got), got)
	}
}

func TestSplitReplySegments_ExplicitMarker(t *testing.T) {
	src := "开场白\n---\n专业解答部分\n【段】\n收尾啦"
	got := SplitReplySegments(src)
	if len(got) != 3 {
		t.Fatalf("len=%d %#v", len(got), got)
	}
	if !strings.Contains(got[1].Content, "专业解答") {
		t.Fatalf("seg1=%q", got[1].Content)
	}
}

func TestSplitReplySegments_OffsetsCover(t *testing.T) {
	src := "你好呀\n\n我在呢"
	got := SplitReplySegments(src)
	if len(got) != 2 {
		t.Fatal(len(got))
	}
	full := []rune(src)
	if got[0].Start != 0 || got[0].End != utf8Len("你好呀") {
		t.Fatalf("s0 %+v", got[0])
	}
	if string(full[got[0].Start:got[0].End]) != "你好呀" {
		t.Fatalf("offset mismatch")
	}
	if string(full[got[1].Start:got[1].End]) != "我在呢" {
		t.Fatalf("offset1 mismatch %+v", got[1])
	}
}
