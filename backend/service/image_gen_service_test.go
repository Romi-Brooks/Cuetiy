package service

import (
	"strings"
	"testing"
	"time"

	"cuetiy-backend/model"
	"cuetiy-backend/skill"
)

const imageSkillMD = `---
name: Image-Appearance
category: image_appearance
load_mode: trigger
triggers:
  intents: [want_image]
  tags: [want_image]
  keywords: [想看看你, 发张照片, 自拍]
appearance: "黑色长发，杏眼，清甜气质"
image_style: "手机自拍，自然光，生活场景"
---

# 出图
长什么样写在 frontmatter appearance
`

func testImageRegistry() *skill.SkillRegistry {
	files := []model.PersonaFile{{FileName: "Image-Appearance.md", ModuleCategory: "image_appearance"}}
	return skill.BuildRegistry(1, files, func(pf *model.PersonaFile) string { return imageSkillMD })
}

func TestImageTrigger_FromSkillKeywords(t *testing.T) {
	reg := testImageRegistry()
	if !reg.ShouldTriggerImage("宝宝想看看你") {
		t.Fatal("keywords from skill should trigger")
	}
	if reg.ShouldTriggerImage("今天好累") {
		t.Fatal("unrelated should not trigger")
	}
	kws := reg.ImageTriggerKeywords()
	if len(kws) < 3 {
		t.Fatalf("keywords=%v", kws)
	}
}

func TestImageTrigger_Tags(t *testing.T) {
	reg := testImageRegistry()
	if !reg.ShouldTriggerImageTags([]string{"sad", "want_image"}) {
		t.Fatal("want_image tag should trigger")
	}
	if reg.ShouldTriggerImageTags([]string{"sad"}) {
		t.Fatal("no image tag should not trigger")
	}
}

func TestBuildImagePrompt_UsesSkillAppearance(t *testing.T) {
	reg := testImageRegistry()
	p := BuildImagePromptWithReg("给你看～", "flirty", "", "", reg)
	if !strings.Contains(p, "黑色长发") {
		t.Fatalf("appearance from skill missing: %s", p)
	}
	if !strings.Contains(p, "手机自拍") {
		t.Fatalf("style from skill missing: %s", p)
	}
	// 引擎不应再写死「恋人从相册」等业务外貌
	if strings.Contains(p, "宝宝") {
		t.Fatalf("engine should not hardcode persona nick: %s", p)
	}
}

func TestBuildImagePrompt_OverrideWins(t *testing.T) {
	md := strings.Replace(imageSkillMD, "---\n\n# 出图", "image_prompt: \"整段覆盖promptXYZ\"\n---\n\n# 出图", 1)
	files := []model.PersonaFile{{FileName: "Image-Appearance.md"}}
	reg := skill.BuildRegistry(1, files, func(pf *model.PersonaFile) string { return md })
	p := BuildImagePromptWithReg("hi", "", "", "", reg)
	if !strings.Contains(p, "整段覆盖promptXYZ") {
		t.Fatalf("override missing: %s", p)
	}
}

func TestShouldTrigger_ServiceUsesRegistry(t *testing.T) {
	s := NewImageGenService(nil, nil)
	reg := testImageRegistry()
	s.BindRegistry(func(int64) *skill.SkillRegistry { return reg })
	ok, src := s.ShouldTrigger("想看看你呀", 1, nil)
	if !ok || src != "skill_keyword" {
		t.Fatalf("ok=%v src=%s", ok, src)
	}
	ok, src = s.ShouldTrigger("随便聊聊", 1, []string{"want_image"})
	if !ok || src != "skill_tag" {
		t.Fatalf("tag path ok=%v src=%s", ok, src)
	}
	ok, src = s.ShouldTrigger("今天天气不错", 1, nil)
	if ok {
		t.Fatalf("should not trigger src=%s", src)
	}
}

func TestAllowImage_StillLimits(t *testing.T) {
	s := NewImageGenService(nil, nil)
	ok, _ := s.AllowImage(9, 2, time.Minute)
	if !ok {
		t.Fatal("first")
	}
	ok, _ = s.AllowImage(9, 2, time.Minute)
	if !ok {
		t.Fatal("second")
	}
	ok, _ = s.AllowImage(9, 2, time.Minute)
	if ok {
		t.Fatal("third should block")
	}
}
