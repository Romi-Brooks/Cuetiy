package service

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"cuetiy-backend/config"
	"cuetiy-backend/model"
	"cuetiy-backend/repository"
	"cuetiy-backend/skill"

	"github.com/google/uuid"
)

const PersonaPrefix = "ai-persona"

type PersonaStorage struct {
	root   string
	pfRepo *repository.PersonaFileRepository
}

func NewPersonaStorage(pfRepo *repository.PersonaFileRepository) *PersonaStorage {
	root := "./data/files"
	if config.AppConfig != nil && config.AppConfig.StorageDir != "" {
		root = config.AppConfig.StorageDir
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	if err := os.MkdirAll(filepath.Join(abs, PersonaPrefix), 0o755); err != nil {
		log.Printf("警告: 创建人格存储目录失败: %v", err)
		return nil
	}
	log.Printf("PersonaStorage 已就绪: %s (前缀: %s)", abs, PersonaPrefix)
	return &PersonaStorage{root: abs, pfRepo: pfRepo}
}

func (ps *PersonaStorage) absPath(rel string) string {
	return filepath.Join(ps.root, filepath.FromSlash(rel))
}

func (ps *PersonaStorage) personaDir(persona *model.Persona) string {
	dirName := persona.DirName
	if dirName == "" {
		dirName = SanitizeDirName(persona.Name)
	}
	return fmt.Sprintf("%s/%s/%d", PersonaPrefix, dirName, persona.ID)
}

func (ps *PersonaStorage) objectPath(persona *model.Persona, fileName string) string {
	return fmt.Sprintf("%s/%s", ps.personaDir(persona), fileName)
}

func (ps *PersonaStorage) write(rel string, content []byte) error {
	full := ps.absPath(rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	return os.WriteFile(full, content, 0o644)
}

func (ps *PersonaStorage) UploadMD(persona *model.Persona, fileName string, content []byte, priority int) (*model.PersonaFile, error) {
	objectName := ps.objectPath(persona, fileName)
	if err := ps.write(objectName, content); err != nil {
		return nil, fmt.Errorf("写入 MD 文件失败: %w", err)
	}

	// frontmatter.category 优先，文件名启发式仅作回退
	category := detectModuleCategory(fileName)
	if parsed, err := skill.ParseSkillContent(fileName, string(content)); err == nil && parsed != nil {
		if c := strings.TrimSpace(parsed.Meta.Category); c != "" {
			category = c
		}
		if priority == 0 || parsed.Meta.PriorityNum > 0 || parsed.Meta.Priority != "" {
			// Upload 传入的 priority 为兼容值；frontmatter 明确时覆盖
			if parsed.Meta.PriorityNum > 0 || parsed.Meta.Priority != "" {
				priority = parsed.Meta.NumericPriority()
			}
		}
	}
	pf := &model.PersonaFile{
		PersonaID:      persona.ID,
		FileName:       fileName,
		StoragePath:    objectName,
		Priority:       priority,
		ModuleCategory: category,
		FileSize:       int64(len(content)),
	}
	if err := ps.pfRepo.Create(pf); err != nil {
		return nil, fmt.Errorf("PersonaFile 入库失败: %w", err)
	}
	return pf, nil
}

func (ps *PersonaStorage) DownloadMD(pf *model.PersonaFile) ([]byte, error) {
	data, err := os.ReadFile(ps.absPath(pf.StoragePath))
	if err != nil {
		return nil, fmt.Errorf("读取 MD 文件失败: %w", err)
	}
	return data, nil
}

func (ps *PersonaStorage) UploadAvatar(personaID int64, fileName string, data []byte) (string, error) {
	ext := filepath.Ext(fileName)
	if ext == "" {
		ext = ".jpg"
	}
	objectName := fmt.Sprintf("avatar/persona/%d/%s%s", personaID, uuid.New().String(), ext)
	if err := ps.write(objectName, data); err != nil {
		return "", fmt.Errorf("写入人格头像失败: %w", err)
	}
	return "/storage/" + objectName, nil
}

// UploadPersonaBackground 人格聊天页背景图，落盘路径与返回 URL 必须一致
func (ps *PersonaStorage) UploadPersonaBackground(personaID int64, fileName string, data []byte) (string, error) {
	ext := filepath.Ext(fileName)
	if ext == "" {
		ext = ".jpg"
	}
	objectName := fmt.Sprintf("persona-bg/%d/%s%s", personaID, uuid.New().String(), ext)
	if err := ps.write(objectName, data); err != nil {
		return "", fmt.Errorf("写入人格背景失败: %w", err)
	}
	return "/storage/" + objectName, nil
}

func (ps *PersonaStorage) DeleteMD(pf *model.PersonaFile) error {
	full := ps.absPath(pf.StoragePath)
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除 MD 文件失败: %w", err)
	}
	return ps.pfRepo.Delete(pf.ID)
}

func (ps *PersonaStorage) DeleteAllByPersona(persona *model.Persona) error {
	files, err := ps.pfRepo.FindByPersonaID(persona.ID)
	if err != nil {
		return err
	}
	for _, pf := range files {
		os.Remove(ps.absPath(pf.StoragePath))
	}
	return ps.pfRepo.DeleteByPersonaID(persona.ID)
}

func (ps *PersonaStorage) ListPersonaDirs() ([]string, error) {
	base := ps.absPath(PersonaPrefix)
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	return dirs, nil
}

func detectModuleCategory(fileName string) string {
	return skill.DetectCategoryFromFileName(fileName)
}

func SanitizeDirName(name string) string {
	sanitized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, name)
	if sanitized == "" {
		sanitized = "persona"
	}
	return strings.ToLower(sanitized)
}
