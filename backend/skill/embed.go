package skill

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

//go:embed default_skills
var defaultSkills embed.FS

// EnsureSkillsDir 保证目标 skills 目录存在。
// 目录不存在（或为空）时，从二进制内嵌的 default_skills 释放默认人格文件；
// 已有内容则不动，避免覆盖用户自定义。
func EnsureSkillsDir(dir string) error {
	if dir == "" {
		dir = "./skills"
	}

	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	root := "default_skills"
	err := fs.WalkDir(defaultSkills, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		data, err := defaultSkills.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0o644)
	})
	if err != nil {
		return err
	}

	log.Printf("已释放默认 skills 到 %s", dir)
	return nil
}
