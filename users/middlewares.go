package users

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/likui628/golang-gin-realworld-example-app/common"
)

const currentUserContextKey = "current_user"

var ErrUnauthorized = errors.New("unauthorized")

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorizationHeader := c.GetHeader("Authorization")
		tokenString, ok := extractToken(authorizationHeader)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, common.NewError("auth", ErrUnauthorized))
			return
		}

		userID, err := common.ParseToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, common.NewError("auth", ErrUnauthorized))
			return
		}

		userModel, err := newUserService().FindByID(userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, common.NewError("auth", ErrUnauthorized))
			return
		}

		c.Set(currentUserContextKey, userModel)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) (UserModel, bool) {
	value, exists := c.Get(currentUserContextKey)
	if !exists {
		return UserModel{}, false
	}

	userModel, ok := value.(UserModel)
	if !ok {
		return UserModel{}, false
	}

	return userModel, true
}

func extractToken(headerValue string) (string, bool) {
	parts := strings.SplitN(strings.TrimSpace(headerValue), " ", 2)
	if len(parts) != 2 {
		return "", false
	}

	scheme := strings.ToLower(parts[0])
	if scheme != "token" && scheme != "bearer" {
		return "", false
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}

	return token, true
}
