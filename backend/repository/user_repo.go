package repository

import (
	"cuetiy-backend/config"
	"cuetiy-backend/model"
)

type UserRepository struct{}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

func (r *UserRepository) Create(user *model.User) error {
	return config.DB.Create(user).Error
}

func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := config.DB.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByID(id int64) (*model.User, error) {
	var user model.User
	err := config.DB.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(user *model.User) error {
	return config.DB.Save(user).Error
}

func (r *UserRepository) UpdateAvatar(userID int64, avatarURL string) error {
	return config.DB.Model(&model.User{}).Where("id = ?", userID).Update("avatar", avatarURL).Error
}
