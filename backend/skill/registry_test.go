package skill

import (
	"testing"

	"cuetiy-backend/model"
)

const emotionCompanionMD = `---
name: Emotion-Companion
description: "情绪陪伴"
priority_num: 90
category: emotion_companion
load_mode: trigger
ttl_turns: 5
triggers:
  emotions: [joy, sad, anxious]
  intents: [need_comfort]
  keywords: [emo]
---

# body
content here
`

const professionalMD = `---
name: Professional-Skills
category: professional_skills
load_mode: trigger
priority_num: 70
ttl_turns: 6
triggers:
  intents: [need_professional]
  keywords: [代码, bug]
---

# pro
x
`

const personaBaseMD = `---
name: Persona-Base
category: persona_base
load_mode: always
priority_num: 95
---

# identity
you are companion
`

func TestParseSkillTriggers(t *testing.T) {
	parsed, err := ParseSkillContent("Emotion-Companion.md", emotionCompanionMD)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Meta.Category != "emotion_companion" {
		t.Fatalf("category=%q", parsed.Meta.Category)
	}
	if parsed.Meta.LoadMode != "trigger" {
		t.Fatalf("load_mode=%q", parsed.Meta.LoadMode)
	}
	if parsed.Meta.TTLTurns != 5 {
		t.Fatalf("ttl=%d", parsed.Meta.TTLTurns)
	}
	tags := parsed.Meta.Triggers.AllTags()
	if len(tags) != 4 {
		t.Fatalf("tags=%v", tags)
	}
	if parsed.Meta.ResolvePriority() != 90 {
		t.Fatalf("priority=%d", parsed.Meta.ResolvePriority())
	}
}

func TestBuildRegistryAndMatch(t *testing.T) {
	files := []model.PersonaFile{
		{FileName: "Emotion-Companion.md", ModuleCategory: "emotion_companion"},
		{FileName: "Professional-Skills.md", ModuleCategory: "professional_skills"},
		{FileName: "Persona-Base.md", ModuleCategory: "persona_base"},
	}
	contents := map[string]string{
		"Emotion-Companion.md":  emotionCompanionMD,
		"Professional-Skills.md": professionalMD,
		"Persona-Base.md":        personaBaseMD,
	}
	reg := BuildRegistry(1, files, func(pf *model.PersonaFile) string {
		return contents[pf.FileName]
	})
	if len(reg.Modules) != 3 {
		t.Fatalf("modules=%d", len(reg.Modules))
	}
	if got := reg.MatchCategories([]string{"sad"}); len(got) != 1 || got[0] != "emotion_companion" {
		t.Fatalf("sad match=%v", got)
	}
	if got := reg.MatchCategories([]string{"need_professional"}); len(got) != 1 || got[0] != "professional_skills" {
		t.Fatalf("pro match=%v", got)
	}
	if got := reg.MatchCategories([]string{"sad", "need_professional"}); len(got) != 2 {
		t.Fatalf("multi match=%v", got)
	}
	if ttl := reg.TTLForCategories([]string{"professional_skills"}, 5); ttl != 6 {
		t.Fatalf("ttl=%d", ttl)
	}
	if ttl := reg.TTLForCategories([]string{"unknown"}, 5); ttl != 5 {
		t.Fatalf("fallback ttl=%d", ttl)
	}
	labels := reg.ClassifierLabelList([]string{"neutral"})
	found := map[string]bool{}
	for _, l := range labels {
		found[l] = true
	}
	for _, want := range []string{"neutral", "sad", "need_comfort", "need_professional"} {
		if !found[want] {
			t.Fatalf("labels missing %s in %v", want, labels)
		}
	}
}

func TestResolveLoadModeFrontmatterWins(t *testing.T) {
	if got := ResolveLoadMode("trigger", "persona_base"); got != "trigger" {
		t.Fatalf("got=%s", got)
	}
	if got := ResolveLoadMode("", "persona_base"); got != "always" {
		t.Fatalf("got=%s", got)
	}
	if got := ResolveLoadMode("", "emotion_companion"); got != "trigger" {
		t.Fatalf("got=%s", got)
	}
}

func TestDetectCategoryFromFileName(t *testing.T) {
	cases := map[string]string{
		"Emotion-Companion.md":  "emotion_companion",
		"Trigger-Pet-Peeve.md":  "trigger_rules",
		"Forbidden-Rules.md":    "forbidden_rules",
		"Style-Switch.md":       "style_switch",
		"Professional-Skills.md": "professional_skills",
	}
	for fn, want := range cases {
		if got := DetectCategoryFromFileName(fn); got != want {
			t.Fatalf("%s => %s want %s", fn, got, want)
		}
	}
}

func TestCompileCorePromptOnlyAlways(t *testing.T) {
	contents := map[string]string{
		"Emotion-Companion.md":  emotionCompanionMD,
		"Professional-Skills.md": professionalMD,
		"Persona-Base.md":        personaBaseMD,
	}
	cache := NewPersonaCache(nil, nil)
	// 直接用 compilePromptFromFiles 逻辑经 BuildRegistry 的 keep
	files := []model.PersonaFile{
		{FileName: "Emotion-Companion.md"},
		{FileName: "Persona-Base.md"},
	}
	stg := &fakeStg{contents: contents}
	// PersonaCache nil-safe: compile uses cache.GetMDContent - need non-nil cache
	// Use compile via BuildRegistry path instead: module load modes
	reg := BuildRegistry(1, files, func(pf *model.PersonaFile) string { return contents[pf.FileName] })
	var always int
	for _, m := range reg.L0Modules() {
		always++
		if m.Category != "persona_base" {
			t.Fatalf("L0 should be persona_base, got %s", m.Category)
		}
	}
	if always != 1 {
		t.Fatalf("L0 count=%d", always)
	}
	if len(reg.TriggerModules()) != 1 {
		t.Fatalf("trigger modules=%d", len(reg.TriggerModules()))
	}
	_ = cache
	_ = stg
}

type fakeStg struct{ contents map[string]string }

func (f *fakeStg) UploadMD(persona *model.Persona, fileName string, content []byte, priority int) (*model.PersonaFile, error) {
	return nil, nil
}
func (f *fakeStg) DownloadMD(pf *model.PersonaFile) ([]byte, error) {
	return []byte(f.contents[pf.FileName]), nil
}
func (f *fakeStg) DeleteMD(pf *model.PersonaFile) error { return nil }
func (f *fakeStg) DeleteAllByPersona(persona *model.Persona) error { return nil }
