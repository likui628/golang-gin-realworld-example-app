package users

import "github.com/gin-gonic/gin"

func UsersRegister(router *gin.RouterGroup, handler UserHandler) {
	router.POST("", handler.Register)
	router.POST("/login", handler.Login)
}

func UserRegister(router *gin.RouterGroup, handler UserHandler) {
	router.GET("", handler.CurrentUser)
	router.PUT("", handler.UpdateUser)
}

func ProfileRegister(router *gin.RouterGroup, handler UserHandler) {
	router.POST("/:uid/follow", handler.FollowUser)
	router.DELETE("/:uid/follow", handler.UnfollowUser)
}

func ProfilePublicRegister(router *gin.RouterGroup, handler UserHandler) {
	router.GET("/:uid", handler.GetProfile)
}
