package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cuetiy-backend/config"
	"cuetiy-backend/model"
	"cuetiy-backend/skill"
	"cuetiy-backend/utils"
)

type AIService struct {
	skillManager *skill.SkillManager
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type StreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

const chatCompletionsPath = "/chat/completions"

func (s *AIService) apiURL(baseURL string) string {
	base := strings.TrimRight(baseURL, "/")
	return base + chatCompletionsPath
}

func NewAIService(skillManager *skill.SkillManager) *AIService {
	return &AIService{skillManager: skillManager}
}

func (s *AIService) buildSystemPrompt(conv *model.Conversation) string {
	systemPrompt := s.skillManager.GetSystemPromptByPersona(conv.PersonaID)

	if conv != nil && conv.AINickname != "" {
		systemPrompt = fmt.Sprintf("你的名字是%s。\n%s", conv.AINickname, systemPrompt)
	}

	// TODO: 接入情绪摘要后，从此处注入用户情绪状态
	// emotionSummary, err := s.emotionRepo.FindByConversationID(conv.ID)
	// if err == nil && emotionSummary != nil {
	//     systemPrompt += fmt.Sprintf("\n\n## 用户近期状态\n%s", emotionSummary.Summary)
	// }

	return systemPrompt
}

func (s *AIService) buildMessages(conv *model.Conversation, userMessage string, history []model.Message) []ChatMessage {
	messages := []ChatMessage{
		{Role: "system", Content: s.buildSystemPrompt(conv)},
	}

	for _, msg := range history {
		messages = append(messages, ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	messages = append(messages, ChatMessage{
		Role:    "user",
		Content: userMessage,
	})

	return messages
}

// SendMessageWithMessages 使用已组装好的 messages 发送（推荐路径）
func (s *AIService) SendMessageWithMessages(messages []ChatMessage, onStream func(content string)) (string, error) {
	cfg := config.AppConfig
	if cfg.DeepSeekAPIKey == "" {
		return "", fmt.Errorf("DeepSeek API Key 未配置，请在 .env 文件中设置 DEEPSEEK_API_KEY")
	}
	if len(messages) == 0 {
		return "", fmt.Errorf("messages 为空")
	}

	reqBody := ChatRequest{
		Model:       "deepseek-chat",
		Messages:    messages,
		Stream:      true,
		Temperature: 0.8,
		MaxTokens:   2000,
	}
	return s.doStream(reqBody, onStream)
}

func (s *AIService) SendMessage(conv *model.Conversation, userMessage string, history []model.Message, onStream func(content string)) (string, error) {
	cfg := config.AppConfig
	if cfg.DeepSeekAPIKey == "" {
		return "", fmt.Errorf("DeepSeek API Key 未配置，请在 .env 文件中设置 DEEPSEEK_API_KEY")
	}

	messages := s.buildMessages(conv, userMessage, history)

	reqBody := ChatRequest{
		Model:       "deepseek-chat",
		Messages:    messages,
		Stream:      true,
		Temperature: 0.8,
		MaxTokens:   2000,
	}
	return s.doStream(reqBody, onStream)
}

func (s *AIService) doStream(reqBody ChatRequest, onStream func(content string)) (string, error) {
	cfg := config.AppConfig
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("请求序列化失败: %v", err)
	}

	httpReq, err := http.NewRequest("POST", s.apiURL(cfg.DeepSeekAPIURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.DeepSeekAPIKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	client := &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("API 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return "", fmt.Errorf("API 返回错误 [%d]: %s", resp.StatusCode, string(bodyBytes))
	}

	var fullContent strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var streamChunk StreamChunk
		if err := json.Unmarshal([]byte(data), &streamChunk); err != nil {
			continue
		}

		if len(streamChunk.Choices) > 0 {
			content := streamChunk.Choices[0].Delta.Content
			if content == "" {
				continue
			}
			fullContent.WriteString(content)

			if onStream != nil {
				onStream(content)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fullContent.String(), fmt.Errorf("读取流失败: %v", err)
	}

	sanitized := utils.SanitizeEmotionalTags(fullContent.String())
	return sanitized, nil
}

func (s *AIService) SendMessageNonStream(conv *model.Conversation, userMessage string, history []model.Message) (string, error) {
	cfg := config.AppConfig
	if cfg.DeepSeekAPIKey == "" {
		return "", fmt.Errorf("DeepSeek API Key 未配置")
	}

	messages := s.buildMessages(conv, userMessage, history)

	reqBody := ChatRequest{
		Model:       "deepseek-chat",
		Messages:    messages,
		Stream:      false,
		Temperature: 0.8,
		MaxTokens:   2000,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("请求序列化失败: %v", err)
	}

	httpReq, err := http.NewRequest("POST", s.apiURL(cfg.DeepSeekAPIURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.DeepSeekAPIKey)

	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("API 请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return "", fmt.Errorf("API 返回错误 [%d]: %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if len(chatResp.Choices) > 0 {
		sanitized := utils.SanitizeEmotionalTags(chatResp.Choices[0].Message.Content)
		return sanitized, nil
	}

	return "", fmt.Errorf("AI 返回空响应")
}
