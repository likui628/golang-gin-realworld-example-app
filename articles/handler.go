package articles

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/likui628/golang-gin-realworld-example-app/common"
	"github.com/likui628/golang-gin-realworld-example-app/users"
)

type ArticleHandler struct {
	service ArticleService
}

func NewArticleHandler(service ArticleService) ArticleHandler {
	return ArticleHandler{service: service}
}

func (handler *ArticleHandler) CreateArticle(c *gin.Context) {
	currentUser, ok := users.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, common.NewError("auth", users.ErrUnauthorized))
		return
	}

	articleValidator := CreateArticleInputValidator{}
	if err := articleValidator.Bind(c); err != nil {
		c.JSON(http.StatusUnprocessableEntity, common.NewValidatorError(err))
		return
	}
	articleInput := articleValidator.Input()
	article, err := handler.service.CreateArticle(currentUser.ID, articleInput)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.NewError("database", err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"article": ArticleSerializer{Article: article}.Response()})
}
