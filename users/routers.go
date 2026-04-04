package users

import (
	"github.com/gin-gonic/gin"
	"github.com/likui628/golang-gin-realworld-example-app/database"
	"net/http"
)

func UsersRegister(router *gin.RouterGroup) {
    router.POST("", UsersRegistration)
	router.POST("/", UsersRegistration)
}


func UsersRegistration(c *gin.Context) {
    var user UserModel
    if err := c.ShouldBindJSON(&user); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    database.DB.Create(&user)
    
    c.JSON(http.StatusOK, gin.H{"message": "registration success"})
}