package service

import (
	"strings"
	"testing"

	"cuetiy-backend/model"
	"cuetiy-backend/skill"
)

func TestGenericL0Placeholder_NoPersonaHardcode(t *testing.T) {
	s := genericL0Placeholder()
	for _, bad := range []string{"女友", "宝宝", "御姐", "软萌", "RainYi", "Cuetiy"} {
		if strings.Contains(s, bad) {
			t.Fatalf("generic L0 must not hardcode %q", bad)
		}
	}
}

func TestIntimateModeFallback_NoPersonaHardcode(t *testing.T) {
	s := IntimateModeFallback()
	for _, bad := range []string{"女友", "宝宝", "御姐"} {
		if strings.Contains(s, bad) {
			t.Fatalf("intimate fallback must not hardcode %q", bad)
		}
	}
	if !strings.Contains(strings.ToLower(s), "adult") {
		t.Fatal("should mention adult boundary")
	}
}

func TestCapabilityIndex_EmptyNotBusinessSpecific(t *testing.T) {
	s := CapabilityIndex(nil)
	if strings.Contains(s, "御姐") || strings.Contains(s, "雷点") {
		t.Fatalf("empty index must not invent business skills: %s", s)
	}
}

func TestCapabilityIndexFromRegistry_UsesFrontmatter(t *testing.T) {
	md := `---
name: Coach
category: coach
load_mode: trigger
description: "学习教练技能"
---
# body
x
`
	files := []model.PersonaFile{{FileName: "Coach.md", ModuleCategory: "coach"}}
	reg := skill.BuildRegistry(1, files, func(pf *model.PersonaFile) string { return md })
	out := CapabilityIndexFromRegistry(reg)
	if !strings.Contains(out, "coach") {
		t.Fatalf("index should list registry category: %s", out)
	}
	if !strings.Contains(out, "学习教练") {
		t.Fatalf("index should use frontmatter description: %s", out)
	}
}

func TestModuleLoadMode_DelegatesToSkillPackage(t *testing.T) {
	if ModuleLoadMode("persona_base") != LoadAlways {
		t.Fatal("persona_base should be always via skill.ResolveLoadMode")
	}
	if ModuleLoadModeMeta("trigger", "persona_base") != LoadTrigger {
		t.Fatal("frontmatter must win over category convention")
	}
	if ModuleLoadMode("nsfw_companion") != LoadTrigger {
		t.Fatal("nsfw_companion should be trigger fallback")
	}
}
