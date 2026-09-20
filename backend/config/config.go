package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// DBDriver: postgres（默认）| mysql | sqlite
	DBDriver string
	DBHost   string
	DBPort   string
	DBUser   string
	DBPassword string
	DBName     string
	// DBSSLMode 仅 postgres：disable / require / verify-ca / verify-full
	DBSSLMode string
	// SQLitePath 仅 sqlite：库文件路径，默认 <RuntimeDir>/data/cuetiy.db
	SQLitePath string

	ServerPort string
	ServerHost string

	JWTSecret string

	DeepSeekAPIKey string
	DeepSeekAPIURL string

	FrontendURL string
	SkillsDir   string

	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	// RuntimeDir：进程工作目录；.env / skills / data 相对 cwd 或本目录
	RuntimeDir string
	StorageDir string
	ArchiveDir string

	EnableEmotionSummary   bool
	EmotionSummaryInterval int

	// 上下文组装
	ContextTokenBudget      int
	ContextCompactThreshold float64
	RecentMinMessages       int
	SummaryMaxChars         int
	MemoryMaxTokens         int
	SkillL2TTLTurns         int
	SkillRouterMode         string // rule | model | hybrid
	EnableAmbientTriggers   bool

	// MiMo TTS
	TTSEnabled    bool
	MIMOAPIKey    string
	MIMOAPIBase   string
	MIMOTTSModel  string
	MIMOTTSVoice  string
	MIMOTTSFormat string
	TTSMaxChars   int

	// 语音条策略：用户点名必发；AI 在概率内自选
	VoiceReplyProbability float64 // 0~1，每条文字回复后是否额外/改为发语音条
	VoiceReplyCooldown    int     // 连续语音条之间的最少文字条数

	// 图片生成（Grsai gpt-image-2.5）
	ImageGenEnabled       bool
	GRSAIAPIKey           string
	GRSAIAPIBase          string
	ImageGenModel         string
	ImageGenAspect        string
	ImageGenQuality       string
	ImageGenReplyType     string
	ImageGenStylePrompt   string
	// ImageGenLLMPrompt 是否用 LLM 生成场景向提示词（与 skills 组合）
	ImageGenLLMPrompt bool
	// 限流：Max<=0 或 WindowMin<=0 表示关闭（当前默认关闭 30 分钟窗）
	ImageGenLimitMax       int
	ImageGenLimitWindowMin int
}

var AppConfig *Config

// loadEnvFile 加载 .env（cwd、可执行文件目录）
func loadEnvFile() {
	candidates := []string{".env"}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, ".env"))
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates, filepath.Join(dir, ".env"))
	}

	seen := map[string]bool{}
	for _, p := range candidates {
		abs, err := filepath.Abs(p)
		if err != nil || seen[abs] {
			continue
		}
		seen[abs] = true
		if _, err := os.Stat(abs); err != nil {
			continue
		}
		if err := godotenv.Load(abs); err != nil {
			log.Printf("警告: 加载 %s 失败: %v", abs, err)
			continue
		}
		log.Printf("已加载环境配置: %s", abs)
		return
	}
	log.Printf("警告: 未找到 .env，将使用默认值/进程环境变量（DB 默认 127.0.0.1）")
}

func LoadConfig() *Config {
	loadEnvFile()

	// 约定：exe/.env/skills/files 与 cwd 对齐（默认当前目录）
	runtimeDir := getEnv("RUNTIME_DIR", ".")
	AppConfig = &Config{
		DBDriver:       getEnv("DB_DRIVER", "postgres"),
		DBHost:         getEnv("DB_HOST", "127.0.0.1"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "postgres"),
		DBPassword:     getEnv("DB_PASSWORD", ""),
		DBName:         getEnv("DB_NAME", "cuetiy"),
		DBSSLMode:      getEnv("DB_SSLMODE", "disable"),
		SQLitePath:     getEnv("SQLITE_PATH", ""),
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		ServerHost:     getEnv("SERVER_HOST", "0.0.0.0"),
		JWTSecret:      getEnv("JWT_SECRET", "cuetiy-secret"),
		DeepSeekAPIKey: getEnv("DEEPSEEK_API_KEY", ""),
		DeepSeekAPIURL: getEnv("DEEPSEEK_API_URL", "https://api.deepseek.com"),
		FrontendURL:    getEnv("FRONTEND_URL", "http://localhost:5173"),
		SkillsDir:      getEnv("SKILLS_DIR", "./skills"),

		RedisHost:     getEnv("REDIS_HOST", ""),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),

		RuntimeDir: runtimeDir,
		StorageDir: getEnv("STORAGE_DIR", "./data/files"),
		ArchiveDir: getEnv("ARCHIVE_DIR", "./data/archives"),

		EnableEmotionSummary:   getEnv("ENABLE_EMOTION_SUMMARY", "false") == "true",
		EmotionSummaryInterval: getEnvAsInt("EMOTION_SUMMARY_INTERVAL", 10),

		ContextTokenBudget:      getEnvAsInt("CONTEXT_TOKEN_BUDGET", 16384),
		ContextCompactThreshold: getEnvAsFloat("CONTEXT_COMPACT_THRESHOLD", 0.55),
		RecentMinMessages:       getEnvAsInt("RECENT_MIN_MESSAGES", 6),
		SummaryMaxChars:         getEnvAsInt("SUMMARY_MAX_CHARS", 800),
		MemoryMaxTokens:         getEnvAsInt("MEMORY_MAX_TOKENS", 400),
		SkillL2TTLTurns:         getEnvAsInt("SKILL_L2_TTL_TURNS", 5),
		SkillRouterMode:         getEnv("SKILL_ROUTER_MODE", "hybrid"),
		EnableAmbientTriggers:   getEnv("ENABLE_AMBIENT_TRIGGERS", "false") == "true",

		TTSEnabled:    getEnv("TTS_ENABLED", "true") == "true",
		MIMOAPIKey:    getEnv("MIMO_API_KEY", ""),
		MIMOAPIBase:   getEnv("MIMO_API_BASE", "https://api.xiaomimimo.com/v1"),
		MIMOTTSModel:  getEnv("MIMO_TTS_MODEL", "mimo-v2.5-tts"),
		MIMOTTSVoice:  getEnv("MIMO_TTS_VOICE", "冰糖"),
		MIMOTTSFormat: getEnv("MIMO_TTS_FORMAT", "wav"),
		TTSMaxChars:   getEnvAsInt("TTS_MAX_CHARS", 800),

		// 默认 0：AI 自选语音条暂关闭；仅用户 want_voice 时发送
		VoiceReplyProbability: getEnvAsFloat("VOICE_REPLY_PROBABILITY", 0),
		VoiceReplyCooldown:    getEnvAsInt("VOICE_REPLY_COOLDOWN", 3),

		ImageGenEnabled: getEnv("IMAGE_GEN_ENABLED", "true") == "true",
		GRSAIAPIKey:     getEnv("GRSAI_API_KEY", ""),
		GRSAIAPIBase:    getEnv("GRSAI_API_BASE", "https://grsaiapi.com"),
		ImageGenModel:   getEnv("IMAGE_GEN_MODEL", "gpt-image-2.5"),
		ImageGenAspect:  getEnv("IMAGE_GEN_ASPECT", "1:1"),
		ImageGenQuality: getEnv("IMAGE_GEN_QUALITY", "auto"),
		ImageGenReplyType: getEnv("IMAGE_GEN_REPLY_TYPE", "json"),
		ImageGenStylePrompt: getEnv("IMAGE_GEN_STYLE_PROMPT",
			"手机自拍风格，真实生活感，自然光线，像恋人从相册发给你的照片，轻微生活场景，构图自然"),
		ImageGenLLMPrompt:      getEnv("IMAGE_GEN_LLM_PROMPT", "true") == "true",
		// 默认 0：30 分钟出图限流暂时关闭；恢复时设 IMAGE_GEN_LIMIT_MAX=2 IMAGE_GEN_LIMIT_WINDOW_MIN=30
		ImageGenLimitMax:       getEnvAsInt("IMAGE_GEN_LIMIT_MAX", 0),
		ImageGenLimitWindowMin: getEnvAsInt("IMAGE_GEN_LIMIT_WINDOW_MIN", 0),
	}

	for _, dir := range []string{
		AppConfig.RuntimeDir,
		AppConfig.SkillsDir,
		AppConfig.StorageDir,
		AppConfig.ArchiveDir,
	} {
		if dir == "" {
			continue
		}
		_ = os.MkdirAll(dir, 0o755)
	}

	redisHost := AppConfig.RedisHost
	if redisHost == "" {
		redisHost = "(未配置)"
	}
	log.Printf("DB driver=%s target=%s@%s:%s/%s sqlite=%s (Redis: %s) runtime=%s",
		AppConfig.DBDriver, AppConfig.DBUser, AppConfig.DBHost, AppConfig.DBPort, AppConfig.DBName,
		AppConfig.SQLitePath, redisHost, AppConfig.RuntimeDir)

	return AppConfig
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil && f > 0 && f <= 1 {
			return f
		}
	}
	return defaultValue
}
