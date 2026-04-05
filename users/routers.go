package users

import "github.com/gin-gonic/gin"

func UsersRegister(router *gin.RouterGroup, handler UserHandler) {
	router.POST("", handler.Register)
	router.POST("/login", handler.Login)
}
