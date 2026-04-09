package articles

import (
	"github.com/likui628/golang-gin-realworld-example-app/common"
)

type CreateArticleInput struct {
	Title       string
	Description string
	Body        string
	TagList     []string
}

type ArticleOutput struct {
	ArticleModel
	Favorited      bool
	FavoritesCount int64
}

type CreateCommentInput struct {
	Body string
}

type CommentOutput struct {
	CommentModel
}

type ArticleService struct {
	repository ArticleRepository
}

func NewArticleService(repository ArticleRepository) ArticleService {
	return ArticleService{repository: repository}
}

func (service *ArticleService) CreateArticle(authorID uint, input CreateArticleInput) (ArticleOutput, error) {
	tags, err := service.repository.FindOrCreateTags(input.TagList)
	if err != nil {
		return ArticleOutput{}, err
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
		return ArticleOutput{}, err
	}

	return ArticleOutput{
		ArticleModel:   article,
		Favorited:      false,
		FavoritesCount: 0,
	}, nil
}

func (service *ArticleService) GetArticleBySlug(slug string) (ArticleOutput, error) {
	article, err := service.repository.GetArticleBySlug(slug)
	if err != nil {
		return ArticleOutput{}, err
	}
	favorited, err := service.repository.IsFavorited(0, article.ID)
	if err != nil {
		return ArticleOutput{}, err
	}
	favoritesCount, err := service.repository.CountFavorites(article.ID)
	if err != nil {
		return ArticleOutput{}, err
	}

	return ArticleOutput{
		ArticleModel:   article,
		Favorited:      favorited,
		FavoritesCount: favoritesCount,
	}, nil
}

func (service *ArticleService) FavoriteArticle(userId uint, slug string) (ArticleOutput, error) {
	article, err := service.repository.FavoriteArticle(userId, slug)
	if err != nil {
		return ArticleOutput{}, err
	}

	favorited, err := service.repository.IsFavorited(userId, article.ID)
	if err != nil {
		return ArticleOutput{}, err
	}
	favoritesCount, err := service.repository.CountFavorites(article.ID)
	if err != nil {
		return ArticleOutput{}, err
	}

	return ArticleOutput{
		ArticleModel:   article,
		Favorited:      favorited,
		FavoritesCount: favoritesCount,
	}, nil
}

func (service *ArticleService) UnfavoriteArticle(userId uint, slug string) (ArticleOutput, error) {
	article, err := service.repository.UnfavoriteArticle(userId, slug)
	if err != nil {
		return ArticleOutput{}, err
	}

	favorited, err := service.repository.IsFavorited(userId, article.ID)
	if err != nil {
		return ArticleOutput{}, err
	}
	favoritesCount, err := service.repository.CountFavorites(article.ID)
	if err != nil {
		return ArticleOutput{}, err
	}

	return ArticleOutput{
		ArticleModel:   article,
		Favorited:      favorited,
		FavoritesCount: favoritesCount,
	}, nil
}

func (service *ArticleService) GetTags() ([]string, error) {
	return service.repository.GetTags()
}

func (service *ArticleService) CreateComment(userId uint, slug string, input CreateCommentInput) (CommentOutput, error) {
	article, err := service.repository.GetArticleBySlug(slug)
	if err != nil {
		return CommentOutput{}, err
	}

	comment := CommentModel{
		Body:      input.Body,
		AuthorId:  userId,
		ArticleId: article.ID,
	}

	if err := service.repository.CreateComment(&comment); err != nil {
		return CommentOutput{}, err
	}

	return CommentOutput{
		CommentModel: comment,
	}, nil
}

func (service *ArticleService) GetCommentsByArticleSlug(slug string) ([]CommentOutput, error) {
	article, err := service.repository.GetArticleBySlug(slug)
	if err != nil {
		return nil, err
	}
	comments, err := service.repository.GetCommentsByArticleId(article.ID)
	if err != nil {
		return nil, err
	}
	var commentOutputs []CommentOutput
	for _, comment := range comments {
		commentOutputs = append(commentOutputs, CommentOutput{CommentModel: comment})
	}
	return commentOutputs, nil
}
