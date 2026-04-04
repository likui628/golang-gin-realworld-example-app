package main

import (
    "github.com/gin-gonic/gin"
    "github.com/likui628/golang-gin-realworld-example-app/database"
    "github.com/likui628/golang-gin-realworld-example-app/users"
)

func main() {
    db := database.Init()
    users.AutoMigrate(db)

    r := gin.Default()

    v1 := r.Group("/api")
    users.UsersRegister(v1.Group("/users"))

    r.Run(":8080")
}