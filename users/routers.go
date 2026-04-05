package users

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/likui628/golang-gin-realworld-example-app/common"
)

func UsersRegister(router *gin.RouterGroup) {
	router.POST("", UsersRegistration)
	router.POST("/login", UsersLogin)
}

func newUserService() UserService {
	return NewUserService(NewUserRepository(common.GetDB()))
}

func UsersRegistration(c *gin.Context) {
	userModelValidator := NewUserModelValidator()
	if err := userModelValidator.Bind(c); err != nil {
		c.JSON(http.StatusUnprocessableEntity, common.NewValidatorError(err))
		return
	}

	userModel, err := newUserService().Register(userModelValidator.Input())
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyTaken) {
			c.JSON(http.StatusUnprocessableEntity, common.NewError("email", err))
			return
		}
		c.JSON(http.StatusInternalServerError, common.NewError("database", err))
		return
	}

	serializer := UserSerializer{User: userModel}
	c.JSON(http.StatusCreated, gin.H{"user": serializer.Response()})
}

func UsersLogin(c *gin.Context) {
	loginValidator := NewUserLoginValidator()
	if err := loginValidator.Bind(c); err != nil {
		c.JSON(http.StatusUnprocessableEntity, common.NewValidatorError(err))
		return
	}

	userModel, err := newUserService().Login(loginValidator.Input())
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, common.NewError("login", err))
			return
		}
		c.JSON(http.StatusInternalServerError, common.NewError("database", err))
		return
	}

	serializer := UserSerializer{User: userModel}
	c.JSON(http.StatusOK, gin.H{"user": serializer.Response()})
}
