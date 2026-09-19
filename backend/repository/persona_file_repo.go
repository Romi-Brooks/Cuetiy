package repository

import (
	"cuetiy-backend/config"
	"cuetiy-backend/model"
)

type PersonaFileRepository struct{}

func NewPersonaFileRepository() *PersonaFileRepository {
	return &PersonaFileRepository{}
}

func (r *PersonaFileRepository) Create(pf *model.PersonaFile) error {
	return config.DB.Create(pf).Error
}

func (r *PersonaFileRepository) CreateBatch(files []model.PersonaFile) error {
	if len(files) == 0 {
		return nil
	}
	return config.DB.Create(&files).Error
}

func (r *PersonaFileRepository) FindByID(id int64) (*model.PersonaFile, error) {
	var pf model.PersonaFile
	err := config.DB.First(&pf, id).Error
	if err != nil {
		return nil, err
	}
	return &pf, nil
}

func (r *PersonaFileRepository) FindByPersonaID(personaID int64) ([]model.PersonaFile, error) {
	var files []model.PersonaFile
	err := config.DB.Where("persona_id = ?", personaID).
		Order("priority ASC, created_at ASC").
		Find(&files).Error
	return files, err
}

func (r *PersonaFileRepository) DeleteByPersonaID(personaID int64) error {
	return config.DB.Where("persona_id = ?", personaID).Delete(&model.PersonaFile{}).Error
}

func (r *PersonaFileRepository) Delete(id int64) error {
	return config.DB.Delete(&model.PersonaFile{}, id).Error
}

func (r *PersonaFileRepository) CountByPersonaID(personaID int64) (int64, error) {
	var count int64
	err := config.DB.Model(&model.PersonaFile{}).Where("persona_id = ?", personaID).Count(&count).Error
	return count, err
}
