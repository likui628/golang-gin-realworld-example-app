package articles

import (
	"github.com/likui628/golang-gin-realworld-example-app/common"
	"github.com/likui628/golang-gin-realworld-example-app/users"
)

type CreateArticleInput struct {
	Title       string
	Description string
	Body        string
	TagList     []string
}

type ArticleOutput struct {
	ArticleModel
	Favorited       bool
	FavoritesCount  int64
	AuthorFollowing bool
}

type CreateCommentInput struct {
	Body string
}

type CommentOutput struct {
	CommentModel
}

type ArticleService struct {
	repository     ArticleRepository
	userRepository users.UserRepository
}

func NewArticleService(repository ArticleRepository, userRepository users.UserRepository) ArticleService {
	return ArticleService{repository: repository, userRepository: userRepository}
}

func (service *ArticleService) buildArticleOutput(article ArticleModel, userId uint, favorited bool, favoritesCount int64) (ArticleOutput, error) {
	authorFollowing := false
	if userId != 0 && userId != article.AuthorId {
		var err error
		authorFollowing, err = service.userRepository.IsFollowing(userId, article.AuthorId)
		if err != nil {
			return ArticleOutput{}, err
		}
	}

	return ArticleOutput{
		ArticleModel:    article,
		Favorited:       favorited,
		FavoritesCount:  favoritesCount,
		AuthorFollowing: authorFollowing,
	}, nil
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

	return service.buildArticleOutput(article, authorID, false, 0)
}

func (service *ArticleService) GetArticleBySlug(slug string, userId uint) (ArticleOutput, error) {
	article, err := service.repository.GetArticleBySlug(slug)
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

	return service.buildArticleOutput(article, userId, favorited, favoritesCount)
}

func (service *ArticleService) GetArticles(userId uint, authorUsername, tag string) ([]ArticleOutput, error) {
	articles, err := service.repository.GetArticles(authorUsername, tag)
	if err != nil {
		return nil, err
	}
	articleIDs := make([]uint, len(articles))
	authorIDs := make([]uint, len(articles))
	for i, a := range articles {
		articleIDs[i] = a.ID
		authorIDs[i] = a.AuthorId
	}

	favoritedMap, err := service.repository.GetFavoritedArticleIDs(userId, articleIDs)
	if err != nil {
		return nil, err
	}
	favoritesCountMap, err := service.repository.CountFavoritesByArticleIDs(articleIDs)
	if err != nil {
		return nil, err
	}

	followingMap, err := service.userRepository.GetFollowingByAuthorIDs(authorIDs)
	if err != nil {
		return nil, err
	}

	outputs := make([]ArticleOutput, len(articles))
	for i, article := range articles {
		outputs[i] = ArticleOutput{
			ArticleModel:    article,
			Favorited:       favoritedMap[article.ID],
			FavoritesCount:  favoritesCountMap[article.ID],
			AuthorFollowing: followingMap[article.AuthorId],
		}
	}
	return outputs, nil
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

	return service.buildArticleOutput(article, userId, favorited, favoritesCount)
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

	return service.buildArticleOutput(article, userId, favorited, favoritesCount)
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

func (service *ArticleService) DeleteComment(commentId uint, userId uint) error {
	if err := service.repository.DeleteComment(commentId, userId); err != nil {
		return err
	}
	return nil
}
