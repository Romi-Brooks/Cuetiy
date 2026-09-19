package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cuetiy-backend/config"
)

// ASRService MiMo-V2.5-ASR：音频 → 文本
type ASRService struct{}

func NewASRService() *ASRService {
	return &ASRService{}
}

type asrRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content []struct {
			Type       string `json:"type"`
			InputAudio struct {
				Data string `json:"data"`
			} `json:"input_audio"`
		} `json:"content"`
	} `json:"messages"`
	ASROptions struct {
		Language string `json:"language"`
	} `json:"asr_options"`
}

type asrResponse struct {
	Choices []struct {
		Message struct {
			Content interface{} `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Transcribe 音频字节 → 文本
func (s *ASRService) Transcribe(data []byte, mime string, language string) (string, error) {
	cfg := config.AppConfig
	if cfg == nil || cfg.MIMOAPIKey == "" {
		return "", fmt.Errorf("MIMO_API_KEY 未配置")
	}
	if len(data) == 0 {
		return "", fmt.Errorf("音频为空")
	}
	if len(data) > 10*1024*1024 {
		return "", fmt.Errorf("音频过大（base64 前需 <10MB）")
	}

	if mime == "" {
		mime = "audio/wav"
	}
	if !strings.HasPrefix(mime, "audio/") {
		mime = "audio/wav"
	}
	if language == "" {
		language = "zh"
	}

	b64 := base64.StdEncoding.EncodeToString(data)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mime, b64)

	var req asrRequest
	req.Model = "mimo-v2.5-asr"
	req.ASROptions.Language = language
	msg := struct {
		Role    string `json:"role"`
		Content []struct {
			Type       string `json:"type"`
			InputAudio struct {
				Data string `json:"data"`
			} `json:"input_audio"`
		} `json:"content"`
	}{Role: "user"}
	item := struct {
		Type       string `json:"type"`
		InputAudio struct {
			Data string `json:"data"`
		} `json:"input_audio"`
	}{Type: "input_audio"}
	item.InputAudio.Data = dataURL
	msg.Content = append(msg.Content, item)
	req.Messages = append(req.Messages, msg)

	payload, _ := json.Marshal(req)
	base := strings.TrimRight(cfg.MIMOAPIBase, "/")
	if base == "" {
		base = "https://api.xiaomimimo.com/v1"
	}
	httpReq, err := http.NewRequest("POST", base+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.MIMOAPIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("ASR 请求失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != 200 {
		snippet := string(raw)
		if len(snippet) > 400 {
			snippet = snippet[:400]
		}
		return "", fmt.Errorf("ASR API [%d]: %s", resp.StatusCode, snippet)
	}

	var ar asrResponse
	if err := json.Unmarshal(raw, &ar); err != nil {
		return "", fmt.Errorf("ASR 响应解析失败: %w", err)
	}
	if ar.Error != nil && ar.Error.Message != "" {
		return "", fmt.Errorf("ASR 错误: %s", ar.Error.Message)
	}
	if len(ar.Choices) == 0 {
		return "", fmt.Errorf("ASR 空结果")
	}

	return contentToString(ar.Choices[0].Message.Content), nil
}

func contentToString(c interface{}) string {
	switch v := c.(type) {
	case string:
		return strings.TrimSpace(v)
	case []interface{}:
		var b strings.Builder
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if t, ok := m["text"].(string); ok {
					b.WriteString(t)
				}
			}
		}
		return strings.TrimSpace(b.String())
	default:
		b, _ := json.Marshal(c)
		return strings.TrimSpace(string(b))
	}
}

// LoadUserVoiceData 读用户音色样本文件
func LoadUserVoiceData(storagePath string) (data []byte, mime string, err error) {
	if storagePath == "" {
		return nil, "", fmt.Errorf("无音色样本")
	}
	root := "./data/files"
	if config.AppConfig != nil && config.AppConfig.StorageDir != "" {
		root = config.AppConfig.StorageDir
	}
	full := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(storagePath, "/")))
	data, err = readFileLimited(full, 10*1024*1024)
	if err != nil {
		return nil, "", err
	}
	ext := strings.ToLower(filepath.Ext(storagePath))
	switch ext {
	case ".mp3":
		mime = "audio/mpeg"
	default:
		mime = "audio/wav"
	}
	return data, mime, nil
}

func readFileLimited(path string, max int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, max))
}
