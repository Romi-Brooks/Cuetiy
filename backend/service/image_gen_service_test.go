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

func TestAllowImage_ZeroMaxDisablesLimit(t *testing.T) {
	s := NewImageGenService(nil, nil)
	// max<=0 视为不限流（30 分钟窗暂时关闭）
	for i := 0; i < 5; i++ {
		ok, _ := s.AllowImage(10, 0, 0)
		if !ok {
			t.Fatalf("iter %d should allow when max=0", i)
		}
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

func TestComposeImagePrompt_SkillsPlusLLM(t *testing.T) {
	reg := testImageRegistry()
	llm := &ImagePromptLLM{
		SceneType: "landscape",
		Prompt:    "黄昏窗外的城市天际线，暖橙色云层，玻璃上有雨滴",
		Caption:   "等我拍一下窗外～",
		HasPerson: false,
	}
	p, src := ComposeImagePrompt("好美", "joy", reg, llm)
	if src != "skills+llm" {
		t.Fatalf("src=%s", src)
	}
	if !strings.Contains(p, "黄昏窗外") {
		t.Fatalf("llm scene missing: %s", p)
	}
	// 无人入画时不应拼接 appearance
	if strings.Contains(p, "黑色长发") {
		t.Fatalf("appearance should not appear when has_person=false: %s", p)
	}
	if !strings.Contains(p, "不要在画面中出现文字水印") {
		t.Fatalf("watermark suffix missing: %s", p)
	}
}

func TestComposeImagePrompt_SkillsPlusLLM_WithPerson(t *testing.T) {
	reg := testImageRegistry()
	llm := &ImagePromptLLM{
		SceneType: "selfie",
		Prompt:    "午后咖啡馆窗边，半身构图，自然逆光，手里拿着马克杯",
		HasPerson: true,
	}
	p, src := ComposeImagePrompt("在喝咖啡", "joy", reg, llm)
	if src != "skills+llm" {
		t.Fatalf("src=%s", src)
	}
	if !strings.Contains(p, "黑色长发") {
		t.Fatalf("skills appearance must be injected: %s", p)
	}
	if !strings.Contains(p, "咖啡馆") {
		t.Fatalf("llm scene missing: %s", p)
	}
}

func TestComposeImagePrompt_LLMNilFallsBackToSkills(t *testing.T) {
	reg := testImageRegistry()
	p, src := ComposeImagePrompt("给你看～", "flirty", reg, nil)
	if src != "skills" {
		t.Fatalf("src=%s", src)
	}
	if !strings.Contains(p, "手机自拍") || !strings.Contains(p, "黑色长发") {
		t.Fatalf("skills fallback broken: %s", p)
	}
}

func TestWaitPhraseFromLLM_PrefersCaption(t *testing.T) {
	got := WaitPhraseFromLLM(&ImagePromptLLM{Caption: "马上给你看窗外～"})
	if got != "马上给你看窗外～" {
		t.Fatalf("got=%q", got)
	}
	got = WaitPhraseFromLLM(nil)
	if got == "" {
		t.Fatal("fallback empty")
	}
}

func TestParseImagePromptLLM(t *testing.T) {
	out, err := parseImagePromptLLM("```json\n{\"scene_type\":\"food\",\"prompt\":\"街角拉面店热气\",\"caption\":\"等我去买～\",\"has_person\":false}\n```")
	if err != nil {
		t.Fatal(err)
	}
	if out.SceneType != "food" || !strings.Contains(out.Prompt, "拉面") {
		t.Fatalf("%+v", out)
	}
}
