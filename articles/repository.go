package articles

import (
	"gorm.io/gorm"
)

type ArticleRepository interface {
	Create(article *ArticleModel) error
	GetArticleBySlug(slug string) (ArticleModel, error)

	IsFavorited(userId uint, articleId uint) (bool, error)
	CountFavorites(articleId uint) (int64, error)
	FavoriteArticle(userId uint, slug string) (ArticleModel, error)
	UnfavoriteArticle(userId uint, slug string) (ArticleModel, error)

	FindOrCreateTags(tags []string) ([]TagModel, error)
	GetTags() ([]string, error)

	CreateComment(comment *CommentModel) error
	GetCommentsByArticleId(articleId uint) ([]CommentModel, error)
	DeleteComment(commentId uint, authId uint) error
}

type GormRepository struct {
	db *gorm.DB
}

func NewArticleRepository(db *gorm.DB) ArticleRepository {
	return GormRepository{db: db}
}

func (repository GormRepository) Create(article *ArticleModel) error {
	if err := repository.db.Create(article).Error; err != nil {
		return err
	}
	return repository.db.Preload("Tags").Preload("Author").First(article, article.ID).Error

}

func (repository GormRepository) FindOrCreateTags(tags []string) ([]TagModel, error) {
	result := make([]TagModel, 0, len(tags))
	for _, tag := range tags {
		var t TagModel
		err := repository.db.Where(TagModel{Tag: tag}).FirstOrCreate(&t).Error
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, nil
}

func (repository GormRepository) GetArticleBySlug(slug string) (ArticleModel, error) {
	var article ArticleModel
	if err := repository.db.Preload("Tags").Preload("Author").Where("slug = ?", slug).First(&article).Error; err != nil {
		return ArticleModel{}, err
	}
	return article, nil
}

func (repository GormRepository) IsFavorited(userId uint, articleId uint) (bool, error) {
	var favorite FavoriteModel
	err := repository.db.Where("user_id = ? AND article_id = ?", userId, articleId).First(&favorite).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (repository GormRepository) CountFavorites(articleId uint) (int64, error) {
	var count int64
	if err := repository.db.Model(&FavoriteModel{}).Where("article_id = ?", articleId).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (repository GormRepository) FavoriteArticle(userId uint, slug string) (ArticleModel, error) {
	var article ArticleModel
	if err := repository.db.Preload("Tags").Preload("Author").Where("slug = ?", slug).First(&article).Error; err != nil {
		return ArticleModel{}, err
	}

	favorite := FavoriteModel{
		UserId:    userId,
		ArticleId: article.ID,
	}
	if err := repository.db.Where(favorite).FirstOrCreate(&favorite).Error; err != nil {
		return ArticleModel{}, err
	}

	return article, nil
}

func (repository GormRepository) UnfavoriteArticle(userId uint, slug string) (ArticleModel, error) {
	article := ArticleModel{}
	err := repository.db.Preload("Tags").Preload("Author").Where("slug = ?", slug).First(&article).Error
	if err != nil {
		return ArticleModel{}, err
	}

	if err := repository.db.Where("user_id = ? AND article_id = ?", userId, article.ID).Delete(&FavoriteModel{}).Error; err != nil {
		return ArticleModel{}, err
	}

	return article, nil
}

func (repository GormRepository) GetTags() ([]string, error) {
	var tags []TagModel
	if err := repository.db.Find(&tags).Error; err != nil {
		return nil, err
	}

	result := make([]string, len(tags))
	for i, tag := range tags {
		result[i] = tag.Tag
	}
	return result, nil
}

func (repository GormRepository) CreateComment(comment *CommentModel) error {
	if err := repository.db.Create(comment).Error; err != nil {
		return err
	}
	return repository.db.Preload("Author").Preload("Article").First(comment, comment.ID).Error
}

func (repository GormRepository) GetCommentsByArticleId(articleId uint) ([]CommentModel, error) {
	var comments []CommentModel
	if err := repository.db.Preload("Author").Where("article_id = ?", articleId).Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

func (repository GormRepository) DeleteComment(commentId uint, authId uint) error {
	if err := repository.db.Where("id = ? AND author_id = ?", commentId, authId).Delete(&CommentModel{}).Error; err != nil {
		return err
	}
	return nil
}
