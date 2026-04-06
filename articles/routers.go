package articles

import "github.com/gin-gonic/gin"

func ArticlesRegister(router *gin.RouterGroup, handler ArticleHandler) {
	router.POST("", handler.CreateArticle)
	router.GET("/:slug", handler.GetArticle)
	router.POST("/:slug/favorite", handler.FavoriteArticle)
	router.DELETE("/:slug/unfavorite", handler.UnfavoriteArticle)
}

func TagsRegister(router *gin.RouterGroup, handler ArticleHandler) {
	router.GET("", handler.GetTags)
}
