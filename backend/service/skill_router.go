package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"rain-yi-backend/config"
)

// EmotionLabel 情绪标签
type EmotionLabel string

const (
	EmotionNeutral   EmotionLabel = "neutral"
	EmotionJoy       EmotionLabel = "joy"
	EmotionSad       EmotionLabel = "sad"
	EmotionAnxious   EmotionLabel = "anxious"
	EmotionAngry     EmotionLabel = "angry"
	EmotionTired     EmotionLabel = "tired"
	EmotionFlirty    EmotionLabel = "flirty"
	EmotionNeedPro   EmotionLabel = "need_professional"
	EmotionPetPeeve  EmotionLabel = "pet_peeve"
	EmotionStyleAsk  EmotionLabel = "style_switch"
)

// EmotionResult 分析结果
type EmotionResult struct {
	Label      EmotionLabel `json:"label"`
	Confidence float64      `json:"confidence"`
	Reason     string       `json:"reason"`
}

// SkillRouter 情绪/意图 → 技能注入
type SkillRouter struct {
	ai *AIService
}

func NewSkillRouter(ai *AIService) *SkillRouter {
	return &SkillRouter{ai: ai}
}

var (
	reCode     = regexp.MustCompile(`(?i)(代码|报错|bug|python|java|前端|后端|算法|编译|接口|api|函数|变量|正则|git|docker|sql|javascript|typescript|golang|rust)`)
	reMusic    = regexp.MustCompile(`(?i)(音乐|歌|专辑|编曲|作曲|r&b|house|techno|dj|旋律|和弦|乐队|歌手|歌单)`)
	reStyle    = regexp.MustCompile(`(?i)(切换|御姐|软萌|清冷|撒娇模式|换.*风格|风格.*换)`)
	reJoy      = regexp.MustCompile(`(太好了|好开心|成功了|通过了|涨薪|中奖|厉害|万岁|开心|高兴|赢了)`)
	reSad      = regexp.MustCompile(`(难过|委屈|哭|崩溃|失落|沮丧|心烦|烦死了|emo|孤独|寂寞|被骂|失恋|失败了)`)
	reAnxious  = regexp.MustCompile(`(焦虑|紧张|担心|害怕|慌|压力大|睡不着|失眠)`)
	rePetPeeve = regexp.MustCompile(`(不理你|故意不理|敷衍|就这\?|哦哦|嗯嗯哦|随便|都行吧|不用你管)`)
	reFlirty   = regexp.MustCompile(`(想你|爱你|亲亲|抱抱|贴贴|宝贝|女朋友)`)
)

// AnalyzeEmotion 规则 + 可选模型 分析本轮情绪/意图
func (r *SkillRouter) AnalyzeEmotion(userMsg string, recentHint string) EmotionResult {
	rule := ruleEmotion(userMsg)
	if rule.Label != EmotionNeutral && rule.Confidence >= 0.6 {
		return rule
	}
	mode := "hybrid"
	if config.AppConfig != nil && config.AppConfig.SkillRouterMode != "" {
		mode = config.AppConfig.SkillRouterMode
	}
	if mode == "rule" || config.AppConfig == nil || config.AppConfig.DeepSeekAPIKey == "" {
		if rule.Label != EmotionNeutral {
			return rule
		}
		return EmotionResult{Label: EmotionNeutral, Confidence: 0.5, Reason: "rule_only"}
	}
	// hybrid：规则不确定时用小模型
	ai := r.ai
	if ai == nil {
		return rule
	}
	res, err := ai.ClassifyEmotion(userMsg, recentHint)
	if err != nil || res.Label == "" {
		return rule
	}
	return res
}

func ruleEmotion(msg string) EmotionResult {
	switch {
	case reStyle.MatchString(msg):
		return EmotionResult{Label: EmotionStyleAsk, Confidence: 0.75, Reason: "style_kw"}
	case rePetPeeve.MatchString(msg) && utf8Len(msg) < 30:
		return EmotionResult{Label: EmotionPetPeeve, Confidence: 0.7, Reason: "pet_kw"}
	case reCode.MatchString(msg):
		return EmotionResult{Label: EmotionNeedPro, Confidence: 0.72, Reason: "code_kw"}
	case reMusic.MatchString(msg) && !reSad.MatchString(msg):
		return EmotionResult{Label: EmotionNeedPro, Confidence: 0.65, Reason: "music_kw"}
	case reSad.MatchString(msg):
		return EmotionResult{Label: EmotionSad, Confidence: 0.8, Reason: "sad_kw"}
	case reAnxious.MatchString(msg):
		return EmotionResult{Label: EmotionAnxious, Confidence: 0.75, Reason: "anxious_kw"}
	case reJoy.MatchString(msg):
		return EmotionResult{Label: EmotionJoy, Confidence: 0.75, Reason: "joy_kw"}
	case reFlirty.MatchString(msg):
		return EmotionResult{Label: EmotionFlirty, Confidence: 0.7, Reason: "flirty_kw"}
	default:
		return EmotionResult{Label: EmotionNeutral, Confidence: 0.4, Reason: "no_match"}
	}
}

// CategoriesForEmotion 情绪 → 应注入的技能模块 category
func CategoriesForEmotion(e EmotionResult) []string {
	switch e.Label {
	case EmotionJoy, EmotionSad, EmotionAnxious, EmotionAngry, EmotionTired, EmotionFlirty:
		return []string{"emotion_companion"}
	case EmotionNeedPro:
		return []string{"professional_skills", "emotion_companion"}
	case EmotionPetPeeve:
		return []string{"trigger_rules", "emotion_companion"}
	case EmotionStyleAsk:
		return []string{"style_switch"}
	default:
		return nil
	}
}

// ClassifyEmotion 调用 DeepSeek 做轻量情绪分类（非流式、小 max_tokens）
func (s *AIService) ClassifyEmotion(userMsg string, recentHint string) (EmotionResult, error) {
	cfg := config.AppConfig
	if cfg == nil || cfg.DeepSeekAPIKey == "" {
		return EmotionResult{}, fmt.Errorf("no api key")
	}

	sys := `你是情绪分类器。只输出 JSON，不要其它文字。
可选标签：neutral, joy, sad, anxious, angry, tired, flirty, need_professional, pet_peeve, style_switch
含义：need_professional=咨询代码/音乐等专业问题；pet_peeve=故意冷落或敷衍；style_switch=要求切换说话风格。
格式：{"label":"...","confidence":0.0-1.0,"reason":"一句话"}`

	user := "用户消息：" + userMsg
	if recentHint != "" {
		user += "\n最近上下文摘要：" + TruncateRunes(recentHint, 200)
	}

	body := map[string]any{
		"model":       "deepseek-chat",
		"messages":    []ChatMessage{{Role: "system", Content: sys}, {Role: "user", Content: user}},
		"stream":      false,
		"temperature": 0.1,
		"max_tokens":  80,
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", s.apiURL(cfg.DeepSeekAPIURL), bytes.NewReader(data))
	if err != nil {
		return EmotionResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.DeepSeekAPIKey)

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return EmotionResult{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != 200 {
		return EmotionResult{}, fmt.Errorf("classify status %d", resp.StatusCode)
	}
	var chatResp ChatResponse
	if err := json.Unmarshal(raw, &chatResp); err != nil || len(chatResp.Choices) == 0 {
		return EmotionResult{}, fmt.Errorf("bad classify resp")
	}
	text := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	var out EmotionResult
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return EmotionResult{}, err
	}
	return out, nil
}

func utf8Len(s string) int {
	return len([]rune(s))
}
