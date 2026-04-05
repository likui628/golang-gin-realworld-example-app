package users

import "gorm.io/gorm"

type UserRepository interface {
	Create(user *UserModel) error
	FindByEmail(email string) (UserModel, error)
	FindByID(id uint) (UserModel, error)
}

type GormUserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) GormUserRepository {
	return GormUserRepository{db: db}
}

func (repository GormUserRepository) Create(user *UserModel) error {
	return repository.db.Create(user).Error
}

func (repository GormUserRepository) FindByEmail(email string) (UserModel, error) {
	var user UserModel
	err := repository.db.Where("email = ?", email).First(&user).Error
	return user, err
}

func (repository GormUserRepository) FindByID(id uint) (UserModel, error) {
	var user UserModel
	err := repository.db.First(&user, id).Error
	return user, err
}
