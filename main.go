package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/likui628/golang-gin-realworld-example-app/articles"
	"github.com/likui628/golang-gin-realworld-example-app/common"
	"github.com/likui628/golang-gin-realworld-example-app/users"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) {
	users.AutoMigrate(db)
	articles.AutoMigrate(db)
}

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func setupRouter(db *gorm.DB) *gin.Engine {
	userRepository := users.NewUserRepository(db)
	userService := users.NewUserService(userRepository)
	userHandler := users.NewUserHandler(userService)

	articleRepository := articles.NewArticleRepository(db)
	articleService := articles.NewArticleService(articleRepository)
	articleHandler := articles.NewArticleHandler(articleService)

	r := gin.Default()
	v1 := r.Group("/api")
	users.UsersRegister(v1.Group("/users"), userHandler)

	authedUser := v1.Group("/user")
	authedUser.Use(users.AuthMiddleware(userService))
	users.UserRegister(authedUser, userHandler)

	authedArticles := v1.Group("/articles")
	authedArticles.Use(users.AuthMiddleware(userService))
	articles.ArticlesRegister(authedArticles, articleHandler)

	publicArticles := v1.Group("/articles")
	articles.ArticlePublicRegister(publicArticles, articleHandler)

	tags := v1.Group("/tags")
	articles.TagsRegister(tags, articleHandler)

	profiles := v1.Group("/profiles")
	profiles.Use(users.OptionalAuthMiddleware(userService))
	users.ProfileRegister(profiles, userHandler)

	return r
}

func closeDB(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Println("failed to get sql.DB:", err)
		return
	}
	sqlDB.Close()
}

func main() {
	LoadEnv()
	db := common.InitDatabase()
	Migrate(db)

	defer closeDB(db)

	r := setupRouter(db)

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to run server:", err)
	}

}
