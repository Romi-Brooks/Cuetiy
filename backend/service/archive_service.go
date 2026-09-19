package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"rain-yi-backend/config"
	"rain-yi-backend/model"
	"rain-yi-backend/repository"
)

// ArchiveService 清空聊天前：把消息+记忆卡导出到文件，便于溯源
type ArchiveService struct {
	msgRepo *repository.MessageRepository
	ctxRepo *repository.ContextRepo
}

func NewArchiveService(msgRepo *repository.MessageRepository, ctxRepo *repository.ContextRepo) *ArchiveService {
	return &ArchiveService{msgRepo: msgRepo, ctxRepo: ctxRepo}
}

type archivePayload struct {
	ConversationID int64                       `json:"conversation_id"`
	UserID         int64                       `json:"user_id"`
	ArchivedAt     time.Time                   `json:"archived_at"`
	Note           string                      `json:"note"`
	Messages       []model.Message             `json:"messages"`
	Memory         *model.ConversationMemory   `json:"memory,omitempty"`
	Summary        *model.ConversationSummary  `json:"summary,omitempty"`
}

// ArchiveConversation 导出消息与记忆卡到文件；Skills 不在此范围（属人格）
func (s *ArchiveService) ArchiveConversation(conv *model.Conversation) (string, int, error) {
	msgs, err := s.msgRepo.FindAllByConversationID(conv.ID)
	if err != nil {
		return "", 0, err
	}

	payload := archivePayload{
		ConversationID: conv.ID,
		UserID:         conv.UserID,
		ArchivedAt:     time.Now(),
		Note:           "清空聊天记录前自动归档；Skills/人格文件不受影响",
		Messages:       msgs,
	}
	if mem, err := s.ctxRepo.GetMemory(conv.ID); err == nil {
		payload.Memory = mem
	}
	if sum, err := s.ctxRepo.GetSummary(conv.ID); err == nil {
		payload.Summary = sum
	}

	dir := "./data/archives"
	if config.AppConfig != nil && config.AppConfig.ArchiveDir != "" {
		dir = config.AppConfig.ArchiveDir
	} else if config.AppConfig != nil && config.AppConfig.StorageDir != "" {
		dir = filepath.Join(filepath.Dir(config.AppConfig.StorageDir), "archives")
	}
	convDir := filepath.Join(dir, fmt.Sprintf("conv_%d", conv.ID))
	if err := os.MkdirAll(convDir, 0o755); err != nil {
		return "", 0, err
	}
	name := fmt.Sprintf("chat_%s.json", time.Now().Format("20060102_150405"))
	path := filepath.Join(convDir, name)

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", 0, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", 0, err
	}

	_ = s.ctxRepo.CreateArchive(&model.ChatArchive{
		ConversationID: conv.ID,
		UserID:         conv.UserID,
		FilePath:       path,
		MessageCount:   len(msgs),
	})

	return path, len(msgs), nil
}
