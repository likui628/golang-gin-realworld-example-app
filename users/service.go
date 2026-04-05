package users

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("Not Registered email or invalid password")
	ErrEmailAlreadyTaken  = errors.New("has already been taken")
)

type RegisterUserInput struct {
	Username string
	Email    string
	Password string
	Bio      string
	Image    string
}

type LoginUserInput struct {
	Email    string
	Password string
}

type UserService struct {
	repository UserRepository
}

func NewUserService(repository UserRepository) UserService {
	return UserService{repository: repository}
}

func (service UserService) Register(input RegisterUserInput) (UserModel, error) {
	_, err := service.repository.FindByEmail(input.Email)
	if err == nil {
		return UserModel{}, ErrEmailAlreadyTaken
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return UserModel{}, err
	}

	user := UserModel{
		Username: input.Username,
		Email:    input.Email,
		Bio:      input.Bio,
	}
	if input.Image != "" {
		user.Image = &input.Image
	}
	if err := user.setPassword(input.Password); err != nil {
		return UserModel{}, err
	}
	if err := service.repository.Create(&user); err != nil {
		if isUniqueConstraintError(err) {
			return UserModel{}, ErrEmailAlreadyTaken
		}
		return UserModel{}, err
	}
	return user, nil
}

func (service UserService) Login(input LoginUserInput) (UserModel, error) {
	user, err := service.repository.FindByEmail(input.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return UserModel{}, ErrInvalidCredentials
		}
		return UserModel{}, err
	}
	if err := user.checkPassword(input.Password); err != nil {
		return UserModel{}, ErrInvalidCredentials
	}
	return user, nil
}

func (service UserService) FindByID(id uint) (UserModel, error) {
	return service.repository.FindByID(id)
}

func isUniqueConstraintError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint") || strings.Contains(message, "duplicated key")
}
