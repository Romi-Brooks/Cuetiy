package skill

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"cuetiy-backend/config"
	"cuetiy-backend/model"
	"cuetiy-backend/repository"

	"gopkg.in/yaml.v3"
)

type MDFileStorage interface {
	UploadMD(persona *model.Persona, fileName string, content []byte, priority int) (*model.PersonaFile, error)
	DownloadMD(pf *model.PersonaFile) ([]byte, error)
	DeleteMD(pf *model.PersonaFile) error
	DeleteAllByPersona(persona *model.Persona) error
}

// SkillTriggers 技能自声明的触发条件（通用协议核心，路由器不写死业务词表）
type SkillTriggers struct {
	Emotions []string `yaml:"emotions"`
	Intents  []string `yaml:"intents"`
	Domains  []string `yaml:"domains"`
	Keywords []string `yaml:"keywords"`
	Tags     []string `yaml:"tags"`
}

func (t SkillTriggers) AllTags() []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(list []string) {
		for _, s := range list {
			s = strings.TrimSpace(s)
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
	add(t.Emotions)
	add(t.Intents)
	add(t.Domains)
	add(t.Tags)
	return out
}

type SkillMeta struct {
	Name         string        `yaml:"name"`
	Description  string        `yaml:"description"`
	AllowedTools string        `yaml:"allowed-tools"`
	Priority     string        `yaml:"priority"`    // 兼容旧中文描述
	PriorityNum  int           `yaml:"priority_num"` // 0-100，越大越优先（同标签命中时）
	Category     string        `yaml:"category"`
	LoadMode     string        `yaml:"load_mode"`  // always | index | trigger
	TTLTurns     int           `yaml:"ttl_turns"`  // L2 激活轮数，0 表示用全局默认
	Triggers     SkillTriggers `yaml:"triggers"`
	Examples     []string      `yaml:"examples"`
	// 图片/形象（出图遵循技能包，不写死在引擎）
	// appearance：长什么样（五官/发型/气质…）
	// image_style：拍摄风格（自拍/光线/场景…）
	// image_prompt：整段覆盖式 prompt（写了则优先）
	Appearance  string `yaml:"appearance"`
	ImageStyle  string `yaml:"image_style"`
	ImagePrompt string `yaml:"image_prompt"`
}

// ResolvePriority 数值优先；未写 priority_num 时回退旧文案启发式
func (m *SkillMeta) ResolvePriority() int {
	if m.PriorityNum > 0 {
		return m.PriorityNum
	}
	switch {
	case m.Priority == "":
		return 50
	case strings.Contains(m.Priority, "最高"), strings.Contains(m.Priority, "high"), strings.Contains(m.Priority, "urgent"):
		return 90
	case strings.Contains(m.Priority, "高"):
		return 80
	case strings.Contains(m.Priority, "中"), strings.Contains(m.Priority, "medium"):
		return 50
	case strings.Contains(m.Priority, "低"), strings.Contains(m.Priority, "low"):
		return 20
	default:
		return 50
	}
}

func (m *SkillMeta) NumericPriority() int {
	// DB priority 字段：数字越小越靠前（沿用旧语义）
	p := m.ResolvePriority()
	return 100 - p
}

type ParsedSkill struct {
	Meta     SkillMeta
	FileName string
	KVList   []ParsedKV
}

type ParsedKV struct {
	Key   string
	Value string
}

type SkillManager struct {
	mu            sync.RWMutex
	repo          *repository.PersonaRepository
	promptCache   *PromptCache
	personaCache  *PersonaCache
	personaStg    MDFileStorage
	regMu         sync.RWMutex
	registryCache map[int64]*SkillRegistry
}

func NewSkillManager(repo *repository.PersonaRepository, pfRepo *repository.PersonaFileRepository, personaStg MDFileStorage) *SkillManager {
	personaCache := NewPersonaCache(repo, pfRepo)
	return &SkillManager{
		repo:          repo,
		promptCache:   NewPromptCache(personaCache, personaStg),
		personaCache:  personaCache,
		personaStg:    personaStg,
		registryCache: make(map[int64]*SkillRegistry),
	}
}

// GetSkillRegistry 获取人格技能注册表（frontmatter 驱动，带缓存）
func (m *SkillManager) GetSkillRegistry(personaID int64) *SkillRegistry {
	if m == nil || personaID == 0 {
		return nil
	}
	m.regMu.RLock()
	if reg, ok := m.registryCache[personaID]; ok {
		m.regMu.RUnlock()
		return reg
	}
	m.regMu.RUnlock()

	if m.personaCache == nil {
		return nil
	}
	files, err := m.personaCache.GetFileIndex(personaID)
	if err != nil || len(files) == 0 {
		return nil
	}
	reg := BuildRegistry(personaID, files, func(pf *model.PersonaFile) string {
		content, cerr := m.personaCache.GetMDContent(personaID, pf)
		if cerr != nil || content == "" {
			if m.personaStg != nil {
				if data, derr := m.personaStg.DownloadMD(pf); derr == nil {
					content = string(data)
					m.personaCache.SetMDContent(personaID, pf, content)
				}
			}
		}
		return content
	})
	m.regMu.Lock()
	m.registryCache[personaID] = reg
	m.regMu.Unlock()
	return reg
}

// InvalidateRegistry 人格文件变更后清除注册表缓存
func (m *SkillManager) InvalidateRegistry(personaID int64) {
	if m == nil {
		return
	}
	m.regMu.Lock()
	if personaID == 0 {
		m.registryCache = make(map[int64]*SkillRegistry)
	} else {
		delete(m.registryCache, personaID)
	}
	m.regMu.Unlock()
}

func (m *SkillManager) LoadSkills() error {
	personas, err := m.repo.FindActive()
	if err != nil {
		return fmt.Errorf("failed to load active personas: %w", err)
	}

	m.personaCache.WarmupList(personas)

	if err := m.seedFromLocalDir(); err != nil {
		log.Printf("警告: skills 目录自动检测失败: %v", err)
	}

	return nil
}

func (m *SkillManager) seedFromLocalDir() error {
	skillsDir := config.AppConfig.SkillsDir
	if skillsDir == "" {
		skillsDir = "./skills"
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return fmt.Errorf("读取 skills 目录失败: %w", err)
	}

	seeded := false
	for _, entry := range entries {
		if entry.IsDir() {
			if err := m.seedPersonaDir(filepath.Join(skillsDir, entry.Name()), entry.Name()); err != nil {
				log.Printf("跳过人格目录 %s: %v", entry.Name(), err)
				continue
			}
			seeded = true
		}
	}

	rootMDFiles := findRootMDFiles(entries)
	for _, fn := range rootMDFiles {
		if err := m.seedDefaultPersona(filepath.Join(skillsDir, fn)); err != nil {
			log.Printf("跳过默认人格文件 %s: %v", fn, err)
			continue
		}
		seeded = true
	}

	if seeded && m.personaCache != nil {
		personas, _ := m.repo.FindActive()
		m.personaCache.WarmupList(personas)
	}

	m.InvalidateRegistry(0)

	return nil
}

func findRootMDFiles(entries []os.DirEntry) []string {
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			files = append(files, e.Name())
		}
	}
	return files
}

func (m *SkillManager) seedPersonaDir(dirPath, dirName string) error {
	if m.personaStg == nil {
		return nil
	}

	existing, _ := m.repo.FindByDirName(dirName)
	if existing != nil {
		return nil
	}

	mdEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	var mdFiles []string
	for _, e := range mdEntries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			mdFiles = append(mdFiles, e.Name())
		}
	}
	if len(mdFiles) == 0 {
		return fmt.Errorf("no md files found")
	}

	persona := &model.Persona{
		UserID:      0,
		Name:        dirName,
		Description: fmt.Sprintf("从 %s 目录自动加载 (%d 个文件)", dirName, len(mdFiles)),
		DirName:     dirName,
		IsActive:    true,
	}
	if err := m.repo.Create(persona); err != nil {
		return fmt.Errorf("创建人格失败: %w", err)
	}

	for _, fn := range mdFiles {
		content, err := os.ReadFile(filepath.Join(dirPath, fn))
		if err != nil {
			continue
		}
		parsed, _ := ParseSkillContent(fn, string(content))
		priority := 5
		if parsed != nil {
			priority = parsed.Meta.NumericPriority()
		}
		if _, err := m.personaStg.UploadMD(persona, fn, content, priority); err != nil {
			log.Printf("上传 %s 失败: %v", fn, err)
		}
	}

	log.Printf("自动加载人格: %s (%s, %d 个文件)", dirName, persona.DirName, len(mdFiles))
	return nil
}

func (m *SkillManager) seedDefaultPersona(filePath string) error {
	if m.personaStg == nil {
		return nil
	}

	existing, _ := m.repo.FindByDirName("default")
	if existing != nil {
		return nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	fileName := filepath.Base(filePath)

	persona := &model.Persona{
		UserID:      0,
		Name:        "默认人格",
		Description: "系统内置默认人格，基于 SKILL-DEFAULT.md",
		DirName:     "default",
		IsActive:    true,
	}
	if err := m.repo.Create(persona); err != nil {
		return fmt.Errorf("创建默认人格失败: %w", err)
	}

	parsed, _ := ParseSkillContent(fileName, string(content))
	priority := 0
	if parsed != nil {
		priority = parsed.Meta.NumericPriority()
	}

	if _, err := m.personaStg.UploadMD(persona, fileName, content, priority); err != nil {
		return fmt.Errorf("上传默认人格 MD 失败: %w", err)
	}

	log.Printf("自动加载默认人格: %s (dir_name=default)", fileName)
	return nil
}

func (m *SkillManager) GetSystemPromptByPersona(personaID *int64) string {
	if personaID == nil {
		return "你是一个温柔、耐心、治愈的情感陪伴助手，像一个温暖的朋友。"
	}

	if cached, ok := m.promptCache.Get(*personaID); ok {
		return cached
	}

	prompt := m.promptCache.CompileAndCache(*personaID)
	if prompt == "" {
		return "你是一个温柔、耐心、治愈的情感陪伴助手，像一个温暖的朋友。"
	}

	return prompt
}

// GetCompiledByCategory 按 module_category 取单个技能模块全文（L2 注入）
func (m *SkillManager) GetCompiledByCategory(personaID int64, category string) string {
	if personaID == 0 || category == "" || m.personaCache == nil || m.personaStg == nil {
		return ""
	}
	files, err := m.personaCache.GetFileIndex(personaID)
	if err != nil {
		return ""
	}
	var picked []model.PersonaFile
	for _, f := range files {
		if f.ModuleCategory == category {
			picked = append(picked, f)
		}
	}
	if len(picked) == 0 {
		return ""
	}
	return CompilePromptFromFiles(picked, m.personaStg, m.personaCache, personaID)
}

// ListModuleSummaries 模块 category → 首行描述（L1 索引）
func (m *SkillManager) ListModuleSummaries(personaID int64) map[string]string {
	out := map[string]string{}
	if personaID == 0 || m.personaCache == nil || m.personaStg == nil {
		return out
	}
	files, err := m.personaCache.GetFileIndex(personaID)
	if err != nil {
		return out
	}
	for _, f := range files {
		cat := f.ModuleCategory
		if cat == "" {
			cat = "general"
		}
		if _, ok := out[cat]; ok {
			continue
		}
		content, cerr := m.personaCache.GetMDContent(personaID, &f)
		if cerr != nil || content == "" {
			if data, derr := m.personaStg.DownloadMD(&f); derr == nil {
				content = string(data)
				m.personaCache.SetMDContent(personaID, &f, content)
			}
		}
		if content == "" {
			out[cat] = f.FileName
			continue
		}
		if parsed, perr := ParseSkillContent(f.FileName, content); perr == nil && len(parsed.KVList) > 0 {
			out[cat] = parsed.KVList[0].Key + "：" + truncateRunes(parsed.KVList[0].Value, 60)
		} else {
			out[cat] = truncateRunes(strings.TrimSpace(content), 60)
		}
	}
	return out
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func (m *SkillManager) GetPersonaNames() ([]string, error) {
	return m.personaCache.GetPersonaNames()
}

func (m *SkillManager) PersonaCache() *PersonaCache {
	return m.personaCache
}

func (m *SkillManager) PromptCache() *PromptCache {
	return m.promptCache
}

func (m *SkillManager) PersonaStorage() MDFileStorage {
	return m.personaStg
}

func (m *SkillManager) Refresh() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.repo.HardDelete(0); err != nil {
		return fmt.Errorf("failed to clear built-in personas: %w", err)
	}

	m.personaCache.InvalidateAll()
	m.InvalidateRegistry(0)

	return nil
}

func ParseSkillContent(fileName string, content string) (*ParsedSkill, error) {
	lines := strings.Split(content, "\n")

	var meta SkillMeta
	var bodyLines []string

	if len(lines) >= 2 && strings.TrimSpace(lines[0]) == "---" {
		endIndex := -1
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				endIndex = i
				break
			}
		}

		if endIndex != -1 {
			metaYaml := strings.Join(lines[1:endIndex], "\n")
			if err := yaml.Unmarshal([]byte(metaYaml), &meta); err == nil {
				bodyLines = lines[endIndex+1:]
			} else {
				bodyLines = lines
			}
		} else {
			bodyLines = lines
		}
	} else {
		bodyLines = lines
	}

	if meta.Name == "" {
		meta.Name = strings.TrimSuffix(fileName, ".md")
	}

	kvList := parseKVFromBody(bodyLines)

	return &ParsedSkill{
		Meta:     meta,
		FileName: fileName,
		KVList:   kvList,
	}, nil
}

func parseKVFromBody(lines []string) []ParsedKV {
	var kvs []ParsedKV
	var currentKey string
	var currentValueLines []string

	flush := func() {
		if currentKey != "" {
			value := strings.TrimSpace(strings.Join(currentValueLines, "\n"))
			if value != "" {
				kvs = append(kvs, ParsedKV{Key: currentKey, Value: value})
			}
		}
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "#") && (len(line) == 1 || line[1] == ' ') {
			flush()
			currentKey = strings.TrimSpace(line[1:])
			currentValueLines = nil
		} else if currentKey != "" {
			currentValueLines = append(currentValueLines, line)
		}
	}
	flush()

	if len(kvs) == 0 {
		content := strings.TrimSpace(strings.Join(lines, "\n"))
		if content != "" {
			kvs = append(kvs, ParsedKV{Key: "content", Value: content})
		}
	}

	return kvs
}

func CompilePromptFromFiles(files []model.PersonaFile, personaStg MDFileStorage, personaCache *PersonaCache, personaID int64) string {
	return compilePromptFromFiles(files, personaStg, personaCache, personaID, nil)
}

// CompileCorePromptFromFiles 仅编译 load_mode=always 的核心模块（L0）
func CompileCorePromptFromFiles(files []model.PersonaFile, personaStg MDFileStorage, personaCache *PersonaCache, personaID int64) string {
	return compilePromptFromFiles(files, personaStg, personaCache, personaID, func(meta SkillMeta, category string) bool {
		return ResolveLoadMode(meta.LoadMode, category) == LoadModeAlways
	})
}

func compilePromptFromFiles(
	files []model.PersonaFile,
	personaStg MDFileStorage,
	personaCache *PersonaCache,
	personaID int64,
	keep func(meta SkillMeta, category string) bool,
) string {
	var builder strings.Builder

	for _, f := range files {
		mdContent, err := personaCache.GetMDContent(personaID, &f)
		if err != nil || mdContent == "" {
			if personaStg != nil {
				data, dlErr := personaStg.DownloadMD(&f)
				if dlErr == nil {
					mdContent = string(data)
					personaCache.SetMDContent(personaID, &f, mdContent)
				}
			}
		}

		if mdContent == "" {
			continue
		}

		parsed, parseErr := ParseSkillContent(f.FileName, mdContent)
		if parseErr != nil {
			if keep != nil && !keep(SkillMeta{}, f.ModuleCategory) {
				continue
			}
			builder.WriteString(mdContent)
			builder.WriteString("\n\n")
			continue
		}

		if keep != nil {
			cat := ResolveCategory(parsed.Meta.Category, f.FileName)
			if !keep(parsed.Meta, cat) {
				continue
			}
		}

		for _, kv := range parsed.KVList {
			builder.WriteString(fmt.Sprintf("# %s\n%s\n\n", kv.Key, kv.Value))
		}
	}

	return strings.TrimSpace(builder.String())
}

func (m *SkillManager) LoadMDFromLocal(persona *model.Persona) error {
	if m.personaStg == nil {
		return fmt.Errorf("persona storage not available")
	}

	skillsDir := config.AppConfig.SkillsDir
	if skillsDir == "" {
		skillsDir = "./skills"
	}

	dirPath := filepath.Join(skillsDir, persona.DirName)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read skills dir %s: %w", dirPath, err)
	}

	for i, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dirPath, entry.Name()))
		if err != nil {
			continue
		}

		_, err = m.personaStg.UploadMD(persona, entry.Name(), content, i)
		if err != nil {
			continue
		}
	}

	return nil
}

func (m *SkillManager) Repo() *repository.PersonaRepository {
	return m.repo
}
