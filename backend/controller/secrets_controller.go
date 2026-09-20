package controller

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"cuetiy-backend/config"

	"github.com/gin-gonic/gin"
)

// SecretsController 运行时配置 AI 密钥（普通用户不打开 .env）
// 响应只返回「是否已配置」，绝不回显密钥明文。
type SecretsController struct{}

func NewSecretsController() *SecretsController {
	return &SecretsController{}
}

type secretsStatus struct {
	DeepSeekConfigured bool `json:"deepseek_configured"`
	MIMOConfigured     bool `json:"mimo_configured"`
	TTSEnabled         bool `json:"tts_enabled"`
	GRSAIConfigured    bool `json:"grsai_configured"`
	ImageGenEnabled    bool `json:"image_gen_enabled"`
	ImageGenModel      string `json:"image_gen_model"`
	StoragePath        string `json:"env_path"`
}

type secretsUpdateRequest struct {
	DeepSeekAPIKey *string `json:"deepseek_api_key"`
	MIMOAPIKey     *string `json:"mimo_api_key"`
	GRSAIAPIKey    *string `json:"grsai_api_key"`
	// 空字符串表示清除
	ClearDeepSeek bool `json:"clear_deepseek"`
	ClearMIMO     bool `json:"clear_mimo"`
	ClearGRSAI    bool `json:"clear_grsai"`
	// 图片生成开关与模型（可选）
	ImageGenEnabled *bool   `json:"image_gen_enabled"`
	ImageGenModel   *string `json:"image_gen_model"`
	ImageGenAspect  *string `json:"image_gen_aspect"`
	ImageGenQuality *string `json:"image_gen_quality"`
}

func (ctl *SecretsController) GetStatus(c *gin.Context) {
	cfg := config.AppConfig
	if cfg == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "配置未加载"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"deepseek_configured": strings.TrimSpace(cfg.DeepSeekAPIKey) != "",
		"mimo_configured":     strings.TrimSpace(cfg.MIMOAPIKey) != "",
		"tts_enabled":         cfg.TTSEnabled,
		"grsai_configured":    strings.TrimSpace(cfg.GRSAIAPIKey) != "",
		"image_gen_enabled":   cfg.ImageGenEnabled,
		"image_gen_model":     cfg.ImageGenModel,
		"note":                "密钥仅保存在服务器/本机 .env，接口不回显明文",
	})
}

func (ctl *SecretsController) Update(c *gin.Context) {
	cfg := config.AppConfig
	if cfg == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "配置未加载"})
		return
	}
	var req secretsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	changed := map[string]string{}

	if req.ClearDeepSeek {
		cfg.DeepSeekAPIKey = ""
		changed["DEEPSEEK_API_KEY"] = ""
	} else if req.DeepSeekAPIKey != nil {
		key := sanitizeKey(*req.DeepSeekAPIKey)
		if key != "" && !looksLikeKey(key) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "DeepSeek Key 格式不正确（应类似 sk-...）"})
			return
		}
		cfg.DeepSeekAPIKey = key
		changed["DEEPSEEK_API_KEY"] = key
	}

	if req.ClearMIMO {
		cfg.MIMOAPIKey = ""
		changed["MIMO_API_KEY"] = ""
	} else if req.MIMOAPIKey != nil {
		key := sanitizeKey(*req.MIMOAPIKey)
		if key != "" && len(key) < 8 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "MiMo Key 过短"})
			return
		}
		cfg.MIMOAPIKey = key
		changed["MIMO_API_KEY"] = key
	}

	if req.ClearGRSAI {
		cfg.GRSAIAPIKey = ""
		changed["GRSAI_API_KEY"] = ""
	} else if req.GRSAIAPIKey != nil {
		key := sanitizeKey(*req.GRSAIAPIKey)
		if key != "" && len(key) < 8 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Grsai Key 过短"})
			return
		}
		cfg.GRSAIAPIKey = key
		changed["GRSAI_API_KEY"] = key
	}

	if req.ImageGenEnabled != nil {
		cfg.ImageGenEnabled = *req.ImageGenEnabled
		if cfg.ImageGenEnabled {
			changed["IMAGE_GEN_ENABLED"] = "true"
		} else {
			changed["IMAGE_GEN_ENABLED"] = "false"
		}
	}
	if req.ImageGenModel != nil && strings.TrimSpace(*req.ImageGenModel) != "" {
		cfg.ImageGenModel = strings.TrimSpace(*req.ImageGenModel)
		changed["IMAGE_GEN_MODEL"] = cfg.ImageGenModel
	}
	if req.ImageGenAspect != nil && strings.TrimSpace(*req.ImageGenAspect) != "" {
		cfg.ImageGenAspect = strings.TrimSpace(*req.ImageGenAspect)
		changed["IMAGE_GEN_ASPECT"] = cfg.ImageGenAspect
	}
	if req.ImageGenQuality != nil && strings.TrimSpace(*req.ImageGenQuality) != "" {
		cfg.ImageGenQuality = strings.TrimSpace(*req.ImageGenQuality)
		changed["IMAGE_GEN_QUALITY"] = cfg.ImageGenQuality
	}

	if len(changed) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有需要更新的密钥"})
		return
	}

	if err := persistEnvKeys(changed); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入 .env 失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":              "配置已保存（立即生效，不回显）",
		"deepseek_configured":  strings.TrimSpace(cfg.DeepSeekAPIKey) != "",
		"mimo_configured":      strings.TrimSpace(cfg.MIMOAPIKey) != "",
		"grsai_configured":     strings.TrimSpace(cfg.GRSAIAPIKey) != "",
		"image_gen_enabled":    cfg.ImageGenEnabled,
		"image_gen_model":      cfg.ImageGenModel,
	})
}

func sanitizeKey(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`)
	return s
}

var deepseekKeyRe = regexp.MustCompile(`^sk-[A-Za-z0-9_\-]{8,}$`)

func looksLikeKey(s string) bool {
	// 兼容自定义网关：非空且无空白即可；标准 sk- 更严格
	if strings.ContainsAny(s, " \t\n") {
		return false
	}
	if strings.HasPrefix(s, "sk-") {
		return deepseekKeyRe.MatchString(s)
	}
	return len(s) >= 8
}

// persistEnvKeys 合并写入运行目录 .env（只改指定键）
func persistEnvKeys(kv map[string]string) error {
	path := resolveEnvPath()
	raw := ""
	if b, err := os.ReadFile(path); err == nil {
		raw = string(b)
	}
	lines := strings.Split(raw, "\n")
	seen := map[string]bool{}
	out := make([]string, 0, len(lines)+len(kv))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		replaced := false
		for k, v := range kv {
			if strings.HasPrefix(trimmed, k+"=") || strings.HasPrefix(trimmed, "# "+k+"=") {
				if v == "" {
					out = append(out, k+"=")
				} else {
					out = append(out, k+"="+v)
				}
				seen[k] = true
				replaced = true
				break
			}
		}
		if !replaced {
			out = append(out, line)
		}
	}
	for k, v := range kv {
		if !seen[k] {
			if v == "" {
				out = append(out, k+"=")
			} else {
				out = append(out, k+"="+v)
			}
		}
	}
	content := strings.Join(out, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}

func resolveEnvPath() string {
	candidates := []string{".env"}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, ".env"))
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), ".env"))
	}
	seen := map[string]bool{}
	for _, p := range candidates {
		abs, err := filepath.Abs(p)
		if err != nil || seen[abs] {
			continue
		}
		seen[abs] = true
		if _, err := os.Stat(abs); err == nil {
			return abs
		}
	}
	// 不存在则写在 cwd
	if wd, err := os.Getwd(); err == nil {
		return filepath.Join(wd, ".env")
	}
	return ".env"
}
