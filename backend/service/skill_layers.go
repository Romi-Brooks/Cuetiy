package service

import (
	"fmt"
	"strings"

	"cuetiy-backend/skill"
)

// SkillLoadMode 技能加载层级（与 skill 包 frontmatter 语义一致）
type SkillLoadMode string

const (
	LoadAlways  SkillLoadMode = skill.LoadModeAlways
	LoadIndex   SkillLoadMode = skill.LoadModeIndex
	LoadTrigger SkillLoadMode = skill.LoadModeTrigger
)

// ModuleLoadMode 加载策略：frontmatter.load_mode 优先；
// 未写时按 category 约定回退（仅作兼容，不承载具体人设文案）。
func ModuleLoadMode(category string) SkillLoadMode {
	return SkillLoadMode(skill.ResolveLoadMode("", category))
}

// ModuleLoadModeMeta frontmatter 优先的加载策略
func ModuleLoadModeMeta(metaLoadMode, category string) SkillLoadMode {
	return SkillLoadMode(skill.ResolveLoadMode(metaLoadMode, category))
}

// IsCoreL0 是否核心常驻模块
func IsCoreL0(category string) bool {
	return ModuleLoadMode(category) == LoadAlways
}

// CompactL0Prompt 压缩 L0 全文；技能包为空时回落中性占位（不含具体人设）。
func CompactL0Prompt(full string) string {
	full = strings.TrimSpace(full)
	if full == "" {
		return genericL0Placeholder()
	}
	// 全文已经较短则原样
	if EstimateTokens(full) <= 900 {
		return full
	}
	// 粗压缩：去掉示例行与多余空行，保留规则行
	var b strings.Builder
	for _, line := range strings.Split(full, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		low := strings.ToLower(t)
		if strings.Contains(low, "示例") || strings.HasPrefix(t, "✅") || strings.HasPrefix(t, "❌") {
			continue
		}
		b.WriteString(t)
		b.WriteString("\n")
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return genericL0Placeholder()
	}
	return out
}

// genericL0Placeholder 无人格技能包时的中性 L0。
// 真正人设/语气/禁止项必须来自技能 MD（load_mode=always），禁止再写死业务文案。
func genericL0Placeholder() string {
	return `# Session
This companion session is defined by persona skill modules when loaded.
If skill modules are present, follow them as the primary source of character, tone, and rules.
If not, reply naturally and briefly in the user's language; stay in-context; do not dump structured lectures.`
}

// IntimateModeFallback 会话开启亲密模式、但技能包未提供 nsfw 模块时的通用说明。
// 不假设具体人设；深度内容应写在技能包的 nsfw 模块里。
func IntimateModeFallback() string {
	return `# Intimate mode (conversation flag: ON)
- If persona skills define intimate behavior, follow those skills first.
- Adult, consensual, companion-oriented interaction only.
- Match the persona voice and short natural pacing; prioritize presence over monologues.
- Stop immediately if the user declines or shifts topic.
- Refuse illegal, non-consensual, or minor-related content; do not invent real-world harm advice.
- When this flag is OFF, do not steer toward intimate topics on your own.`
}

// CapabilityIndex 生成 L1 能力索引。
// modules: category → 描述；空集合时不编造业务能力清单。
func CapabilityIndex(modules map[string]string) string {
	if len(modules) == 0 {
		return "# Optional skills\n(none registered for this persona)"
	}
	var b strings.Builder
	b.WriteString("# Optional skills (activate on demand; full text only when triggered)\n")
	for cat, desc := range modules {
		if IsCoreL0(cat) {
			continue
		}
		b.WriteString(fmt.Sprintf("- %s：%s\n", cat, firstLine(desc)))
	}
	return strings.TrimSpace(b.String())
}

// CapabilityIndexFromRegistry 从技能注册表生成 L1（推荐路径：load_mode 来自 frontmatter）。
func CapabilityIndexFromRegistry(reg *skill.SkillRegistry) string {
	if reg == nil || len(reg.Modules) == 0 {
		return CapabilityIndex(nil)
	}
	var b strings.Builder
	b.WriteString("# Optional skills (activate on demand; full text only when triggered)\n")
	for _, m := range reg.Modules {
		if m.LoadMode == skill.LoadModeAlways {
			continue
		}
		desc := strings.TrimSpace(m.Description)
		if desc == "" {
			desc = m.Name
		}
		if desc == "" {
			desc = m.Category
		}
		cat := m.Category
		if cat == "" {
			cat = m.FileName
		}
		b.WriteString(fmt.Sprintf("- %s：%s\n", cat, firstLine(desc)))
	}
	return strings.TrimSpace(b.String())
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\n"); i != -1 {
		s = s[:i]
	}
	return TruncateRunes(s, 80)
}
