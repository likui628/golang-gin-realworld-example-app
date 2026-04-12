package articles

import "github.com/gin-gonic/gin"

func ArticlesRegister(router *gin.RouterGroup, handler ArticleHandler) {
	router.POST("", handler.CreateArticle)
	router.DELETE("/:slug", handler.DeleteArticle)
	router.POST("/:slug/favorite", handler.FavoriteArticle)
	router.DELETE("/:slug/favorite", handler.UnfavoriteArticle)

	router.POST("/:slug/comments", handler.CreateComment)
	router.DELETE("/:slug/comments/:id", handler.DeleteComment)
}

func ArticlePublicRegister(router *gin.RouterGroup, handler ArticleHandler) {
	router.GET("", handler.GetArticles)

	router.GET("/:slug", handler.GetArticle)
	router.GET("/:slug/comments", handler.GetComments)
}

func TagsRegister(router *gin.RouterGroup, handler ArticleHandler) {
	router.GET("", handler.GetTags)
}
