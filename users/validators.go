package users

import (
	"github.com/likui628/golang-gin-realworld-example-app/common"

	"github.com/gin-gonic/gin"
)

type UserModelValidator struct {
	User struct {
		Username string `form:"username" json:"username" binding:"required,min=4,max=255"`
		Email    string `form:"email" json:"email" binding:"required,email"`
		Password string `form:"password" json:"password" binding:"required,min=8,max=255"`
		Bio      string `form:"bio" json:"bio" binding:"max=1024"`
		Image    string `form:"image" json:"image" binding:"omitempty,url"`
	} `json:"user"`
}

func (validator *UserModelValidator) Bind(c *gin.Context) error {
	return common.Bind(c, validator)
}

func (validator UserModelValidator) Input() RegisterUserInput {
	return RegisterUserInput{
		Username: validator.User.Username,
		Email:    validator.User.Email,
		Password: validator.User.Password,
		Bio:      validator.User.Bio,
		Image:    validator.User.Image,
	}
}

func NewUserModelValidator() UserModelValidator {
	userModelValidator := UserModelValidator{}
	return userModelValidator
}

type UserLoginValidator struct {
	User struct {
		Email    string `form:"email" json:"email" binding:"required,email"`
		Password string `form:"password" json:"password" binding:"required,min=8,max=255"`
	} `json:"user"`
}

func (validator *UserLoginValidator) Bind(c *gin.Context) error {
	return common.Bind(c, validator)
}

func (validator UserLoginValidator) Input() LoginUserInput {
	return LoginUserInput{
		Email:    validator.User.Email,
		Password: validator.User.Password,
	}
}

func NewUserLoginValidator() UserLoginValidator {
	userLoginValidator := UserLoginValidator{}
	return userLoginValidator
}
