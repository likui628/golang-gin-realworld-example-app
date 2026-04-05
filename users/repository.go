package users

import "gorm.io/gorm"

type UserRepository interface {
	Create(user *UserModel) error
	Update(user *UserModel) error
	FindByEmail(email string) (UserModel, error)
	FindByID(id uint) (UserModel, error)
}

type GormRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return GormRepository{db: db}
}

func (repository GormRepository) Create(user *UserModel) error {
	return repository.db.Create(user).Error
}

func (repository GormRepository) Update(user *UserModel) error {
	return repository.db.Save(user).Error
}

func (repository GormRepository) FindByEmail(email string) (UserModel, error) {
	var user UserModel
	err := repository.db.Where("email = ?", email).First(&user).Error
	return user, err
}

func (repository GormRepository) FindByID(id uint) (UserModel, error) {
	var user UserModel
	err := repository.db.First(&user, id).Error
	return user, err
}
