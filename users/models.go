package users

import "gorm.io/gorm"

type UserModel struct {
	ID           uint    `gorm:"primaryKey"`
	Username     string  `gorm:"column:username"`
	Email        string  `gorm:"column:email;uniqueIndex"`
	Bio          string  `gorm:"column:bio;size:1024"`
	Image        *string `gorm:"column:image"`
	PasswordHash string  `gorm:"column:password;not null"`
}

func AutoMigrate(db *gorm.DB) {
	db.AutoMigrate(&UserModel{})
}
