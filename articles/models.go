package articles

import (
	"time"

	"github.com/likui628/golang-gin-realworld-example-app/users"
	"gorm.io/gorm"
)

type ArticleModel struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	Slug        string `gorm:"uniqueIndex;not null"`
	Title       string `gorm:"size:255;not null"`
	Description string `gorm:"size:2048"`
	Body        string `gorm:"type:text"`

	AuthorId uint            `gorm:"not null"`
	Author   users.UserModel `gorm:"foreignKey:AuthorId"`

	Tags []TagModel `gorm:"many2many:article_tags;"`
}

type TagModel struct {
	ID  uint   `gorm:"primaryKey"`
	Tag string `gorm:"size:255;uniqueIndex;not null"`
}

func AutoMigrate(db *gorm.DB) {
	db.AutoMigrate(&ArticleModel{})
	db.AutoMigrate(&TagModel{})
}
