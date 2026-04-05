package articles

import "github.com/likui628/golang-gin-realworld-example-app/common"

type CreateArticleInput struct {
	Title       string
	Description string
	Body        string
}

type ArticleService struct {
	repository ArticleRepository
}

func NewArticleService(repository ArticleRepository) ArticleService {
	return ArticleService{repository: repository}
}

func (service *ArticleService) CreateArticle(authorID uint, input CreateArticleInput) (ArticleModel, error) {
	article := ArticleModel{
		Slug:        common.GenerateSlug(input.Title),
		Title:       input.Title,
		Description: input.Description,
		Body:        input.Body,
		AuthorId:    authorID,
	}

	if err := service.repository.Create(&article); err != nil {
		return ArticleModel{}, err
	}

	return article, nil
}
