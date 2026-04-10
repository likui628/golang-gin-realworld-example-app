package users

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/likui628/golang-gin-realworld-example-app/common"
)

type UserHandler struct {
	service UserService
}

func NewUserHandler(service UserService) UserHandler {
	return UserHandler{service: service}
}

func (handler UserHandler) Register(c *gin.Context) {
	userModelValidator := NewUserModelValidator()
	if err := userModelValidator.Bind(c); err != nil {
		c.JSON(http.StatusUnprocessableEntity, common.NewValidatorError(err))
		return
	}

	userModel, err := handler.service.Register(userModelValidator.Input())
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyTaken) {
			c.JSON(http.StatusUnprocessableEntity, common.NewError("email", err))
			return
		}
		c.JSON(http.StatusInternalServerError, common.NewError("database", err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": UserSerializer{User: userModel}.Response()})
}

func (handler UserHandler) Login(c *gin.Context) {
	loginValidator := NewUserLoginValidator()
	if err := loginValidator.Bind(c); err != nil {
		c.JSON(http.StatusUnprocessableEntity, common.NewValidatorError(err))
		return
	}

	userModel, err := handler.service.Login(loginValidator.Input())
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, common.NewError("login", err))
			return
		}
		c.JSON(http.StatusInternalServerError, common.NewError("database", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": UserSerializer{User: userModel}.Response()})
}

func (handler UserHandler) CurrentUser(c *gin.Context) {
	currentUser, ok := CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, common.NewError("auth", ErrUnauthorized))
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": UserSerializer{User: UserOutput{
		UserModel: currentUser,
		Token:     common.GenToken(currentUser.ID),
	}}.Response()})
}

func (handler UserHandler) UpdateUser(c *gin.Context) {
	currentUser, ok := CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, common.NewError("auth", ErrUnauthorized))
		return
	}

	updateValidator := NewUpdateValidator()
	if err := updateValidator.Bind(c); err != nil {
		c.JSON(http.StatusUnprocessableEntity, common.NewValidatorError(err))
		return
	}

	updatedUser, err := handler.service.UpdateUser(currentUser.ID, updateValidator.Input())
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.NewError("database", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": UserSerializer{User: updatedUser}.Response()})
}

func (handler UserHandler) GetProfile(c *gin.Context) {
	uid := c.Param("uid")
	uidUint, err := strconv.ParseUint(uid, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.NewError("profile", ErrInvalidID))
		return
	}

	var currentUserID *uint
	if currentUser, ok := CurrentUser(c); ok {
		currentUserID = &currentUser.ID
	}

	profile, err := handler.service.GetProfile(uint(uidUint), currentUserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			c.JSON(http.StatusNotFound, common.NewError("profile", err))
			return
		}
		c.JSON(http.StatusInternalServerError, common.NewError("database", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"profile": ProfileSerializer{Profile: profile}.Response()})
}

func (handler UserHandler) FollowUser(c *gin.Context) {
	currentUser, ok := CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, common.NewError("auth", ErrUnauthorized))
		return
	}

	uid := c.Param("uid")
	uidUint, err := strconv.ParseUint(uid, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.NewError("profile", ErrInvalidID))
		return
	}

	profile, err := handler.service.FollowUser(currentUser.ID, uint(uidUint))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			c.JSON(http.StatusNotFound, common.NewError("profile", err))
			return
		}
		c.JSON(http.StatusInternalServerError, common.NewError("database", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"profile": ProfileSerializer{Profile: profile}.Response()})
}
