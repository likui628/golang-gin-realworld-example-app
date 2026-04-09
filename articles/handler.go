package articles

import (
	"log"
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

func (handler *ArticleHandler) GetArticle(c *gin.Context) {
	slug := c.Param("slug")
	article, err := handler.service.GetArticleBySlug(slug)
	log.Printf("article.Tags: %+v", article.Tags)
	if err != nil {
		c.JSON(http.StatusNotFound, common.NewError("article", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"article": ArticleSerializer{Article: article}.Response()})
}

func (handler *ArticleHandler) FavoriteArticle(c *gin.Context) {
	currentUser, ok := users.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, common.NewError("auth", users.ErrUnauthorized))
		return
	}

	slug := c.Param("slug")

	article, err := handler.service.FavoriteArticle(currentUser.ID, slug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.NewError("database", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"article": ArticleSerializer{Article: article}.Response()})
}

func (handler *ArticleHandler) UnfavoriteArticle(c *gin.Context) {
	currentUser, ok := users.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, common.NewError("auth", users.ErrUnauthorized))
		return
	}

	slug := c.Param("slug")
	log.Printf("%d - %s\n", currentUser.ID, slug)

	article, err := handler.service.UnfavoriteArticle(currentUser.ID, slug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.NewError("database", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"article": ArticleSerializer{Article: article}.Response()})
}

func (handler *ArticleHandler) GetTags(c *gin.Context) {
	tags, err := handler.service.GetTags()
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.NewError("database", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

func (handler *ArticleHandler) CreateComment(c *gin.Context) {
	currentUser, ok := users.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, common.NewError("auth", users.ErrUnauthorized))
		return
	}

	slug := c.Param("slug")

	commentValidator := CreateCommentInputValidator{}
	if err := commentValidator.Bind(c); err != nil {
		c.JSON(http.StatusUnprocessableEntity, common.NewValidatorError(err))
		return
	}
	commentInput := commentValidator.Input()
	comment, err := handler.service.CreateComment(currentUser.ID, slug, commentInput)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.NewError("database", err))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"comment": CommentSerializer{Comment: comment}.Response()})
}

func (handler *ArticleHandler) GetComments(c *gin.Context) {
	slug := c.Param("slug")
	comments, err := handler.service.GetCommentsByArticleSlug(slug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.NewError("database", err))
		return
	}
	var commentResponses []CommentResponse
	for _, comment := range comments {
		commentResponses = append(commentResponses, CommentSerializer{Comment: comment}.Response())
	}
	c.JSON(http.StatusOK, gin.H{"comments": commentResponses})
}
