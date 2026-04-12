package articles

import (
	"github.com/likui628/golang-gin-realworld-example-app/users"
	"gorm.io/gorm"
)

type ArticleRepository interface {
	CreateArticle(article *ArticleModel) error
	UpdateArticle(article *ArticleModel) error
	DeleteArticle(slug string, authId uint) error
	GetArticleBySlug(slug string) (ArticleModel, error)
	GetArticles(authorUsername, tag string, limit, offset int) ([]ArticleModel, error)
	GetArticlesFeed(userId uint, limit, offset int) ([]ArticleModel, error)

	IsFavorited(userId uint, articleId uint) (bool, error)
	GetFavoritedArticleIDs(userId uint, articleIds []uint) (map[uint]bool, error)

	CountFavorites(articleId uint) (int64, error)
	CountFavoritesByArticleIDs(articleIds []uint) (map[uint]int64, error)
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

func (repository GormRepository) CreateArticle(article *ArticleModel) error {
	if err := repository.db.Create(article).Error; err != nil {
		return err
	}
	return repository.db.Preload("Tags").Preload("Author").First(article, article.ID).Error
}

func (repository GormRepository) UpdateArticle(article *ArticleModel) error {
	if err := repository.db.Save(article).Error; err != nil {
		return err
	}
	return repository.db.Preload("Tags").Preload("Author").First(article, article.ID).Error
}

func (repository GormRepository) DeleteArticle(slug string, authId uint) error {
	return repository.db.Where("slug = ? AND author_id = ?", slug, authId).Delete(&ArticleModel{}).Error
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

func (repository GormRepository) GetArticles(authorUsername, tag string, limit, offset int) ([]ArticleModel, error) {
	var articles []ArticleModel
	query := repository.db.Preload("Tags").Preload("Author")
	if authorUsername != "" {
		authorIDs := repository.db.Model(&users.UserModel{}).Select("id").Where("username = ?", authorUsername)
		query = query.Where("author_id IN (?)", authorIDs)
	}
	if tag != "" {
		articleIDs := repository.db.Table("article_tags").
			Select("article_model_id").
			Joins("JOIN tag_models ON tag_models.id = article_tags.tag_model_id").
			Where("tag_models.tag = ?", tag)
		query = query.Where("id IN (?)", articleIDs)
	}
	if err := query.Limit(limit).Offset(offset).Find(&articles).Error; err != nil {
		return nil, err
	}
	return articles, nil
}

func (repository GormRepository) GetArticlesFeed(userId uint, limit, offset int) ([]ArticleModel, error) {
	var articles []ArticleModel
	subQuery := repository.db.Model(&users.FollowModel{}).Select("followed_id").Where("follower_id = ?", userId)
	if err := repository.db.Preload("Tags").Preload("Author").Where("author_id IN (?)", subQuery).Limit(limit).Offset(offset).Find(&articles).Error; err != nil {
		return nil, err
	}
	return articles, nil
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

func (repository GormRepository) GetFavoritedArticleIDs(userId uint, articleIds []uint) (map[uint]bool, error) {
	var favorites []FavoriteModel
	if err := repository.db.Where("user_id = ? AND article_id IN ?", userId, articleIds).Find(&favorites).Error; err != nil {
		return nil, err
	}
	result := make(map[uint]bool)
	for _, favorite := range favorites {
		result[favorite.ArticleId] = true
	}
	return result, nil
}

func (repository GormRepository) CountFavorites(articleId uint) (int64, error) {
	var count int64
	if err := repository.db.Model(&FavoriteModel{}).Where("article_id = ?", articleId).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (repository GormRepository) CountFavoritesByArticleIDs(articleIds []uint) (map[uint]int64, error) {
	var favorites []FavoriteModel
	if err := repository.db.Where("article_id IN ?", articleIds).Find(&favorites).Error; err != nil {
		return nil, err
	}
	counts := make(map[uint]int64)
	for _, favorite := range favorites {
		counts[favorite.ArticleId]++
	}
	return counts, nil
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
