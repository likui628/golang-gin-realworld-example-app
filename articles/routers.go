package articles

import "github.com/gin-gonic/gin"

func ArticlesRegister(router *gin.RouterGroup, handler ArticleHandler) {
	router.POST("", handler.CreateArticle)
}
