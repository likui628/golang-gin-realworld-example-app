package articles

import "gorm.io/gorm"

type ArticleRepository interface {
	Create(article *ArticleModel) error
	FindOrCreateTags(tags []string) ([]TagModel, error)
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
