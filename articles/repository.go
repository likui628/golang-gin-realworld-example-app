package articles

import "gorm.io/gorm"

type ArticleRepository interface {
	Create(article *ArticleModel) error
}

type GormRepository struct {
	db *gorm.DB
}

func NewArticleRepository(db *gorm.DB) ArticleRepository {
	return GormRepository{db: db}
}

func (repository GormRepository) Create(article *ArticleModel) error {
	return repository.db.Create(article).Error
}
