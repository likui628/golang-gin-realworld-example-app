package users

import (
	"errors"
	"strings"

	"github.com/likui628/golang-gin-realworld-example-app/common"
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

type UserOutput struct {
	UserModel
	Token string
}

type UserService struct {
	repository UserRepository
}

func NewUserService(repository UserRepository) UserService {
	return UserService{repository: repository}
}

func (service UserService) Register(input RegisterUserInput) (UserOutput, error) {
	_, err := service.repository.FindByEmail(input.Email)
	if err == nil {
		return UserOutput{}, ErrEmailAlreadyTaken
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return UserOutput{}, err
	}

	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		return UserOutput{}, err
	}

	user := UserModel{
		Username:     input.Username,
		Email:        input.Email,
		Bio:          input.Bio,
		PasswordHash: passwordHash,
	}
	if input.Image != "" {
		user.Image = &input.Image
	}
	if err := service.repository.Create(&user); err != nil {
		if isUniqueConstraintError(err) {
			return UserOutput{}, ErrEmailAlreadyTaken
		}
		return UserOutput{}, err
	}
	return UserOutput{UserModel: user, Token: common.GenToken(user.ID)}, nil
}

func (service UserService) Login(input LoginUserInput) (UserOutput, error) {
	user, err := service.repository.FindByEmail(input.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return UserOutput{}, ErrInvalidCredentials
		}
		return UserOutput{}, err
	}
	if err := checkPassword(user.PasswordHash, input.Password); err != nil {
		return UserOutput{}, ErrInvalidCredentials
	}
	return UserOutput{UserModel: user, Token: common.GenToken(user.ID)}, nil
}

func (service UserService) FindByID(id uint) (UserModel, error) {
	return service.repository.FindByID(id)
}

func isUniqueConstraintError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint") || strings.Contains(message, "duplicated key")
}
