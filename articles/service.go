package articles

import "github.com/likui628/golang-gin-realworld-example-app/common"

type CreateArticleInput struct {
	Title       string
	Description string
	Body        string
	TagList     []string
}

type ArticleService struct {
	repository ArticleRepository
}

func NewArticleService(repository ArticleRepository) ArticleService {
	return ArticleService{repository: repository}
}

func (service *ArticleService) CreateArticle(authorID uint, input CreateArticleInput) (ArticleModel, error) {
	tags, err := service.repository.FindOrCreateTags(input.TagList)
	if err != nil {
		return ArticleModel{}, err
	}

	article := ArticleModel{
		Slug:        common.GenerateSlug(input.Title),
		Title:       input.Title,
		Description: input.Description,
		Body:        input.Body,
		AuthorId:    authorID,
		Tags:        tags,
	}

	if err := service.repository.Create(&article); err != nil {
		return ArticleModel{}, err
	}

	return article, nil
}
