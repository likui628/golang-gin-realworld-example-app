package articles

import (
	"github.com/gin-gonic/gin"
	"github.com/likui628/golang-gin-realworld-example-app/common"
)

type CreateArticleInputValidator struct {
	Article struct {
		Title       string   `json:"title" binding:"required"`
		Description string   `json:"description" binding:"required"`
		Body        string   `json:"body" binding:"required"`
		TagList     []string `json:"tagList"`
	} `json:"article"`
}

func (validator *CreateArticleInputValidator) Bind(c *gin.Context) error {
	return common.Bind(c, validator)
}

func (validator *CreateArticleInputValidator) Input() CreateArticleInput {
	return CreateArticleInput{
		Title:       validator.Article.Title,
		Description: validator.Article.Description,
		Body:        validator.Article.Body,
		TagList:     validator.Article.TagList,
	}
}

type CreateCommentInputValidator struct {
	Comment struct {
		Body string `json:"body" binding:"required"`
	} `json:"comment"`
}

func (validator *CreateCommentInputValidator) Bind(c *gin.Context) error {
	return common.Bind(c, validator)
}

func (validator *CreateCommentInputValidator) Input() CreateCommentInput {
	return CreateCommentInput{
		Body: validator.Comment.Body,
	}
}
