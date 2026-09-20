package service

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID     string
	Conn   *websocket.Conn
	Send   chan []byte
	UserID int64
	// PublicBase 对外基址（http://host:port），用于把相对资源路径转成绝对 URL
	PublicBase string
}

type WebSocketHub struct {
	clients map[int64]*Client
	mu      chan struct{}
}

type wsMessage struct {
	Type           string      `json:"type"`
	Content        string      `json:"content,omitempty"`
	MessageID      int64       `json:"message_id,omitempty"`
	ConversationID int64       `json:"conversation_id,omitempty"`
	Debug          interface{} `json:"debug,omitempty"`
	// complete 时一并携带的调试载荷（避免前端用临时 id 对不上）
	TTSDebug   interface{} `json:"tts_debug,omitempty"`
	ContextDebug interface{} `json:"context_debug,omitempty"`
	// 语音条
	MessageType     string `json:"message_type,omitempty"`
	AudioURL        string `json:"audio_url,omitempty"`
	AudioDurationMs int64  `json:"audio_duration_ms,omitempty"`
	// 显式段分割：complete 时携带，前端按段渲染
	Segments []ReplySegment `json:"segments,omitempty"`
	// 图片消息（AI 自拍等）
	ImageURL   string      `json:"image_url,omitempty"`
	ImageDebug interface{} `json:"image_debug,omitempty"`
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients: make(map[int64]*Client),
		mu:      make(chan struct{}, 1),
	}
}

func (h *WebSocketHub) lock() {
	h.mu <- struct{}{}
}

func (h *WebSocketHub) unlock() {
	<-h.mu
}

// TODO: 分布式部署时 WebSocketHub 需迁移到 Redis Pub/Sub
// 1. 发消息时 publish 到 Redis channel "ws:notify:{userID}"
// 2. 每台机器订阅自己的 channel
// 3. 收到消息后查询本地 clients map 推送
// 4. 进程重启后需从 Redis 恢复心跳
func (h *WebSocketHub) Register(client *Client) {
	h.lock()
	defer h.unlock()
	if existing, ok := h.clients[client.UserID]; ok {
		close(existing.Send)
		existing.Conn.Close()
	}
	h.clients[client.UserID] = client
}

func (h *WebSocketHub) Unregister(client *Client) {
	h.lock()
	defer h.unlock()
	if existing, ok := h.clients[client.UserID]; ok && existing == client {
		delete(h.clients, client.UserID)
		close(client.Send)
	}
}

func (h *WebSocketHub) GetClient(userID int64) *Client {
	h.lock()
	defer h.unlock()
	return h.clients[userID]
}

func (h *WebSocketHub) SendToUser(userID int64, message interface{}) {
	client := h.GetClient(userID)
	if client == nil {
		return
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("WebSocket marshal message error: %v", err)
		return
	}

	select {
	case client.Send <- data:
	default:
		log.Printf("WebSocket send channel full for user %d, dropping message", userID)
	}
}

func (h *WebSocketHub) BroadcastStarting(userID int64) {
	h.SendToUser(userID, wsMessage{Type: "ai_start"})
}

func (h *WebSocketHub) SendStreamChunk(userID int64, content string) {
	h.SendToUser(userID, wsMessage{Type: "stream", Content: content})
}

func (h *WebSocketHub) SendComplete(userID int64, fullContent string) {
	h.SendToUser(userID, wsMessage{Type: "complete", Content: fullContent})
}

func (h *WebSocketHub) SendCompleteWithID(userID int64, fullContent string, messageID, conversationID int64) {
	h.SendToUser(userID, wsMessage{
		Type:           "complete",
		Content:        fullContent,
		MessageID:      messageID,
		ConversationID: conversationID,
	})
}

func (h *WebSocketHub) SendError(userID int64, errMsg string) {
	h.SendToUser(userID, wsMessage{Type: "error", Content: errMsg})
}

// SendCompleteWithDebug 推送完整回复（可带 TTS / 上下文 debug / 显式分段）
func (h *WebSocketHub) SendCompleteWithDebug(
	userID, conversationID int64,
	content, messageType, audioURL string,
	durationMs int64,
	messageID int64,
	ttsDebug, ctxDebug interface{},
	segments []ReplySegment,
) {
	h.SendToUser(userID, wsMessage{
		Type:            "complete",
		Content:         content,
		ConversationID:  conversationID,
		MessageID:       messageID,
		MessageType:     messageType,
		AudioURL:        audioURL,
		AudioDurationMs: durationMs,
		TTSDebug:        ttsDebug,
		ContextDebug:    ctxDebug,
		Segments:        segments,
	})
}

// SendCompleteVoice 推送语音条完整消息
func (h *WebSocketHub) SendCompleteVoice(userID, conversationID int64, text, audioURL string, durationMs int64, ttsDebug, ctxDebug interface{}, segments []ReplySegment) {
	h.SendCompleteWithDebug(userID, conversationID, text, "voice", audioURL, durationMs, 0, ttsDebug, ctxDebug, segments)
}

// SendTTSDebug 推送本轮给 MiMo TTS 的请求明细
func (h *WebSocketHub) SendTTSDebug(userID, conversationID int64, debug interface{}) {
	h.SendToUser(userID, wsMessage{
		Type:           "tts_debug",
		ConversationID: conversationID,
		Debug:          debug,
	})
}

// SendContextDebug 推送本轮上下文组装明细（点 AI 名字可看）
func (h *WebSocketHub) SendContextDebug(userID int64, conversationID int64, debug interface{}) {
	h.SendToUser(userID, wsMessage{
		Type:           "context_debug",
		ConversationID: conversationID,
		Debug:          debug,
	})
}

// SendImageMessage 文字 complete 之后追加一张 AI 图片（先一句话再发图）
func (h *WebSocketHub) SendImageMessage(userID, conversationID, messageID int64, caption, imageURL string, debug interface{}) {
	h.SendToUser(userID, wsMessage{
		Type:           "image_message",
		Content:        caption,
		ConversationID: conversationID,
		MessageID:      messageID,
		MessageType:    "image",
		ImageURL:       imageURL,
		ImageDebug:     debug,
	})
}
