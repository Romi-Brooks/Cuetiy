package repository

import (
	"cuetiy-backend/config"
	"cuetiy-backend/model"
)

type VoiceRepo struct{}

func NewVoiceRepo() *VoiceRepo {
	return &VoiceRepo{}
}

func (r *VoiceRepo) FindByUserID(userID int64) (*model.UserVoice, error) {
	var v model.UserVoice
	err := config.DB.Where("user_id = ?", userID).First(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *VoiceRepo) Upsert(v *model.UserVoice) error {
	if existing, err := r.FindByUserID(v.UserID); err == nil && existing != nil {
		v.ID = existing.ID
		v.CreatedAt = existing.CreatedAt
		return config.DB.Save(v).Error
	}
	return config.DB.Create(v).Error
}

func (r *VoiceRepo) DeleteByUserID(userID int64) error {
	return config.DB.Where("user_id = ?", userID).Delete(&model.UserVoice{}).Error
}
