package service

import (
	"testing"

	"cuetiy-backend/model"
	"cuetiy-backend/skill"
)

func TestRuleTags_GenericEmotion(t *testing.T) {
	r := NewSkillRouter(nil)
	res := r.ruleTags("今天好难过啊", nil)
	if res.Label != EmotionSad {
		t.Fatalf("label=%s", res.Label)
	}
	if res.Confidence < 0.6 {
		t.Fatalf("conf=%v", res.Confidence)
	}
	found := false
	for _, tag := range res.Tags {
		if tag == "sad" {
			found = true
		}
	}
	if !found {
		t.Fatalf("tags=%v", res.Tags)
	}
}

func TestRoute_RegistryDrivesCategories(t *testing.T) {
	md := `---
name: Study-Coach
category: study_coach
load_mode: trigger
triggers:
  intents: [study_plan]
  keywords: [刷题]
---
# coach
study
`
	files := []model.PersonaFile{{FileName: "Study-Coach.md", ModuleCategory: "study_coach"}}
	reg := skill.BuildRegistry(1, files, func(pf *model.PersonaFile) string { return md })

	r := NewSkillRouter(nil)
	// 关键词命中技能包 tags
	res := r.Route("帮我刷题规划一下", "", reg)
	if len(res.Categories) == 0 || res.Categories[0] != "study_coach" {
		// rule may tag study_plan via pkg keywords
		// If generic need_professional also matches 规划? not in lexicon
		t.Fatalf("cats=%v tags=%v label=%s reason=%s", res.Categories, res.Tags, res.Label, res.Reason)
	}

	// 情绪路径：sad → emotion 模块（YiSkill 风格注册表）
	yi := `---
name: Emotion-Companion
category: emotion_companion
load_mode: trigger
triggers:
  emotions: [joy, sad, anxious, angry, tired, flirty]
  intents: [need_comfort]
---
# e
x
`
	yiFiles := []model.PersonaFile{{FileName: "Emotion-Companion.md"}}
	yiReg := skill.BuildRegistry(2, yiFiles, func(pf *model.PersonaFile) string { return yi })
	res2 := r.Route("我好难过", "", yiReg)
	if len(res2.Categories) != 1 || res2.Categories[0] != "emotion_companion" {
		t.Fatalf("cats=%v tags=%v", res2.Categories, res2.Tags)
	}
}

func TestCategoriesFor_LegacyFallback(t *testing.T) {
	cats := CategoriesForEmotion(EmotionResult{Label: EmotionSad})
	if len(cats) != 1 || cats[0] != "emotion_companion" {
		t.Fatalf("%v", cats)
	}
	cats = CategoriesForEmotion(EmotionResult{Label: EmotionStyleAsk})
	if len(cats) != 1 || cats[0] != "style_switch" {
		t.Fatalf("%v", cats)
	}
}

func TestRoute_NoRegistryLegacy(t *testing.T) {
	r := NewSkillRouter(nil)
	res := r.Route("切换成御姐模式", "", nil)
	if res.Label != EmotionStyleAsk {
		t.Fatalf("label=%s tags=%v", res.Label, res.Tags)
	}
	if len(res.Categories) != 1 || res.Categories[0] != "style_switch" {
		t.Fatalf("cats=%v", res.Categories)
	}
}

func TestRuleTags_PackageKeywords(t *testing.T) {
	md := `---
name: Cooking
category: cooking_skill
load_mode: trigger
triggers:
  intents: [need_recipe]
  keywords: [红烧肉]
---
# cook
x
`
	files := []model.PersonaFile{{FileName: "Cooking.md"}}
	reg := skill.BuildRegistry(1, files, func(pf *model.PersonaFile) string { return md })
	r := NewSkillRouter(nil)
	res := r.Route("晚上想吃红烧肉怎么做", "", reg)
	if len(res.Categories) == 0 || res.Categories[0] != "cooking_skill" {
		t.Fatalf("cats=%v tags=%v reason=%s", res.Categories, res.Tags, res.Reason)
	}
}
