package repository

import (
	"cuetiy-backend/config"
	"cuetiy-backend/model"
)

type MessageRepository struct{}

func NewMessageRepository() *MessageRepository {
	return &MessageRepository{}
}

// whereActive 活跃消息：兼容 PG 中 is_deleted 为 NULL 的历史行
func whereActive() string {
	return "(is_deleted IS NULL OR is_deleted = ?)"
}

func (r *MessageRepository) Create(msg *model.Message) error {
	return config.DB.Create(msg).Error
}

// FindOlderThan 取比 beforeID 更旧的消息（分页上翻）。
// beforeID<=0 时等价于取最新一页。
func (r *MessageRepository) FindOlderThan(convID int64, beforeID int64, limit int) ([]model.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	q := config.DB.Where("conversation_id = ? AND "+whereActive(), convID, false)
	if beforeID > 0 {
		q = q.Where("id < ?", beforeID)
	}

	var messages []model.Message
	err := q.Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&messages).Error
	if err != nil {
		return nil, err
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

// FindByConversationID 兼容旧 offset 接口，内部转发到 FindOlderThan。
func (r *MessageRepository) FindByConversationID(convID int64, limit, offset int) ([]model.Message, error) {
	if offset == 0 {
		return r.FindOlderThan(convID, 0, limit)
	}
	// offset 分页：先取 offset+limit 条再截断
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var messages []model.Message
	err := config.DB.Where("conversation_id = ? AND "+whereActive(), convID, false).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	if err != nil {
		return nil, err
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

func (r *MessageRepository) GetRecentMessages(convID int64, limit int) ([]model.Message, error) {
	if limit <= 0 {
		limit = 50
	}

	var messages []model.Message
	err := config.DB.Where("conversation_id = ? AND "+whereActive(), convID, false).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error
	if err != nil {
		return nil, err
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

func (r *MessageRepository) SoftDeleteByConversationID(convID int64) error {
	return config.DB.Model(&model.Message{}).
		Where("conversation_id = ?", convID).
		Update("is_deleted", true).Error
}

// HardDeleteByConversationID 归档后物理删除该会话消息，保证下一轮 history 为空
func (r *MessageRepository) HardDeleteByConversationID(convID int64) error {
	return config.DB.Where("conversation_id = ?", convID).
		Delete(&model.Message{}).Error
}

func (r *MessageRepository) CountByConversationID(convID int64) (int64, error) {
	var count int64
	err := config.DB.Model(&model.Message{}).
		Where("conversation_id = ? AND "+whereActive(), convID, false).
		Count(&count).Error
	return count, err
}

func (r *MessageRepository) GetLastMessage(convID int64) (*model.Message, error) {
	var msg model.Message
	err := config.DB.Where("conversation_id = ? AND "+whereActive(), convID, false).
		Order("created_at DESC").
		First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// FindAllByConversationID 取会话全部有效消息（归档用）
func (r *MessageRepository) FindAllByConversationID(convID int64) ([]model.Message, error) {
	var messages []model.Message
	err := config.DB.Where("conversation_id = ? AND "+whereActive(), convID, false).
		Order("created_at ASC, id ASC").
		Find(&messages).Error
	return messages, err
}
