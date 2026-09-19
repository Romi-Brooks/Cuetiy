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

	"cuetiy-backend/config"
	"cuetiy-backend/skill"
)

// EmotionLabel 情绪标签（兼容旧命名；通用路由下标签可来自技能包）
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
	EmotionComfort   EmotionLabel = "need_comfort"
)

// BaseClassifierLabels 无技能包时的基础标签空间
var BaseClassifierLabels = []string{
	"neutral", "joy", "sad", "anxious", "angry", "tired", "flirty",
	"need_professional", "need_comfort", "pet_peeve", "style_switch",
}

// EmotionResult 分析结果（兼容旧 debug 字段）
type EmotionResult struct {
	Label      EmotionLabel `json:"label"`
	Confidence float64      `json:"confidence"`
	Reason     string       `json:"reason"`
	// Tags 本轮命中的全部标签（含 label 与其它匹配）
	Tags []string `json:"tags,omitempty"`
	// Categories 注册表匹配出的 L2 category
	Categories []string `json:"categories,omitempty"`
}

// SkillRouter 情绪/意图 → 技能注入（registry 驱动，不写死具体技能包）
type SkillRouter struct {
	ai *AIService
}

func NewSkillRouter(ai *AIService) *SkillRouter {
	return &SkillRouter{ai: ai}
}

// 通用中文情绪/意图信号（与具体技能包无关；包专属词走 frontmatter.keywords）
var (
	genericSignals = []struct {
		tag string
		re  *regexp.Regexp
	}{
		{"style_switch", regexp.MustCompile(`(?i)(切换|换.*风格|风格.*换|模式切换)`)},
		{"pet_peeve", regexp.MustCompile(`(不理你|故意不理|敷衍|就这\?|哦哦|嗯嗯哦|随便|都行吧|不用你管)`)},
		{"need_professional", regexp.MustCompile(`(?i)(代码|报错|bug|python|java|前端|后端|算法|编译|接口|api|函数|变量|正则|git|docker|sql|javascript|typescript|golang|rust|音乐|歌|专辑|编曲|作曲|旋律|和弦|乐队|歌手|歌单)`)},
		{"sad", regexp.MustCompile(`(难过|委屈|哭|崩溃|失落|沮丧|心烦|烦死了|emo|孤独|寂寞|被骂|失恋|失败了)`)},
		{"anxious", regexp.MustCompile(`(焦虑|紧张|担心|害怕|慌|压力大|睡不着|失眠)`)},
		{"joy", regexp.MustCompile(`(太好了|好开心|成功了|通过了|涨薪|中奖|厉害|万岁|开心|高兴|赢了)`)},
		{"flirty", regexp.MustCompile(`(想你|爱你|亲亲|抱抱|贴贴|宝贝)`)},
		{"need_comfort", regexp.MustCompile(`(安慰|陪陪|抱抱我|好难受|撑不住|撑不下去)`)},
		{"angry", regexp.MustCompile(`(生气|愤怒|气死|讨厌|烦死)`)},
		{"tired", regexp.MustCompile(`(好累|疲惫|加班|熬夜|没精神|想睡)`)},
	}
)

// AnalyzeEmotion 兼容旧调用：无 registry 时走基础标签空间
func (r *SkillRouter) AnalyzeEmotion(userMsg string, recentHint string) EmotionResult {
	return r.Route(userMsg, recentHint, nil)
}

// Route 通用路由：规则（通用词 + 技能包关键词）→ 可选 AI 分类 → registry 匹配 category
func (r *SkillRouter) Route(userMsg, recentHint string, reg *skill.SkillRegistry) EmotionResult {
	rule := r.ruleTags(userMsg, reg)
	if rule.Label != EmotionNeutral && rule.Confidence >= 0.6 {
		rule.Categories = r.categoriesFor(rule, reg)
		return rule
	}

	mode := "hybrid"
	if config.AppConfig != nil && config.AppConfig.SkillRouterMode != "" {
		mode = config.AppConfig.SkillRouterMode
	}
	if mode == "rule" || config.AppConfig == nil || config.AppConfig.DeepSeekAPIKey == "" {
		if rule.Label != EmotionNeutral {
			rule.Categories = r.categoriesFor(rule, reg)
			return rule
		}
		out := EmotionResult{Label: EmotionNeutral, Confidence: 0.5, Reason: "rule_only", Tags: []string{"neutral"}}
		out.Categories = r.categoriesFor(out, reg)
		return out
	}

	ai := r.ai
	if ai == nil {
		rule.Categories = r.categoriesFor(rule, reg)
		return rule
	}
	labels := reg.ClassifierLabelList(BaseClassifierLabels)
	res, err := ai.ClassifyEmotionTags(userMsg, recentHint, labels)
	if err != nil || res.Label == "" {
		rule.Categories = r.categoriesFor(rule, reg)
		return rule
	}
	if len(res.Tags) == 0 {
		res.Tags = []string{string(res.Label)}
	}
	res.Categories = r.categoriesFor(res, reg)
	return res
}

func (r *SkillRouter) ruleTags(msg string, reg *skill.SkillRegistry) EmotionResult {
	type hit struct {
		tag  string
		conf float64
		reason string
	}
	var hits []hit

	// 优先：意图类通用信号（更具体）
	intentPriority := []string{"style_switch", "pet_peeve", "need_professional", "need_comfort"}
	emotionPriority := []string{"sad", "anxious", "joy", "flirty", "angry", "tired"}

	matchGeneric := func(tag string) *hit {
		for _, g := range genericSignals {
			if g.tag != tag {
				continue
			}
			if tag == "pet_peeve" && utf8Len(msg) >= 30 {
				return nil
			}
			if g.re.MatchString(msg) {
				conf := 0.75
				switch tag {
				case "sad":
					conf = 0.8
				case "pet_peeve":
					conf = 0.7
				case "need_professional":
					conf = 0.7
				case "style_switch":
					conf = 0.75
				case "flirty":
					conf = 0.7
				case "need_comfort":
					conf = 0.68
				}
				return &hit{tag: tag, conf: conf, reason: tag + "_kw"}
			}
			return nil
		}
		return nil
	}

	for _, tag := range intentPriority {
		if h := matchGeneric(tag); h != nil {
			hits = append(hits, *h)
		}
	}
	// 技能包自声明关键词
	if reg != nil {
		for _, m := range reg.TriggerModules() {
			for _, kw := range m.Keywords {
				if kw == "" || !strings.Contains(strings.ToLower(msg), strings.ToLower(kw)) {
					continue
				}
				for _, tag := range m.Triggers.AllTags() {
					hits = append(hits, hit{tag: tag, conf: 0.72, reason: "pkg_kw:" + kw})
				}
				if len(m.Triggers.AllTags()) == 0 && m.Category != "" {
					// 仅有 category 时用 category 作为弱标签
					hits = append(hits, hit{tag: m.Category, conf: 0.65, reason: "pkg_cat:" + m.Category})
				}
			}
		}
	}
	// 情绪类通用信号（intent 未命中或叠加）
	for _, tag := range emotionPriority {
		if h := matchGeneric(tag); h != nil {
			// 需要专业时情绪降为附加标签，不抢主 label
			hits = append(hits, *h)
		}
	}

	if len(hits) == 0 {
		return EmotionResult{Label: EmotionNeutral, Confidence: 0.4, Reason: "no_match", Tags: []string{"neutral"}}
	}

	// 主标签：优先 intent，再 emotion，再 conf
	rank := func(tag string) int {
		for i, t := range intentPriority {
			if t == tag {
				return 100 - i
			}
		}
		for i, t := range emotionPriority {
			if t == tag {
				return 50 - i
			}
		}
		return 0
	}
	primary := hits[0]
	for _, h := range hits[1:] {
		if rank(h.tag) > rank(primary.tag) || (rank(h.tag) == rank(primary.tag) && h.conf > primary.conf) {
			primary = h
		}
	}

	var tags []string
	seen := map[string]struct{}{}
	for _, h := range hits {
		if _, ok := seen[h.tag]; ok {
			continue
		}
		seen[h.tag] = struct{}{}
		tags = append(tags, h.tag)
	}

	return EmotionResult{
		Label:      EmotionLabel(primary.tag),
		Confidence: primary.conf,
		Reason:     primary.reason,
		Tags:       tags,
	}
}

func (r *SkillRouter) categoriesFor(res EmotionResult, reg *skill.SkillRegistry) []string {
	tags := res.Tags
	if len(tags) == 0 && res.Label != "" {
		tags = []string{string(res.Label)}
	}
	if reg != nil && len(reg.Modules) > 0 {
		cats := reg.MatchCategories(tags)
		if len(cats) > 0 {
			return cats
		}
		// 注册表存在但未命中：不再回落硬编码映射
		return nil
	}
	// 无 registry：兼容旧 YiSkill 类 category 约定
	return legacyCategoriesFor(res.Label)
}

// CategoriesForEmotion 兼容旧 API
func CategoriesForEmotion(e EmotionResult) []string {
	return legacyCategoriesFor(e.Label)
}

func legacyCategoriesFor(label EmotionLabel) []string {
	switch label {
	case EmotionJoy, EmotionSad, EmotionAnxious, EmotionAngry, EmotionTired, EmotionFlirty, EmotionComfort:
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

// ClassifyEmotion 兼容旧调用
func (s *AIService) ClassifyEmotion(userMsg string, recentHint string) (EmotionResult, error) {
	return s.ClassifyEmotionTags(userMsg, recentHint, BaseClassifierLabels)
}

// ClassifyEmotionTags 小模型分类，标签集由 registry 决定（通用协议）
func (s *AIService) ClassifyEmotionTags(userMsg string, recentHint string, labels []string) (EmotionResult, error) {
	cfg := config.AppConfig
	if cfg == nil || cfg.DeepSeekAPIKey == "" {
		return EmotionResult{}, fmt.Errorf("no api key")
	}
	if len(labels) == 0 {
		labels = BaseClassifierLabels
	}

	sys := fmt.Sprintf(`你是对话意图/情绪分类器。只输出 JSON，不要其它文字。
可选标签（仅可从中选择）：%s
要求：label 为最主要的一个标签；tags 为命中的全部标签（含 label）。
格式：{"label":"...","confidence":0.0-1.0,"reason":"一句话","tags":["..."]}`, strings.Join(labels, ", "))

	user := "用户消息：" + userMsg
	if recentHint != "" {
		user += "\n最近上下文摘要：" + TruncateRunes(recentHint, 200)
	}

	body := map[string]any{
		"model":       "deepseek-chat",
		"messages":    []ChatMessage{{Role: "system", Content: sys}, {Role: "user", Content: user}},
		"stream":      false,
		"temperature": 0.1,
		"max_tokens":  120,
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
	if len(out.Tags) == 0 && out.Label != "" {
		out.Tags = []string{string(out.Label)}
	}
	// 标签白名单过滤
	allow := map[string]struct{}{}
	for _, l := range labels {
		allow[strings.ToLower(strings.TrimSpace(l))] = struct{}{}
	}
	if _, ok := allow[strings.ToLower(string(out.Label))]; !ok && out.Label != "" {
		// 非法 label 时保留 tags，label 降为 neutral
		out.Reason = out.Reason + ";label_filtered"
		if len(out.Tags) > 0 {
			out.Label = EmotionLabel(out.Tags[0])
			if _, ok2 := allow[strings.ToLower(string(out.Label))]; !ok2 {
				out.Label = EmotionNeutral
			}
		} else {
			out.Label = EmotionNeutral
		}
	}
	var filtered []string
	for _, t := range out.Tags {
		if _, ok := allow[strings.ToLower(t)]; ok {
			filtered = append(filtered, t)
		}
	}
	out.Tags = filtered
	return out, nil
}

func utf8Len(s string) int {
	return len([]rune(s))
}
