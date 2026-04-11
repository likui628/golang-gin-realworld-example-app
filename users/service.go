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
	ErrUserNotFound       = errors.New("User not found")
	ErrInvalidID          = errors.New("Invalid ID")
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

type UpdateUserInput struct {
	Username string
	Email    string
	Bio      string
	Image    string
}

type UserOutput struct {
	UserModel
	Token string
}

type UserProfileOutput struct {
	UserModel
	Following bool
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
	if !errors.Is(err, gorm.ErrRecordNotFound) {
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

func (service UserService) UpdateUser(id uint, input UpdateUserInput) (UserOutput, error) {
	user, err := service.repository.FindByID(id)
	if err != nil {
		return UserOutput{}, err
	}

	if input.Username != "" {
		user.Username = input.Username
	}
	if input.Email != "" {
		user.Email = input.Email
	}
	if input.Bio != "" {
		user.Bio = input.Bio
	}
	if input.Image != "" {
		user.Image = &input.Image
	}

	if err := service.repository.Update(&user); err != nil {
		if isUniqueConstraintError(err) {
			return UserOutput{}, ErrEmailAlreadyTaken
		}
		return UserOutput{}, err
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

func (service UserService) GetProfile(uid uint, currentUserID *uint) (UserProfileOutput, error) {
	profile, err := service.repository.FindByID(uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return UserProfileOutput{}, ErrUserNotFound
		}
		return UserProfileOutput{}, err
	}

	following := false
	if currentUserID != nil && *currentUserID != profile.ID {
		following, err = service.repository.IsFollowing(*currentUserID, profile.ID)
		if err != nil {
			return UserProfileOutput{}, err
		}
	}

	return UserProfileOutput{UserModel: profile, Following: following}, nil
}

func (service UserService) FollowUser(followerID uint, followedID uint) (UserProfileOutput, error) {
	if followerID == followedID {
		return UserProfileOutput{}, errors.New("cannot follow yourself")
	}
	profile, err := service.repository.FindByID(followedID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return UserProfileOutput{}, ErrUserNotFound
		}
		return UserProfileOutput{}, err
	}

	if err := service.repository.FollowUser(followerID, followedID); err != nil {
		return UserProfileOutput{}, err
	}

	return UserProfileOutput{UserModel: profile, Following: true}, nil
}

func (service UserService) UnfollowUser(followerID uint, followedID uint) (UserProfileOutput, error) {
	if followerID == followedID {
		return UserProfileOutput{}, errors.New("cannot unfollow yourself")
	}
	profile, err := service.repository.FindByID(followedID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return UserProfileOutput{}, ErrUserNotFound
		}
		return UserProfileOutput{}, err
	}

	if err := service.repository.UnfollowUser(followerID, followedID); err != nil {
		return UserProfileOutput{}, err
	}

	return UserProfileOutput{UserModel: profile, Following: false}, nil
}
