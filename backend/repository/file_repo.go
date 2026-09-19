package repository

import (
	"cuetiy-backend/config"
	"cuetiy-backend/model"
)

type FileRepository struct{}

func NewFileRepository() *FileRepository {
	return &FileRepository{}
}

func (r *FileRepository) Create(record *model.FileRecord) error {
	return config.DB.Create(record).Error
}

func (r *FileRepository) Update(record *model.FileRecord) error {
	return config.DB.Save(record).Error
}

func (r *FileRepository) FindByID(id int64) (*model.FileRecord, error) {
	var record model.FileRecord
	err := config.DB.First(&record, id).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *FileRepository) FindByUserID(userID int64, fileType string) ([]model.FileRecord, error) {
	var records []model.FileRecord
	query := config.DB.Where("user_id = ?", userID)
	if fileType != "" {
		query = query.Where("file_type = ?", fileType)
	}
	err := query.Order("created_at DESC").Find(&records).Error
	return records, err
}

func (r *FileRepository) FindByReference(referenceType string, referenceID int64) ([]model.FileRecord, error) {
	var records []model.FileRecord
	err := config.DB.Where("reference_type = ? AND reference_id = ?", referenceType, referenceID).
		Order("created_at DESC").
		Find(&records).Error
	return records, err
}

func (r *FileRepository) SoftDelete(id int64) error {
	return config.DB.Delete(&model.FileRecord{}, id).Error
}

func (r *FileRepository) HardDelete(id int64) error {
	return config.DB.Unscoped().Delete(&model.FileRecord{}, id).Error
}
