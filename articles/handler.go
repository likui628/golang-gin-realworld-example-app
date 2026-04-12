package articles

import (
	"errors"
	"net/http"
	"strconv"

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

func (handler *ArticleHandler) DeleteArticle(c *gin.Context) {
	currentUser, ok := users.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, common.NewError("auth", users.ErrUnauthorized))
		return
	}
	slug := c.Param("slug")
	if err := handler.service.DeleteArticle(slug, currentUser.ID); err != nil {
		c.JSON(http.StatusInternalServerError, common.NewError("database", err))
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (handler *ArticleHandler) GetArticle(c *gin.Context) {
	slug := c.Param("slug")
	currentUser, ok := users.CurrentUser(c)
	var userId uint
	if ok {
		userId = currentUser.ID
	}
	article, err := handler.service.GetArticleBySlug(slug, userId)
	if err != nil {
		c.JSON(http.StatusNotFound, common.NewError("article", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"article": ArticleSerializer{Article: article}.Response()})
}

func (handler *ArticleHandler) GetArticles(c *gin.Context) {
	currentUser, ok := users.CurrentUser(c)
	var userId uint
	if ok {
		userId = currentUser.ID
	}
	author := c.Query("author")
	tag := c.Query("tag")
	limit := c.Query("limit")
	if limit == "" {
		limit = "20"
	}
	offset := c.Query("offset")
	if offset == "" {
		offset = "0"
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt < 0 {
		c.JSON(http.StatusBadRequest, common.NewError("limit", errors.New("invalid limit")))
		return
	}
	offsetInt, err := strconv.Atoi(offset)
	if err != nil || offsetInt < 0 {
		c.JSON(http.StatusBadRequest, common.NewError("offset", errors.New("invalid offset")))
		return
	}
	articles, err := handler.service.GetArticles(userId, author, tag, limitInt, offsetInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.NewError("database", err))
		return
	}
	var articleResponses []ArticleResponse
	for _, article := range articles {
		articleResponses = append(articleResponses, ArticleSerializer{Article: article}.Response())
	}
	c.JSON(http.StatusOK, gin.H{"articles": articleResponses, "articleCount": len(articleResponses)})
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

func (handler *ArticleHandler) DeleteComment(c *gin.Context) {
	currentUser, ok := users.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, common.NewError("auth", users.ErrUnauthorized))
		return
	}
	commentId := c.Param("id")
	commentIdUint, err := strconv.ParseUint(commentId, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.NewError("invalid_id", err))
		return
	}
	if err := handler.service.DeleteComment(uint(commentIdUint), currentUser.ID); err != nil {
		c.JSON(http.StatusInternalServerError, common.NewError("database", err))
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
