package controller

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"cuetiy-backend/config"
	"cuetiy-backend/middleware"
	"cuetiy-backend/model"
	"cuetiy-backend/repository"
	"cuetiy-backend/service"
	"cuetiy-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

type ChatController struct {
	msgRepo        *repository.MessageRepository
	convRepo       *repository.ConversationRepository
	aiService      *service.AIService
	contextManager *service.ContextManager
	assembler      *service.ContextAssembler
	msgWriter      *service.MessageWriter
	tts            *service.TTSService
	imageGen       *service.ImageGenService
	hub            *service.WebSocketHub
	upgrader       websocket.Upgrader

	// 语音条冷却：convID -> 距上次语音条的文字条数
	voiceCooldown map[int64]int
}

func NewChatController(
	msgRepo *repository.MessageRepository,
	convRepo *repository.ConversationRepository,
	aiService *service.AIService,
	contextManager *service.ContextManager,
	assembler *service.ContextAssembler,
	msgWriter *service.MessageWriter,
	tts *service.TTSService,
	hub *service.WebSocketHub,
	imageGen *service.ImageGenService,
) *ChatController {
	return &ChatController{
		msgRepo:        msgRepo,
		convRepo:       convRepo,
		aiService:      aiService,
		contextManager: contextManager,
		assembler:      assembler,
		msgWriter:      msgWriter,
		tts:            tts,
		imageGen:       imageGen,
		hub:            hub,
		voiceCooldown:  make(map[int64]int),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

type wsIncomingMessage struct {
	ConversationID int64  `json:"conversation_id"`
	Content        string `json:"content"`
	// WantVoice 用户点名「想听语音」：本轮 AI 回复强制走语音条
	WantVoice bool `json:"want_voice"`
}

func (ctl *ChatController) HandleWebSocket(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		authHeader := c.GetHeader("Authorization")
		tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
	}

	if tokenStr == "" || tokenStr == c.GetHeader("Authorization") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
		return
	}

	claims := &middleware.Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.AppConfig.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "令牌无效"})
		return
	}

	if config.RDB != nil && claims.ID != "" {
		blacklisted, blErr := config.RDB.Exists(config.RedisCtx, fmt.Sprintf("auth:blacklist:%s", claims.ID)).Result()
		if blErr == nil && blacklisted == 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "令牌已被吊销"})
			return
		}
	}

	conn, err := ctl.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &service.Client{
		ID:     claims.Email,
		UserID: claims.UserID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		PublicBase: func() string {
			scheme := "http"
			if c.Request.TLS != nil {
				scheme = "https"
			}
			if p := c.Request.Header.Get("X-Forwarded-Proto"); p != "" {
				scheme = strings.ToLower(strings.TrimSpace(p))
			}
			if c.Request.Host != "" {
				return scheme + "://" + c.Request.Host
			}
			return ""
		}(),
	}

	ctl.hub.Register(client)

	go ctl.readPump(client)
	go ctl.writePump(client)
}

func (ctl *ChatController) readPump(client *service.Client) {
	defer func() {
		ctl.hub.Unregister(client)
		client.Conn.Close()
	}()

	client.Conn.SetReadLimit(64 * 1024)
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg wsIncomingMessage
		err := client.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		content := utils.SanitizeContent(msg.Content)
		userID := client.UserID

		var conv *model.Conversation
		var findErr error
		if msg.ConversationID > 0 {
			conv, findErr = ctl.convRepo.FindByID(msg.ConversationID)
			if findErr != nil || conv.UserID != userID {
				log.Printf("Conversation not found or not owned: %d", msg.ConversationID)
				ctl.hub.SendError(userID, "会话不存在")
				continue
			}
		} else {
			conv, findErr = ctl.convRepo.FindOrCreateDefault(userID)
			if findErr != nil {
				log.Printf("FindOrCreateDefault error: %v", findErr)
				ctl.hub.SendError(userID, "会话创建失败")
				continue
			}
		}

		// 分层组装：L0/L1/L2 + 记忆 + 摘要 + 近期原文（token 预算 / 55% 压缩）
		assembled, aerr := ctl.assembler.Assemble(conv, content)
		if aerr != nil {
			log.Printf("AssembleContext error: %v", aerr)
			ctl.hub.SendError(userID, "上下文构建失败")
			continue
		}
		log.Printf("[ctx] conv=%d emotion=%s skills=%v tokens=%d sys=%d recent=%d compacted=%v ver=%d",
			conv.ID, assembled.Emotion, assembled.ActivatedSkills,
			assembled.EstimatedTokens, assembled.SystemTokens, assembled.RecentTokens,
			assembled.Compacted, assembled.SummaryVersion)

		if assembled.Debug != nil {
			ctl.hub.SendContextDebug(userID, conv.ID, assembled.Debug)
		}
		ctxDebug := interface{}(nil)
		if assembled.Debug != nil {
			ctxDebug = assembled.Debug
		}

		// 本地先展示/组装；MySQL 落库放到本轮结束后异步执行
		userMessage := &model.Message{
			ConversationID: conv.ID,
			Role:           "user",
			Content:        content,
			CreatedAt:      time.Now(),
		}

		ctl.hub.BroadcastStarting(userID)

		aiResponse, err := ctl.aiService.SendMessageWithMessages(assembled.Messages, func(chunk string) {
			ctl.hub.SendStreamChunk(userID, chunk)
		})
		if err != nil {
			log.Printf("AI SendMessage error: %v", err)
			ctl.hub.SendError(userID, "AI 响应失败: "+err.Error())
			ctl.msgWriter.EnqueueAsync(userMessage)
			continue
		}

		aiMessage := &model.Message{
			ConversationID: conv.ID,
			Role:           "assistant",
			Content:        aiResponse,
			MessageType:    "text",
			CreatedAt:      time.Now(),
		}

		// 完整回复 → 显式小段（本期只做切分展示，不做流式打断）
		segments := service.SplitReplySegments(aiResponse)

		// 语音条策略：仅用户点名 want_voice 时合成
		if ctl.tts != nil && ctl.shouldSendVoice(conv.ID, msg.WantVoice) {
			url, _, ttsDbg, terr := ctl.tts.Synthesize(userID, aiResponse, "", assembled.Emotion)
			if ttsDbg != nil {
				ctl.hub.SendTTSDebug(userID, conv.ID, ttsDbg)
			}
			if terr == nil && url != "" {
				absAudio := url
				if client.PublicBase != "" && strings.HasPrefix(url, "/") {
					absAudio = strings.TrimRight(client.PublicBase, "/") + url
				}
				aiMessage.MessageType = "voice"
				aiMessage.AudioURL = absAudio
				if ttsDbg != nil && ttsDbg.DurationMs > 0 {
					aiMessage.AudioDurationMs = ttsDbg.DurationMs
				} else {
					aiMessage.AudioDurationMs = estimateVoiceMs(aiResponse)
				}
				ctl.voiceCooldown[conv.ID] = 0
				var ttsAny interface{}
				if ttsDbg != nil {
					ttsAny = ttsDbg
				}
				// complete 内嵌 tts+ctx，前端按 audio_url 落库，不再依赖临时 id
				ctl.hub.SendCompleteVoice(userID, conv.ID, aiResponse, absAudio, aiMessage.AudioDurationMs, ttsAny, ctxDebug, segments)
			} else {
				if terr != nil {
					log.Printf("[tts] voice bar failed: %v", terr)
				}
				ctl.noteTextTurn(conv.ID)
				ctl.hub.SendCompleteWithDebug(userID, conv.ID, aiResponse, "text", "", 0, 0, nil, ctxDebug, segments)
			}
		} else {
			ctl.noteTextTurn(conv.ID)
			ctl.hub.SendCompleteWithDebug(userID, conv.ID, aiResponse, "text", "", 0, 0, nil, ctxDebug, segments)
		}

		// 出图遵循 skills：触发词/标签来自 registry，外貌 prompt 也来自技能包
		ctl.maybeSendImage(userID, client, conv.ID, conv.PersonaID, content, aiResponse, assembled.Emotion)

		ctl.msgWriter.EnqueueAsync(userMessage, aiMessage)
		go func(convID int64, u, a *model.Message) {
			ctl.contextManager.AppendToContext(convID, u)
			ctl.contextManager.AppendToContext(convID, a)
			if c, e := ctl.convRepo.FindByID(convID); e == nil {
				_ = ctl.convRepo.Update(c)
			}
		}(conv.ID, userMessage, aiMessage)
	}
}

// maybeSendImage 出图触发与外貌均来自技能包（registry）；引擎只做开关与限流
func (ctl *ChatController) maybeSendImage(userID int64, client *service.Client, convID int64, personaID *int64, userMsg, aiReply, emotion string) {
	if ctl.imageGen == nil || ctl.hub == nil {
		return
	}
	cfg := config.AppConfig
	if cfg == nil || !cfg.ImageGenEnabled || strings.TrimSpace(cfg.GRSAIAPIKey) == "" {
		return
	}
	pid := int64(0)
	if personaID != nil {
		pid = *personaID
	}
	var tags []string
	// 有 registry 时：keywords 或 want_image 标签触发
	triggered, src := ctl.imageGen.ShouldTrigger(userMsg, pid, tags)
	if !triggered {
		log.Printf("[image] not triggered by skills user=%d persona=%d src=%s msg=%q", userID, pid, src, userMsg)
		return
	}
	// 限流：max/window <=0 表示暂时关闭（当前默认关闭 30 分钟窗）
	hits := 0
	limitEnabled := cfg.ImageGenLimitMax > 0 && cfg.ImageGenLimitWindowMin > 0
	if limitEnabled {
		window := time.Duration(cfg.ImageGenLimitWindowMin) * time.Minute
		ok, h := ctl.imageGen.AllowImage(userID, cfg.ImageGenLimitMax, window)
		hits = h
		if !ok {
			log.Printf("[image] rate limit user=%d hits=%d/%d", userID, hits, cfg.ImageGenLimitMax)
			return
		}
	}

	// 先取 LLM 场景提示词 + 等待句（失败回退 skills-only + 固定短句）
	llmResult := ctl.imageGen.MaybeLLMPromptWithPersona(userMsg, aiReply, emotion, pid)
	waitPhrase := service.WaitPhraseFromLLM(llmResult)

	earlyDbg := &service.ImageGenDebug{
		Enabled:    true,
		Triggered:  true,
		TriggerSrc: src,
		MaxPerWin:  cfg.ImageGenLimitMax,
		HitsInWin:  hits,
		Model:      cfg.ImageGenModel,
		APIBase:    cfg.GRSAIAPIBase,
		Status:     "generating",
	}
	if llmResult != nil {
		earlyDbg.LLMPrompt = llmResult.Prompt
		earlyDbg.LLMCaption = llmResult.Caption
		earlyDbg.LLMScene = llmResult.SceneType
		earlyDbg.LLMHasPerson = llmResult.HasPerson
	}
	ctl.hub.SendImageGenerating(userID, convID, waitPhrase, earlyDbg)
	log.Printf("[image] generating start user=%d src=%s limit=%v phrase=%q scene=%s",
		userID, src, limitEnabled, waitPhrase, earlyDbg.LLMScene)

	imgURL, dbg, err := ctl.imageGen.Generate(userID, pid, userMsg, aiReply, emotion, tags, llmResult)
	if err != nil || imgURL == "" {
		log.Printf("[image] generate failed user=%d: %v dbg=%+v", userID, err, dbg)
		return
	}
	abs := imgURL
	if client != nil && client.PublicBase != "" && strings.HasPrefix(imgURL, "/") {
		abs = service.AbsImageURL(client.PublicBase, imgURL)
	}

	imgMsg := &model.Message{
		ConversationID: convID,
		Role:           "assistant",
		Content:        "",
		MessageType:    "image",
		HasAttachment:  true,
		AttachmentType: "image",
		AttachmentURL:  abs,
		CreatedAt:      time.Now(),
	}
	msgID := time.Now().UnixMilli()
	imgMsg.ID = msgID
	if dbg != nil {
		dbg.Triggered = true
		dbg.TriggerSrc = src
	}
	ctl.hub.SendImageMessage(userID, convID, msgID, "", abs, dbg)
	ctl.msgWriter.EnqueueAsync(imgMsg)
	log.Printf("[image] sent user=%d conv=%d src=%s url=%s model=%s ref=%v appear=%v",
		userID, convID, src, abs, dbg.Model, dbg.HasRefImage, dbg.Appearance != "")
}

func pickImageWaitPhrase() string {
	return service.WaitPhraseFromLLM(nil)
}

func (ctl *ChatController) noteTextTurn(convID int64) {
	ctl.voiceCooldown[convID]++
}

func (ctl *ChatController) shouldSendVoice(convID int64, want bool) bool {
	// 仅用户点名要语音时才发语音条；AI 自选策略后续再开
	if !want {
		return false
	}
	cfg := config.AppConfig
	if cfg == nil || !cfg.TTSEnabled || cfg.MIMOAPIKey == "" {
		return false
	}
	return true
}

// estimateVoiceMs 粗估时长：中文约 4.5 字/秒
func estimateVoiceMs(text string) int64 {
	n := len([]rune(service.StripForTTS(text)))
	if n == 0 {
		return 1000
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

func (ctl *ChatController) writePump(client *service.Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
