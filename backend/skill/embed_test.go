package skill_test
import (
  "os"
  "path/filepath"
  "testing"
  "rain-yi-backend/skill"
)
func TestEnsureSkillsDir(t *testing.T) {
  dir := filepath.Join(t.TempDir(), "skills")
  if err := skill.EnsureSkillsDir(dir); err != nil { t.Fatal(err) }
  if _, err := os.Stat(filepath.Join(dir, "SKILL-DEFAULT.md")); err != nil { t.Fatal(err) }
  // second call should not fail / should leave file
  if err := skill.EnsureSkillsDir(dir); err != nil { t.Fatal(err) }
}
