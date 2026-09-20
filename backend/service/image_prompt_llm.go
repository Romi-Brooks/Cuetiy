package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"cuetiy-backend/config"
)

// ImagePromptLLM LLM 产出的场景向提示词（与 skills 约束组合后进 Image API）
type ImagePromptLLM struct {
	// SceneType: selfie | portrait | landscape | street | food | indoor | pet | object | other
	SceneType string `json:"scene_type"`
	// Prompt 场景/构图/光线描述；人物外貌不写，由 skills 注入
	Prompt string `json:"prompt"`
	// Caption 出图前的等待句 / 随图短句
	Caption string `json:"caption"`
	// HasPerson 画面是否出现人物（决定是否拼接 skills appearance）
	HasPerson bool `json:"has_person"`
}

// GenerateImagePromptLLM
// Skills 负责身份/风格硬约束；LLM 负责场景丰富度（风景/街拍/美食等，不限于自拍）。
func (s *AIService) GenerateImagePromptLLM(
	userMsg, aiReply, emotion string,
	skillAppearance, skillStyle string,
) (*ImagePromptLLM, error) {
	if s == nil {
		return nil, fmt.Errorf("ai service nil")
	}
	cfg := config.AppConfig
	if cfg == nil || strings.TrimSpace(cfg.DeepSeekAPIKey) == "" {
		return nil, fmt.Errorf("DeepSeek API Key 未配置")
	}

	sys := `你是文生图（TTI）提示词与出图旁白写手。只输出 JSON，不要其它文字。

根据对话决定「这张图拍什么」，不要永远自拍：
- selfie / portrait：角色出镜（自拍、半身、镜中、被抓拍）
- landscape：窗外、天空、海边、山、城市远景
- street：街边、店铺、夜景、行人
- food / indoor / pet / object / other：按对话意境选

规则：
1. prompt 写场景、构图、光线、氛围、材质；具体人物五官/发型/发色不要写（系统会按人设外貌拼接）
2. 若 has_person=false，不要出现人物外貌描述
3. 若 has_person=true，可写动作/表情/景别，但外貌交给系统
4. caption 是拟人等待句（≤18字），像真人打字，口语、可带语气词；不要说「AI/生成/图片」
5. 与用户消息和 AI 回复意境一致；可以是风景/物件，不必有人
6. prompt 建议 40–120 字，具体可感，避免空泛形容词堆砌
7. 不要输出水印、文字、logo 相关要求（系统会追加）

输出格式：
{"scene_type":"selfie|portrait|landscape|street|food|indoor|pet|object|other","prompt":"...","caption":"...","has_person":true}`

	var user strings.Builder
	user.WriteString("用户消息：")
	user.WriteString(truncRunes(userMsg, 200))
	user.WriteString("\nAI 回复：")
	user.WriteString(truncRunes(aiReply, 200))
	if emotion != "" {
		user.WriteString("\n情绪标签：")
		user.WriteString(emotion)
	}
	if skillAppearance != "" {
		user.WriteString("\n技能包人物外貌（仅当 has_person=true 时画面中会出现，你不必复述）：")
		user.WriteString(truncRunes(skillAppearance, 120))
	}
	if skillStyle != "" {
		user.WriteString("\n技能包默认画面风格（可参考语气，也可按场景改写，不必照抄）：")
		user.WriteString(truncRunes(skillStyle, 120))
	}
	user.WriteString("\n请输出 JSON。")

	raw, err := s.chatOnceJSON(sys, user.String(), 0.7, 300)
	if err != nil {
		return nil, err
	}
	out, err := parseImagePromptLLM(raw)
	if err != nil {
		return nil, err
	}
	if out.Caption = strings.TrimSpace(out.Caption); out.Caption != "" {
		out.Caption = truncRunes(out.Caption, 24)
	}
	if out.SceneType = strings.ToLower(strings.TrimSpace(out.SceneType)); out.SceneType == "" {
		out.SceneType = "other"
	}
	return out, nil
}

// GenerateImageWaitPhrase 仅生成出图等待句（轻量路径；与 prompt 同批时用 caption 即可）
func (s *AIService) GenerateImageWaitPhrase(userMsg, aiReply string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("ai service nil")
	}
	cfg := config.AppConfig
	if cfg == nil || strings.TrimSpace(cfg.DeepSeekAPIKey) == "" {
		return "", fmt.Errorf("DeepSeek API Key 未配置")
	}
	sys := `你是聊天拟人旁白。用户即将收到一张图，先写一句等待提示。
要求：口语、自然、≤18字、可带语气词；不要说 AI/生成/图片/Prompt；不要重复套话。
只输出这一句，不要引号、不要解释。`
	user := "用户：" + truncRunes(userMsg, 120) + "\nAI 刚回复：" + truncRunes(aiReply, 120)
	raw, err := s.chatOnceJSON(sys, user, 0.8, 60)
	if err != nil {
		return "", err
	}
	line := firstLine(raw)
	line = strings.Trim(line, "「」\"'“” ")
	if line == "" {
		return "", fmt.Errorf("empty wait phrase")
	}
	return truncRunes(line, 24), nil
}

// chatOnceJSON 非流式单轮，返回正文
func (s *AIService) chatOnceJSON(system, user string, temperature float64, maxTokens int) (string, error) {
	cfg := config.AppConfig
	if cfg == nil || cfg.DeepSeekAPIKey == "" {
		return "", fmt.Errorf("DeepSeek API Key 未配置")
	}
	body := marshalChatBody("deepseek-chat", []ChatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}, temperature, maxTokens, false)
	req, err := newDeepSeekRequest(s.apiURL(cfg.DeepSeekAPIURL), body, cfg.DeepSeekAPIKey)
	if err != nil {
		return "", err
	}
	resp, err := deepSeekClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("API 请求失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := readAllLimited(resp.Body, 16<<10)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API [%d]: %s", resp.StatusCode, truncRunes(string(raw), 200))
	}
	var chatResp ChatResponse
	if err := json.Unmarshal(raw, &chatResp); err != nil || len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("bad chat resp")
	}
	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}

func parseImagePromptLLM(raw string) (*ImagePromptLLM, error) {
	text := strings.TrimSpace(raw)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	// 截取首个 { 到最后一个 }
	if i := strings.IndexByte(text, '{'); i >= 0 {
		if j := strings.LastIndexByte(text, '}'); j > i {
			text = text[i : j+1]
		}
	}
	var out ImagePromptLLM
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("parse image prompt json: %w", err)
	}
	out.Prompt = strings.TrimSpace(out.Prompt)
	if out.Prompt == "" {
		return nil, fmt.Errorf("llm prompt empty")
	}
	return &out, nil
}

func truncRunes(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n])
}
