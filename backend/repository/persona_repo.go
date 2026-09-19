package repository

import (
	"cuetiy-backend/config"
	"cuetiy-backend/model"
)

type PersonaRepository struct{}

func NewPersonaRepository() *PersonaRepository {
	return &PersonaRepository{}
}

func (r *PersonaRepository) Create(persona *model.Persona) error {
	return config.DB.Create(persona).Error
}

func (r *PersonaRepository) FindByID(id int64) (*model.Persona, error) {
	var persona model.Persona
	err := config.DB.First(&persona, id).Error
	if err != nil {
		return nil, err
	}
	return &persona, nil
}

func (r *PersonaRepository) FindByUserID(userID int64) ([]model.Persona, error) {
	var personas []model.Persona
	err := config.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&personas).Error
	return personas, err
}

func (r *PersonaRepository) FindAllVisible(userID int64) ([]model.Persona, error) {
	var personas []model.Persona
	err := config.DB.Where("user_id = ? OR user_id = 0", userID).
		Order("user_id ASC, created_at DESC").
		Find(&personas).Error
	return personas, err
}

func (r *PersonaRepository) Update(persona *model.Persona) error {
	return config.DB.Save(persona).Error
}

func (r *PersonaRepository) Delete(id int64) error {
	return config.DB.Delete(&model.Persona{}, id).Error
}

func (r *PersonaRepository) FindByName(name string) (*model.Persona, error) {
	var persona model.Persona
	err := config.DB.Where("name = ?", name).First(&persona).Error
	if err != nil {
		return nil, err
	}
	return &persona, nil
}

func (r *PersonaRepository) FindByDirName(dirName string) (*model.Persona, error) {
	var persona model.Persona
	err := config.DB.Where("dir_name = ?", dirName).First(&persona).Error
	if err != nil {
		return nil, err
	}
	return &persona, nil
}

func (r *PersonaRepository) FindActive() ([]model.Persona, error) {
	var personas []model.Persona
	err := config.DB.Where("is_active = ?", true).
		Order("user_id ASC, created_at DESC").
		Find(&personas).Error
	return personas, err
}

func (r *PersonaRepository) DeleteAllByUserID(userID int64) error {
	return config.DB.Unscoped().Where("user_id = ?", userID).Delete(&model.Persona{}).Error
}

func (r *PersonaRepository) HardDelete(id int64) error {
	return config.DB.Unscoped().Delete(&model.Persona{}, id).Error
}

func (r *PersonaRepository) SetConversationPersona(convID int64, personaID *int64) error {
	return config.DB.Model(&model.Conversation{}).Where("id = ?", convID).Update("persona_id", personaID).Error
}
