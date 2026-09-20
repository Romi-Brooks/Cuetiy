package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func newDeepSeekRequest(url string, body []byte, apiKey string) (*http.Request, error) {
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	return req, nil
}

func deepSeekClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}

func readAllLimited(r io.Reader, n int64) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r, n))
}

// marshalChatBody 统一 chat completions body
func marshalChatBody(model string, messages []ChatMessage, temperature float64, maxTokens int, stream bool) []byte {
	body := map[string]any{
		"model":       model,
		"messages":    messages,
		"stream":      stream,
		"temperature": temperature,
		"max_tokens":  maxTokens,
	}
	data, _ := json.Marshal(body)
	return data
}
