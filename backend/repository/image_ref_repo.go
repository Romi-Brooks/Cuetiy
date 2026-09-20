package repository

import (
	"cuetiy-backend/config"
	"cuetiy-backend/model"
)

type ImageRefRepo struct{}

func NewImageRefRepo() *ImageRefRepo {
	return &ImageRefRepo{}
}

func (r *ImageRefRepo) FindByUserID(userID int64) (*model.UserImageRef, error) {
	var v model.UserImageRef
	err := config.DB.Where("user_id = ?", userID).First(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *ImageRefRepo) Upsert(v *model.UserImageRef) error {
	if existing, err := r.FindByUserID(v.UserID); err == nil && existing != nil {
		v.ID = existing.ID
		v.CreatedAt = existing.CreatedAt
		return config.DB.Save(v).Error
	}
	return config.DB.Create(v).Error
}

func (r *ImageRefRepo) DeleteByUserID(userID int64) error {
	return config.DB.Where("user_id = ?", userID).Delete(&model.UserImageRef{}).Error
}
