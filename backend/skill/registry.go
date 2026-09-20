package skill

import (
	"path/filepath"
	"sort"
	"strings"

	"cuetiy-backend/model"
)

const (
	LoadModeAlways  = "always"
	LoadModeIndex   = "index"
	LoadModeTrigger = "trigger"
)

// SkillModule 注册表中的一个技能模块（由 frontmatter 自描述）
type SkillModule struct {
	Category    string        `json:"category"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	FileName    string        `json:"file_name"`
	LoadMode    string        `json:"load_mode"`
	Priority    int           `json:"priority"`
	TTLTurns    int           `json:"ttl_turns"`
	Triggers    SkillTriggers `json:"triggers"`
	Keywords    []string      `json:"keywords"`
	Examples    []string      `json:"examples"`
	// 出图：外貌 / 拍摄风格 / 覆盖 prompt
	Appearance  string `json:"appearance"`
	ImageStyle  string `json:"image_style"`
	ImagePrompt string `json:"image_prompt"`
}

// SkillRegistry 人格包内全部技能模块的路由视图
type SkillRegistry struct {
	PersonaID int64         `json:"persona_id"`
	Modules   []SkillModule `json:"modules"`
}

// ResolveLoadMode frontmatter 优先，其次按 category 旧约定回退
func ResolveLoadMode(metaLoadMode, category string) string {
	switch strings.ToLower(strings.TrimSpace(metaLoadMode)) {
	case LoadModeAlways, LoadModeIndex, LoadModeTrigger:
		return strings.ToLower(strings.TrimSpace(metaLoadMode))
	}
	switch category {
	case "persona_base", "persona_tone", "forbidden_rules":
		return LoadModeAlways
	case "emotion_companion", "style_switch", "professional_skills", "trigger_rules", "nsfw_companion":
		return LoadModeTrigger
	case "image_appearance", "visual_appearance":
		// 外貌/出图模块：chat 层可不注入全文，但 registry 始终可读 appearance/keywords
		return LoadModeTrigger
	default:
		return LoadModeIndex
	}
}

// DetectCategoryFromFileName 文件名启发式（frontmatter 未写 category 时的回退）
func DetectCategoryFromFileName(fileName string) string {
	name := strings.ToLower(strings.ReplaceAll(fileName, "_", "-"))
	base := strings.TrimSuffix(name, ".md")
	switch {
	case strings.Contains(base, "persona-base"), strings.Contains(base, "personabase"):
		return "persona_base"
	case strings.Contains(base, "persona-tone"), strings.Contains(base, "personatone"):
		return "persona_tone"
	case strings.Contains(base, "forbidden"):
		return "forbidden_rules"
	case strings.Contains(base, "emotion"), strings.Contains(base, "companion"):
		return "emotion_companion"
	case strings.Contains(base, "professional"):
		return "professional_skills"
	case strings.Contains(base, "style"):
		return "style_switch"
	case strings.Contains(base, "trigger"), strings.Contains(base, "pet"):
		return "trigger_rules"
	case strings.Contains(base, "image"), strings.Contains(base, "appearance"), strings.Contains(base, "visual"):
		return "image_appearance"
	default:
		return "general"
	}
}

func ResolveCategory(metaCategory, fileName string) string {
	c := strings.TrimSpace(metaCategory)
	if c != "" {
		return c
	}
	return DetectCategoryFromFileName(fileName)
}

// ModuleFromFile 从 persona 文件内容构建模块项
func ModuleFromFile(fileName, content string) SkillModule {
	parsed, err := ParseSkillContent(fileName, content)
	if err != nil || parsed == nil {
		cat := DetectCategoryFromFileName(fileName)
		return SkillModule{
			Category: cat,
			Name:     strings.TrimSuffix(filepath.Base(fileName), ".md"),
			FileName: fileName,
			LoadMode: ResolveLoadMode("", cat),
			Priority: 50,
			Triggers: SkillTriggers{},
		}
	}
	meta := parsed.Meta
	cat := ResolveCategory(meta.Category, fileName)
	lm := ResolveLoadMode(meta.LoadMode, cat)
	desc := strings.TrimSpace(meta.Description)
	if desc == "" && len(parsed.KVList) > 0 {
		desc = firstNonEmptyLine(parsed.KVList[0].Value)
	}
	kws := append([]string{}, meta.Triggers.Keywords...)
	return SkillModule{
		Category:    cat,
		Name:        meta.Name,
		Description: desc,
		FileName:    fileName,
		LoadMode:    lm,
		Priority:    meta.ResolvePriority(),
		TTLTurns:    meta.TTLTurns,
		Triggers:    meta.Triggers,
		Keywords:    kws,
		Examples:    meta.Examples,
		Appearance:  strings.TrimSpace(meta.Appearance),
		ImageStyle:  strings.TrimSpace(meta.ImageStyle),
		ImagePrompt: strings.TrimSpace(meta.ImagePrompt),
	}
}

// BuildRegistry 从人格文件列表构建注册表
func BuildRegistry(personaID int64, files []model.PersonaFile, contentOf func(*model.PersonaFile) string) *SkillRegistry {
	reg := &SkillRegistry{PersonaID: personaID}
	for i := range files {
		f := files[i]
		content := ""
		if contentOf != nil {
			content = contentOf(&f)
		}
		mod := ModuleFromFile(f.FileName, content)
		// DB 里已有 category 且 frontmatter 未写时，保留 DB 值
		if strings.TrimSpace(f.ModuleCategory) != "" && !frontmatterHasCategory(content) {
			mod.Category = f.ModuleCategory
			mod.LoadMode = ResolveLoadMode("", mod.Category)
		}
		reg.Modules = append(reg.Modules, mod)
	}
	sort.SliceStable(reg.Modules, func(i, j int) bool {
		if reg.Modules[i].Priority != reg.Modules[j].Priority {
			return reg.Modules[i].Priority > reg.Modules[j].Priority
		}
		return reg.Modules[i].Category < reg.Modules[j].Category
	})
	return reg
}

func frontmatterHasCategory(content string) bool {
	parsed, err := ParseSkillContent("x.md", content)
	if err != nil || parsed == nil {
		return false
	}
	return strings.TrimSpace(parsed.Meta.Category) != ""
}

// AllTriggerTags 注册表声明的全部可匹配标签（供分类器 prompt 使用）
func (r *SkillRegistry) AllTriggerTags() []string {
	if r == nil {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	for _, m := range r.Modules {
		for _, t := range m.Triggers.AllTags() {
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}
	sort.Strings(out)
	return out
}

// AllKeywords 技能包自带关键词
func (r *SkillRegistry) AllKeywords() []string {
	if r == nil {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	for _, m := range r.Modules {
		for _, k := range m.Keywords {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, k)
		}
	}
	return out
}

// L0Modules 常驻模块
func (r *SkillRegistry) L0Modules() []SkillModule {
	return r.modulesByLoadMode(LoadModeAlways)
}

// TriggerModules 可触发注入模块
func (r *SkillRegistry) TriggerModules() []SkillModule {
	return r.modulesByLoadMode(LoadModeTrigger)
}

func (r *SkillRegistry) modulesByLoadMode(mode string) []SkillModule {
	if r == nil {
		return nil
	}
	var out []SkillModule
	for _, m := range r.Modules {
		if m.LoadMode == mode {
			out = append(out, m)
		}
	}
	return out
}

// MatchCategories 标签集合 → 应激活的 category（去重，按优先级）
func (r *SkillRegistry) MatchCategories(tags []string) []string {
	if r == nil || len(tags) == 0 {
		return nil
	}
	tagSet := map[string]struct{}{}
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		tagSet[t] = struct{}{}
	}
	seen := map[string]struct{}{}
	var out []string
	for _, m := range r.Modules {
		if m.LoadMode != LoadModeTrigger {
			continue
		}
		if !moduleMatchesTags(m, tagSet) {
			continue
		}
		if _, ok := seen[m.Category]; ok {
			continue
		}
		seen[m.Category] = struct{}{}
		out = append(out, m.Category)
	}
	return out
}

func moduleMatchesTags(m SkillModule, tagSet map[string]struct{}) bool {
	for _, t := range m.Triggers.AllTags() {
		if _, ok := tagSet[strings.ToLower(strings.TrimSpace(t))]; ok {
			return true
		}
	}
	return false
}

// TTLForCategories 命中 category 的 TTL：取模块声明最大值，否则 fallback
func (r *SkillRegistry) TTLForCategories(cats []string, fallback int) int {
	if fallback <= 0 {
		fallback = 5
	}
	if r == nil || len(cats) == 0 {
		return fallback
	}
	want := map[string]struct{}{}
	for _, c := range cats {
		want[c] = struct{}{}
	}
	best := 0
	for _, m := range r.Modules {
		if _, ok := want[m.Category]; !ok {
			continue
		}
		if m.TTLTurns > best {
			best = m.TTLTurns
		}
	}
	if best <= 0 {
		return fallback
	}
	return best
}

// ClassifierLabelList 分类器可选标签：注册表标签 ∪ 基础情绪
func (r *SkillRegistry) ClassifierLabelList(base []string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(list []string) {
		for _, s := range list {
			s = strings.ToLower(strings.TrimSpace(s))
			if s == "" {
				continue
			}
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	add(base)
	add(r.AllTriggerTags())
	return out
}

// --- 出图 / 形象（由技能包描述，引擎不写死人设） ---

func isImageCategory(cat string) bool {
	switch strings.ToLower(strings.TrimSpace(cat)) {
	case "image_appearance", "visual_appearance", "image":
		return true
	default:
		return false
	}
}

func (r *SkillRegistry) imageModules() []SkillModule {
	if r == nil {
		return nil
	}
	var out []SkillModule
	for _, m := range r.Modules {
		if isImageCategory(m.Category) ||
			strings.TrimSpace(m.Appearance) != "" ||
			strings.TrimSpace(m.ImageStyle) != "" ||
			strings.TrimSpace(m.ImagePrompt) != "" {
			out = append(out, m)
		}
	}
	return out
}

// ImageTriggerKeywords 技能包声明的「想看图」触发词
func (r *SkillRegistry) ImageTriggerKeywords() []string {
	if r == nil {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, m := range r.imageModules() {
		for _, k := range m.Keywords {
			add(k)
		}
		// intents/tags 含 want_image / image 时，模块 keywords 仍生效；也可在 tags 里放触发词
		for _, t := range m.Triggers.Tags {
			// tags 主要是语义标签；仅当看起来像中文触发短语时也纳入
			if strings.ContainsAny(t, "看看") || strings.Contains(t, "照片") || strings.Contains(t, "自拍") {
				add(t)
			}
		}
		for _, e := range m.Examples {
			// examples 过短且像请求时可作为弱触发（可选）；默认不把 examples 全当触发
			_ = e
		}
	}
	return out
}

// ShouldTriggerImage 是否由技能包触发出图；无包声明时返回 false（引擎不硬编码业务词）
func (r *SkillRegistry) ShouldTriggerImage(userMsg string) bool {
	kws := r.ImageTriggerKeywords()
	if len(kws) == 0 {
		return false
	}
	lower := strings.ToLower(userMsg)
	for _, k := range kws {
		if k == "" {
			continue
		}
		if strings.Contains(userMsg, k) || strings.Contains(lower, strings.ToLower(k)) {
			return true
		}
	}
	// 语义标签：分类结果若含 want_image / image_ask 也可由调用方传入 tags
	return false
}

// ShouldTriggerImageTags 分类标签命中 want_image / image_ask 等
func (r *SkillRegistry) ShouldTriggerImageTags(tags []string) bool {
	if r == nil {
		return false
	}
	set := map[string]struct{}{}
	for _, t := range tags {
		set[strings.ToLower(strings.TrimSpace(t))] = struct{}{}
	}
	want := []string{"want_image", "image_ask", "want_photo", "want_selfie", "image"}
	for _, m := range r.imageModules() {
		for _, t := range m.Triggers.Intents {
			want = append(want, strings.ToLower(strings.TrimSpace(t)))
		}
		for _, t := range m.Triggers.Tags {
			want = append(want, strings.ToLower(strings.TrimSpace(t)))
		}
	}
	for _, w := range want {
		if w == "" {
			continue
		}
		if _, ok := set[w]; ok {
			return true
		}
	}
	return false
}

// ImageAppearanceFromSkills 外貌描述（用于 TTI prompt）
func (r *SkillRegistry) ImageAppearanceFromSkills() string {
	if r == nil {
		return ""
	}
	var parts []string
	seen := map[string]struct{}{}
	for _, m := range r.imageModules() {
		a := strings.TrimSpace(m.Appearance)
		if a == "" {
			continue
		}
		if _, ok := seen[a]; ok {
			continue
		}
		seen[a] = struct{}{}
		parts = append(parts, a)
	}
	return strings.Join(parts, "；")
}

// ImageStyleFromSkills 拍摄/画面风格
func (r *SkillRegistry) ImageStyleFromSkills() string {
	if r == nil {
		return ""
	}
	var parts []string
	seen := map[string]struct{}{}
	for _, m := range r.imageModules() {
		s := strings.TrimSpace(m.ImageStyle)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		parts = append(parts, s)
	}
	return strings.Join(parts, "；")
}

// ImagePromptOverride 技能包整段覆盖 prompt（优先级最高）
func (r *SkillRegistry) ImagePromptOverride() string {
	if r == nil {
		return ""
	}
	// 按 priority 降序，取第一个写了 image_prompt 的
	mods := r.imageModules()
	sort.SliceStable(mods, func(i, j int) bool { return mods[i].Priority > mods[j].Priority })
	for _, m := range mods {
		if p := strings.TrimSpace(m.ImagePrompt); p != "" {
			return p
		}
	}
	return ""
}

func firstNonEmptyLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			if r := []rune(line); len(r) > 80 {
				return string(r[:80]) + "…"
			}
			return line
		}
	}
	return ""
}
