package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cuetiy-backend/config"
	"cuetiy-backend/model"
)

// SummaryService 由 AI 做滚动摘要压缩
type SummaryService struct{}

func NewSummaryService() *SummaryService {
	return &SummaryService{}
}

// Compress 将一批较旧消息压缩为纪要；oldSummary 为空则新建
func (s *SummaryService) Compress(oldSummary string, msgs []model.Message, maxChars int) (string, error) {
	if len(msgs) == 0 {
		return oldSummary, nil
	}
	cfg := config.AppConfig
	if cfg == nil || cfg.DeepSeekAPIKey == "" {
		return fallbackSummary(oldSummary, msgs, maxChars), nil
	}

	var conv strings.Builder
	for _, m := range msgs {
		role := "用户"
		if m.Role == "assistant" {
			role = "助手"
		}
		conv.WriteString(role)
		conv.WriteString("：")
		conv.WriteString(TruncateRunes(m.Content, 200))
		conv.WriteString("\n")
	}

	sys := `你是对话压缩器，不是聊天角色。把对话压成第三人称纪要。
必须保留：用户称呼偏好、事实与承诺、情绪转折、未决话题、关键决定。
禁止：写成女友语气、编造未出现内容。
输出纯文本，不超过限制字数。`

	user := ""
	if oldSummary != "" {
		user += "## 已有摘要\n" + oldSummary + "\n\n"
	}
	user += "## 新增对话\n" + conv.String() + "\n"
	user += fmt.Sprintf("## 要求\n合并为一份新摘要，不超过 %d 个汉字。", maxChars)

	body := map[string]any{
		"model":       "deepseek-chat",
		"messages":    []ChatMessage{{Role: "system", Content: sys}, {Role: "user", Content: user}},
		"stream":      false,
		"temperature": 0.2,
		"max_tokens":  600,
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", s.apiURL(cfg.DeepSeekAPIURL), bytes.NewReader(data))
	if err != nil {
		return fallbackSummary(oldSummary, msgs, maxChars), nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.DeepSeekAPIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fallbackSummary(oldSummary, msgs, maxChars), nil
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode != 200 {
		return fallbackSummary(oldSummary, msgs, maxChars), nil
	}
	var chatResp ChatResponse
	if err := json.Unmarshal(raw, &chatResp); err != nil || len(chatResp.Choices) == 0 {
		return fallbackSummary(oldSummary, msgs, maxChars), nil
	}
	out := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	if out == "" {
		return fallbackSummary(oldSummary, msgs, maxChars), nil
	}
	if maxChars > 0 {
		out = TruncateRunes(out, maxChars)
	}
	return out, nil
}

func (s *SummaryService) apiURL(base string) string {
	return strings.TrimRight(base, "/") + "/chat/completions"
}

func fallbackSummary(old string, msgs []model.Message, maxChars int) string {
	var b strings.Builder
	if old != "" {
		b.WriteString(old)
		b.WriteString("\n")
	}
	b.WriteString("（近段纪要）")
	for i, m := range msgs {
		if i >= 8 {
			break
		}
		role := "U"
		if m.Role == "assistant" {
			role = "A"
		}
		b.WriteString(role)
		b.WriteString(":")
		b.WriteString(TruncateRunes(m.Content, 40))
		b.WriteString("; ")
	}
	return TruncateRunes(strings.TrimSpace(b.String()), maxChars)
}

// ExtractMemory 用 AI 从对话中抽取用户记忆卡字段
func (s *SummaryService) ExtractMemory(existingJSON string, msgs []model.Message, maxChars int) string {
	if len(msgs) == 0 {
		return existingJSON
	}
	cfg := config.AppConfig
	if cfg == nil || cfg.DeepSeekAPIKey == "" {
		return existingJSON
	}

	var conv strings.Builder
	for _, m := range msgs {
		if m.Role != "user" {
			continue
		}
		conv.WriteString(TruncateRunes(m.Content, 120))
		conv.WriteString("\n")
	}
	if strings.TrimSpace(conv.String()) == "" {
		return existingJSON
	}

	sys := `你是用户档案抽取器。只输出 JSON，不要其它文字。
格式：{"facts":[{"k":"键","v":"值"}],"open_threads":["未决话题"]}
只抽取用户明确表达的事实（职业、喜好、雷点、称呼偏好等），不要臆测。`

	user := ""
	if existingJSON != "" {
		user += "## 已有记忆\n" + existingJSON + "\n\n"
	}
	user += "## 新用户发言\n" + conv.String() + "\n合并更新，facts 控制在 12 条内。"

	body := map[string]any{
		"model":       "deepseek-chat",
		"messages":    []ChatMessage{{Role: "system", Content: sys}, {Role: "user", Content: user}},
		"stream":      false,
		"temperature": 0.1,
		"max_tokens":  400,
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", s.apiURL(cfg.DeepSeekAPIURL), bytes.NewReader(data))
	if err != nil {
		return existingJSON
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.DeepSeekAPIKey)
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return existingJSON
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != 200 {
		return existingJSON
	}
	var chatResp ChatResponse
	if err := json.Unmarshal(raw, &chatResp); err != nil || len(chatResp.Choices) == 0 {
		return existingJSON
	}
	out := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	out = strings.TrimPrefix(out, "```json")
	out = strings.TrimPrefix(out, "```")
	out = strings.TrimSuffix(out, "```")
	out = strings.TrimSpace(out)
	if !strings.HasPrefix(out, "{") {
		return existingJSON
	}
	if maxChars > 0 {
		out = TruncateRunes(out, maxChars)
	}
	return out
}
