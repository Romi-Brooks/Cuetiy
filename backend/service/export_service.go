package service

import (
	"fmt"
	"sort"
	"time"

	"rain-yi-backend/model"
	"rain-yi-backend/repository"
)

const (
	ChatExportFormat  = "rainyi-chat-export"
	ChatExportVersion = 1
)

type ChatExportConversation struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	AINickname  string    `json:"ai_nickname"`
	AIAvatar    string    `json:"ai_avatar"`
	PersonaID   *int64    `json:"persona_id,omitempty"`
	PersonaName string    `json:"persona_name,omitempty"`
	PersonaDir  string    `json:"persona_dir,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ChatExportPayload struct {
	Format       string                        `json:"format"`
	Version      int                           `json:"version"`
	ExportedAt   time.Time                     `json:"exported_at"`
	Note         string                        `json:"note,omitempty"`
	// 兼容清空时的旧归档字段
	ConversationID int64                       `json:"conversation_id,omitempty"`
	UserID         int64                       `json:"user_id,omitempty"`
	ArchivedAt     *time.Time                  `json:"archived_at,omitempty"`
	Conversation   ChatExportConversation      `json:"conversation"`
	Messages       []model.Message             `json:"messages"`
	Memory         *model.ConversationMemory   `json:"memory,omitempty"`
	Summary        *model.ConversationSummary  `json:"summary,omitempty"`
	SkillState     *model.ConversationSkillState `json:"skill_state,omitempty"`
}

type ChatExportBundle struct {
	Format        string              `json:"format"`
	Version       int                 `json:"version"`
	ExportedAt    time.Time           `json:"exported_at"`
	Note          string              `json:"note,omitempty"`
	Conversations []ChatExportPayload `json:"conversations"`
}

type ExportService struct {
	convRepo    *repository.ConversationRepository
	msgRepo     *repository.MessageRepository
	ctxRepo     *repository.ContextRepo
	personaRepo *repository.PersonaRepository
}

func NewExportService(
	convRepo *repository.ConversationRepository,
	msgRepo *repository.MessageRepository,
	ctxRepo *repository.ContextRepo,
	personaRepo *repository.PersonaRepository,
) *ExportService {
	return &ExportService{
		convRepo:    convRepo,
		msgRepo:     msgRepo,
		ctxRepo:     ctxRepo,
		personaRepo: personaRepo,
	}
}

func (s *ExportService) ExportConversation(conv *model.Conversation) (*ChatExportPayload, error) {
	if conv == nil {
		return nil, fmt.Errorf("conversation is nil")
	}
	msgs, err := s.msgRepo.FindAllByConversationID(conv.ID)
	if err != nil {
		return nil, err
	}
	if msgs == nil {
		msgs = []model.Message{}
	}

	meta := ChatExportConversation{
		ID:         conv.ID,
		Title:      conv.Title,
		AINickname: conv.AINickname,
		AIAvatar:   conv.AIAvatar,
		PersonaID:  conv.PersonaID,
		CreatedAt:  conv.CreatedAt,
		UpdatedAt:  conv.UpdatedAt,
	}
	if conv.PersonaID != nil && s.personaRepo != nil {
		if p, err := s.personaRepo.FindByID(*conv.PersonaID); err == nil && p != nil {
			meta.PersonaName = p.Name
			meta.PersonaDir = p.DirName
		}
	}

	payload := &ChatExportPayload{
		Format:         ChatExportFormat,
		Version:        ChatExportVersion,
		ExportedAt:     time.Now(),
		Note:           "聊天记录导出；附件 URL 可能随服务器变化，导入以文本为准",
		ConversationID: conv.ID,
		UserID:         conv.UserID,
		Conversation:   meta,
		Messages:       msgs,
	}
	if s.ctxRepo != nil {
		if mem, err := s.ctxRepo.GetMemory(conv.ID); err == nil && mem != nil {
			payload.Memory = mem
		}
		if sum, err := s.ctxRepo.GetSummary(conv.ID); err == nil && sum != nil {
			payload.Summary = sum
		}
		if st, err := s.ctxRepo.GetSkillState(conv.ID); err == nil && st != nil {
			payload.SkillState = st
		}
	}
	return payload, nil
}

func (s *ExportService) ExportConversationByID(convID int64) (*ChatExportPayload, error) {
	conv, err := s.convRepo.FindByID(convID)
	if err != nil || conv == nil {
		return nil, fmt.Errorf("会话不存在")
	}
	return s.ExportConversation(conv)
}

func (s *ExportService) ExportUser(userID int64) (*ChatExportBundle, error) {
	convs, err := s.convRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}
	list := make([]ChatExportPayload, 0, len(convs))
	for i := range convs {
		p, err := s.ExportConversation(&convs[i])
		if err != nil {
			continue
		}
		list = append(list, *p)
	}
	return &ChatExportBundle{
		Format:        ChatExportFormat,
		Version:       ChatExportVersion,
		ExportedAt:    time.Now(),
		Note:          "当前用户全部会话导出",
		Conversations: list,
	}, nil
}

type ImportResult struct {
	ConversationID int64  `json:"conversation_id"`
	Title          string `json:"title"`
	MessageCount   int    `json:"message_count"`
	PersonaLinked  bool   `json:"persona_linked"`
	MemoryRestored bool   `json:"memory_restored"`
}

type ImportService struct {
	convRepo    *repository.ConversationRepository
	msgRepo     *repository.MessageRepository
	ctxRepo     *repository.ContextRepo
	personaRepo *repository.PersonaRepository
}

func NewImportService(
	convRepo *repository.ConversationRepository,
	msgRepo *repository.MessageRepository,
	ctxRepo *repository.ContextRepo,
	personaRepo *repository.PersonaRepository,
) *ImportService {
	return &ImportService{
		convRepo:    convRepo,
		msgRepo:     msgRepo,
		ctxRepo:     ctxRepo,
		personaRepo: personaRepo,
	}
}

// ImportOne 将一份导出包导入为当前用户下的新会话（消息 ID 重映射）
func (s *ImportService) ImportOne(userID int64, payload *ChatExportPayload) (*ImportResult, error) {
	if payload == nil {
		return nil, fmt.Errorf("empty payload")
	}

	title := payload.Conversation.Title
	if title == "" && payload.ConversationID > 0 {
		title = fmt.Sprintf("导入会话 #%d", payload.ConversationID)
	}
	if title == "" {
		title = "导入的对话"
	}
	nickname := payload.Conversation.AINickname
	if nickname == "" {
		nickname = "RainYi"
	}

	conv := &model.Conversation{
		UserID:     userID,
		Title:      title,
		AINickname: nickname,
		AIAvatar:   payload.Conversation.AIAvatar,
	}

	linked := false
	if s.personaRepo != nil {
		var persona *model.Persona
		if payload.Conversation.PersonaDir != "" {
			if p, err := s.personaRepo.FindByDirName(payload.Conversation.PersonaDir); err == nil {
				persona = p
			}
		}
		if persona == nil && payload.Conversation.PersonaName != "" {
			if p, err := s.personaRepo.FindByName(payload.Conversation.PersonaName); err == nil {
				persona = p
			}
		}
		if persona != nil {
			pid := persona.ID
			conv.PersonaID = &pid
			linked = true
		}
	}

	if err := s.convRepo.Create(conv); err != nil {
		return nil, err
	}

	msgs := append([]model.Message(nil), payload.Messages...)
	sort.SliceStable(msgs, func(i, j int) bool {
		if !msgs[i].CreatedAt.Equal(msgs[j].CreatedAt) {
			return msgs[i].CreatedAt.Before(msgs[j].CreatedAt)
		}
		return msgs[i].ID < msgs[j].ID
	})

	count := 0
	for _, m := range msgs {
		role := m.Role
		if role != "user" && role != "assistant" && role != "system" {
			role = "user"
		}
		msgType := m.MessageType
		if msgType == "" {
			msgType = "text"
		}
		content := m.Content
		if content == "" {
			continue
		}
		row := &model.Message{
			ConversationID:  conv.ID,
			Role:            role,
			MessageType:     msgType,
			Content:         content,
			AudioURL:        m.AudioURL,
			AudioDurationMs: m.AudioDurationMs,
			HasAttachment:   m.HasAttachment,
			AttachmentType:  m.AttachmentType,
			AttachmentURL:   m.AttachmentURL,
			CreatedAt:       m.CreatedAt,
		}
		if row.CreatedAt.IsZero() {
			row.CreatedAt = time.Now()
		}
		if err := s.msgRepo.Create(row); err != nil {
			continue
		}
		count++
	}

	memRestored := false
	if s.ctxRepo != nil && payload.Memory != nil && payload.Memory.Content != "" {
		mem := &model.ConversationMemory{
			ConversationID: conv.ID,
			UserID:         userID,
			Content:        payload.Memory.Content,
			TokenEstimate:  payload.Memory.TokenEstimate,
		}
		if err := s.ctxRepo.SaveMemory(mem); err == nil {
			memRestored = true
		}
	}
	if s.ctxRepo != nil && payload.Summary != nil && payload.Summary.Content != "" {
		_ = s.ctxRepo.SaveSummary(&model.ConversationSummary{
			ConversationID: conv.ID,
			Content:        payload.Summary.Content,
			Version:        1,
			TokenEstimate:  payload.Summary.TokenEstimate,
		})
	}
	if s.ctxRepo != nil && payload.SkillState != nil {
		_ = s.ctxRepo.SaveSkillState(&model.ConversationSkillState{
			ConversationID: conv.ID,
			ActiveSkills:   payload.SkillState.ActiveSkills,
			TurnCounter:    payload.SkillState.TurnCounter,
			LastEmotion:    payload.SkillState.LastEmotion,
		})
	}

	return &ImportResult{
		ConversationID: conv.ID,
		Title:          conv.Title,
		MessageCount:   count,
		PersonaLinked:  linked,
		MemoryRestored: memRestored,
	}, nil
}

func (s *ImportService) ImportBundle(userID int64, bundle *ChatExportBundle) ([]ImportResult, error) {
	if bundle == nil || len(bundle.Conversations) == 0 {
		return nil, fmt.Errorf("bundle 为空")
	}
	out := make([]ImportResult, 0, len(bundle.Conversations))
	for i := range bundle.Conversations {
		r, err := s.ImportOne(userID, &bundle.Conversations[i])
		if err != nil {
			continue
		}
		out = append(out, *r)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("没有成功导入任何会话")
	}
	return out, nil
}
