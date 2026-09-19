package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cuetiy-backend/config"
	"cuetiy-backend/model"
)

// TTSService 调用 MiMo TTS（OpenAI 兼容），结果落到本地 files/tts
type TTSService struct {
	storage   FileStorage
	voiceRepo interface {
		FindByUserID(userID int64) (*model.UserVoice, error)
	}
}

func NewTTSService(storage FileStorage, voiceRepo interface {
	FindByUserID(userID int64) (*model.UserVoice, error)
}) *TTSService {
	return &TTSService{storage: storage, voiceRepo: voiceRepo}
}

var emotionalStrip = regexp.MustCompile(`(?is)<emotional>[\s\S]*?</emotional>`)

// StylePromptFromEmotion 把上下文情绪映射为 MiMo TTS 风格指令（user 消息）
// 文档：自然语言控制放 role=user；也可在 assistant 正文加 (风格) 标签
func StylePromptFromEmotion(emotion string) string {
	base := "中文女声，亲密女友聊天，自然口语，语速适中，不要播音腔。"
	switch strings.ToLower(strings.TrimSpace(emotion)) {
	case "joy", "happy", "excited":
		return base + "语气轻快上扬，带着开心和一点点得意，像收到好消息在跟宝宝分享。"
	case "sad":
		return base + "语速稍慢，声音软一点，带着委屈和需要安慰的感觉，不要过分哭腔。"
	case "anxious":
		return base + "语速略快，有一点点着急和担心，但仍然温柔。"
	case "angry", "pet_peeve":
		return base + "可爱的小别扭、小撒娇式的生气，尾音上扬，绝不凶狠或攻击性。"
	case "tired":
		return base + "声音轻、略慵懒，像有点累但还在陪宝宝说话。"
	case "flirty":
		return base + "软糯撒娇，稍微拖一点尾音，亲昵但不过分。"
	case "need_professional":
		return base + "耐心温柔，讲解感清晰，像在认真帮宝宝解决问题。"
	case "style_switch":
		return base + "稍微沉稳一点的温柔，有安全感。"
	default:
		return base + "温柔软萌，轻松治愈。"
	}
}

// AudioStyleTagFromEmotion 可选：写在 assistant 正文开头的 (风格) 标签
func AudioStyleTagFromEmotion(emotion string) string {
	switch strings.ToLower(strings.TrimSpace(emotion)) {
	case "joy":
		return "开心"
	case "sad":
		return "委屈"
	case "anxious":
		return "忐忑"
	case "angry", "pet_peeve":
		return "撒娇"
	case "tired":
		return "疲惫"
	case "flirty":
		return "动情"
	case "need_professional":
		return "温柔"
	default:
		return ""
	}
}

// StyleFromEmotionalTags 从 <emotional>动作</emotional> 里猜额外语气线索
func StyleFromEmotionalTags(text string) string {
	low := strings.ToLower(text)
	var hints []string
	contains := func(sub string) bool { return strings.Contains(low, sub) }
	if contains("害羞") || contains("捂脸") {
		hints = append(hints, "害羞")
	}
	if contains("得意") || contains("晃脑") {
		hints = append(hints, "得意")
	}
	if contains("哭") || contains("委屈") {
		hints = append(hints, "委屈")
	}
	if contains("生气") || contains("哼") {
		hints = append(hints, "撒娇式的小别扭")
	}
	if contains("笑") {
		hints = append(hints, "带着笑意")
	}
	if len(hints) == 0 {
		return ""
	}
	return "另外注意：" + strings.Join(hints, "，") + "的语气，要自然体现在声音里。"
}

// StripForTTS 去掉动作标签，避免念出来
func StripForTTS(text string) string {
	s := emotionalStrip.ReplaceAllString(text, "")
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	return s
}

type ttsChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ttsRequest struct {
	Model    string `json:"model"`
	Messages []ttsChatMessage `json:"messages"`
	Audio    struct {
		Format string `json:"format"`
		Voice  string `json:"voice,omitempty"`
	} `json:"audio"`
}

type ttsResponse struct {
	Choices []struct {
		Message struct {
			Audio struct {
				Data string `json:"data"`
			} `json:"audio"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// TTSDebug 发给 MiMo TTS 的请求摘要（调试用，不含 base64 音色全文）
type TTSDebug struct {
	Model          string `json:"model"`
	APIBase        string `json:"api_base"`
	Voice          string `json:"voice"` // 预置名或 data:audio/... 前缀说明
	VoiceMode      string `json:"voice_mode"` // preset | voiceclone
	Format         string `json:"format"`
	StylePrompt    string `json:"style_prompt"`
	Emotion        string `json:"emotion"`
	AudioStyleTag  string `json:"audio_style_tag"`
	TextSent       string `json:"text_sent"`       // 实际发给 TTS 的正文（已去 emotional，可能含风格前缀）
	TextOriginal   string `json:"text_original"`   // 原始回复
	CharCount      int    `json:"char_count"`
	Truncated      bool   `json:"truncated"`
	AudioURL       string `json:"audio_url"`
	StoragePath    string `json:"storage_path"`
	AudioBytes     int    `json:"audio_bytes"`
	CacheHit       bool   `json:"cache_hit"`
	DurationMs     int64  `json:"duration_ms"`
	LatencyMs      int64  `json:"latency_ms"`
	Error          string `json:"error,omitempty"`
}

// Synthesize 合成语音；userID>0 时优先用该用户上传的音色（voiceclone）
// style 为空则用默认女友风格；emotion 用于生成风格指令与 (风格) 标签
func (s *TTSService) Synthesize(userID int64, text, style, emotion string) (url string, path string, dbg *TTSDebug, err error) {
	cfg := config.AppConfig
	started := time.Now()
	dbg = &TTSDebug{
		TextOriginal: text,
		Emotion:      emotion,
	}
	if cfg == nil {
		dbg.Error = "config not ready"
		return "", "", dbg, fmt.Errorf("config not ready")
	}
	dbg.APIBase = cfg.MIMOAPIBase
	if !cfg.TTSEnabled {
		dbg.Error = "TTS disabled"
		return "", "", dbg, fmt.Errorf("TTS 未启用")
	}
	if cfg.MIMOAPIKey == "" {
		dbg.Error = "MIMO_API_KEY missing"
		return "", "", dbg, fmt.Errorf("MIMO_API_KEY 未配置")
	}

	original := text
	clean := StripForTTS(text)
	if clean == "" {
		dbg.Error = "empty text"
		return "", "", dbg, fmt.Errorf("文本为空")
	}
	maxChars := cfg.TTSMaxChars
	if maxChars <= 0 {
		maxChars = 800
	}
	runes := []rune(clean)
	if len(runes) > maxChars {
		clean = string(runes[:maxChars])
		dbg.Truncated = true
	}

	// 风格：显式 style > 情绪映射 + 动作标签线索
	if strings.TrimSpace(style) == "" {
		style = StylePromptFromEmotion(emotion)
		if extra := StyleFromEmotionalTags(original); extra != "" {
			style = style + extra
		}
	}
	dbg.StylePrompt = style

	// 文档：assistant 正文开头可加 (风格) 标签，强化语气
	tag := AudioStyleTagFromEmotion(emotion)
	dbg.AudioStyleTag = tag
	if tag != "" && !strings.HasPrefix(clean, "(") && !strings.HasPrefix(clean, "（") {
		clean = "(" + tag + ")" + clean
	}
	dbg.TextSent = clean
	dbg.CharCount = len([]rune(clean))
	dbg.TextOriginal = original

	// 用户私有音色 → voiceclone；否则全局预置音色
	modelID := cfg.MIMOTTSModel
	voiceParam := cfg.MIMOTTSVoice
	userVoiceID := int64(0)
	dbg.VoiceMode = "preset"
	if userID > 0 && s.voiceRepo != nil {
		if uv, verr := s.voiceRepo.FindByUserID(userID); verr == nil && uv != nil && uv.StoragePath != "" {
			data, mime, lerr := LoadUserVoiceData(uv.StoragePath)
			if lerr == nil && len(data) > 0 {
				b64 := base64.StdEncoding.EncodeToString(data)
				voiceParam = fmt.Sprintf("data:%s;base64,%s", mime, b64)
				modelID = "mimo-v2.5-tts-voiceclone"
				userVoiceID = uv.ID
				dbg.VoiceMode = "voiceclone"
				dbg.Voice = fmt.Sprintf("data:%s;base64,<%d bytes sample>", mime, len(data))
			}
		}
	}
	if dbg.VoiceMode == "preset" {
		dbg.Voice = cfg.MIMOTTSVoice
	}
	if modelID == "" {
		modelID = "mimo-v2.5-tts"
	}
	dbg.Model = modelID
	dbg.Format = cfg.MIMOTTSFormat
	if dbg.Format == "" {
		dbg.Format = "wav"
	}

	// 缓存：同用户音色+文本只合成一次
	key := fmt.Sprintf("%d|%s|%s|%s|%s", userVoiceID, modelID, voiceParam[:min(32, len(voiceParam))], style, clean)
	sum := sha256.Sum256([]byte(key))
	hash := hex.EncodeToString(sum[:8])
	ext := dbg.Format
	objectName := fmt.Sprintf("tts/%s.%s", hash, ext)
	if s.storage != nil {
		if rc, gerr := s.storage.Get(objectName); gerr == nil {
			rc.Close()
			dbg.CacheHit = true
			dbg.AudioURL = s.storage.GetURL(objectName)
			dbg.StoragePath = objectName
			dbg.LatencyMs = time.Since(started).Milliseconds()
			dbg.DurationMs = estimateVoiceDurationMs(clean)
			return s.storage.GetURL(objectName), objectName, dbg, nil
		}
	}

	base := strings.TrimRight(cfg.MIMOAPIBase, "/")
	if base == "" {
		base = "https://api.xiaomimimo.com/v1"
	}

	msgs := []ttsChatMessage{}
	if strings.TrimSpace(style) != "" {
		msgs = append(msgs, ttsChatMessage{Role: "user", Content: style})
	} else {
		msgs = append(msgs, ttsChatMessage{
			Role:    "user",
			Content: StylePromptFromEmotion(emotion),
		})
		dbg.StylePrompt = msgs[0].Content
	}
	msgs = append(msgs, ttsChatMessage{Role: "assistant", Content: clean})

	var reqBody ttsRequest
	reqBody.Model = modelID
	reqBody.Messages = msgs
	reqBody.Audio.Format = ext
	if voiceParam != "" {
		reqBody.Audio.Voice = voiceParam
	}

	payload, _ := json.Marshal(reqBody)
	httpReq, err := http.NewRequest("POST", base+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		dbg.Error = err.Error()
		return "", "", dbg, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.MIMOAPIKey)

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		dbg.Error = err.Error()
		dbg.LatencyMs = time.Since(started).Milliseconds()
		return "", "", dbg, fmt.Errorf("TTS 请求失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	dbg.LatencyMs = time.Since(started).Milliseconds()
	if resp.StatusCode != 200 {
		snippet := string(raw)
		if len(snippet) > 400 {
			snippet = snippet[:400]
		}
		dbg.Error = snippet
		return "", "", dbg, fmt.Errorf("TTS API [%d]: %s", resp.StatusCode, snippet)
	}

	var tr ttsResponse
	if err := json.Unmarshal(raw, &tr); err != nil {
		dbg.Error = err.Error()
		return "", "", dbg, fmt.Errorf("TTS 响应解析失败: %w", err)
	}
	if tr.Error != nil && tr.Error.Message != "" {
		dbg.Error = tr.Error.Message
		return "", "", dbg, fmt.Errorf("TTS 错误: %s", tr.Error.Message)
	}
	if len(tr.Choices) == 0 || tr.Choices[0].Message.Audio.Data == "" {
		dbg.Error = "empty audio"
		return "", "", dbg, fmt.Errorf("TTS 返回空音频")
	}

	audio, err := base64.StdEncoding.DecodeString(tr.Choices[0].Message.Audio.Data)
	if err != nil {
		dbg.Error = err.Error()
		return "", "", dbg, fmt.Errorf("音频 base64 解码失败: %w", err)
	}
	if len(audio) == 0 {
		dbg.Error = "empty audio bytes"
		return "", "", dbg, fmt.Errorf("音频为空")
	}
	dbg.AudioBytes = len(audio)
	dbg.DurationMs = estimateVoiceDurationMs(clean)

	if s.storage != nil {
		if _, serr := s.storage.SaveToPath(
			objectName, 0, "tts", "message", 0,
			filepath.Base(objectName),
			bytes.NewReader(audio), int64(len(audio)),
		); serr != nil {
			_ = writeTTSFallback(objectName, audio)
		}
		dbg.AudioURL = s.storage.GetURL(objectName)
		dbg.StoragePath = objectName
		return s.storage.GetURL(objectName), objectName, dbg, nil
	}

	if err := writeTTSFallback(objectName, audio); err != nil {
		dbg.Error = err.Error()
		return "", "", dbg, err
	}
	dbg.AudioURL = "/storage/" + objectName
	dbg.StoragePath = objectName
	return "/storage/" + objectName, objectName, dbg, nil
}

func estimateVoiceDurationMs(text string) int64 {
	n := len([]rune(text))
	if n == 0 {
		return 800
	}
	ms := int64(float64(n) / 4.5 * 1000)
	if ms < 800 {
		ms = 800
	}
	if ms > 60000 {
		ms = 60000
	}
	return ms
}

func writeTTSFallback(objectName string, data []byte) error {
	root := "./data/files"
	if config.AppConfig != nil && config.AppConfig.StorageDir != "" {
		root = config.AppConfig.StorageDir
	}
	full := filepath.Join(root, filepath.FromSlash(objectName))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, data, 0o644)
}
