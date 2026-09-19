package repository

import (
	"cuetiy-backend/config"
	"cuetiy-backend/model"
)

type ContextRepo struct{}

func NewContextRepo() *ContextRepo {
	return &ContextRepo{}
}

// --- Summary ---

func (r *ContextRepo) GetSummary(convID int64) (*model.ConversationSummary, error) {
	var s model.ConversationSummary
	err := config.DB.Where("conversation_id = ?", convID).
		Order("version DESC").First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ContextRepo) SaveSummary(s *model.ConversationSummary) error {
	return config.DB.Save(s).Error
}

func (r *ContextRepo) DeleteSummary(convID int64) error {
	return config.DB.Where("conversation_id = ?", convID).
		Delete(&model.ConversationSummary{}).Error
}

// --- Memory ---

func (r *ContextRepo) GetMemory(convID int64) (*model.ConversationMemory, error) {
	var m model.ConversationMemory
	err := config.DB.Where("conversation_id = ?", convID).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *ContextRepo) SaveMemory(m *model.ConversationMemory) error {
	return config.DB.Save(m).Error
}

// DeleteMemory 清记忆卡（不清 Skills）
func (r *ContextRepo) DeleteMemory(convID int64) error {
	return config.DB.Where("conversation_id = ?", convID).
		Delete(&model.ConversationMemory{}).Error
}

// --- Skill state ---

func (r *ContextRepo) GetSkillState(convID int64) (*model.ConversationSkillState, error) {
	var s model.ConversationSkillState
	err := config.DB.Where("conversation_id = ?", convID).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ContextRepo) SaveSkillState(s *model.ConversationSkillState) error {
	return config.DB.Save(s).Error
}

func (r *ContextRepo) ResetSkillState(convID int64) error {
	return config.DB.Where("conversation_id = ?", convID).
		Delete(&model.ConversationSkillState{}).Error
}

// --- Archive ---

func (r *ContextRepo) CreateArchive(a *model.ChatArchive) error {
	return config.DB.Create(a).Error
}

// --- Messages for assembler ---

func (r *ContextRepo) GetMessagesAfterID(convID int64, afterID int64, limit int) ([]model.Message, error) {
	if limit <= 0 {
		limit = 200
	}
	q := config.DB.Where("conversation_id = ? AND is_deleted = ?", convID, false)
	if afterID > 0 {
		q = q.Where("id > ?", afterID)
	}
	var msgs []model.Message
	err := q.Order("created_at ASC, id ASC").Limit(limit).Find(&msgs).Error
	return msgs, err
}

func (r *ContextRepo) GetOlderMessages(convID int64, maxID int64, limit int) ([]model.Message, error) {
	if limit <= 0 {
		limit = 80
	}
	var msgs []model.Message
	err := config.DB.Where("conversation_id = ? AND is_deleted = ? AND id <= ?", convID, false, maxID).
		Order("created_at ASC, id ASC").Limit(limit).Find(&msgs).Error
	return msgs, err
}
