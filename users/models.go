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

type FollowModel struct {
	ID         uint      `gorm:"primaryKey"`
	FollowerId uint      `gorm:"not null;uniqueIndex:idx_follower_followed"`
	Follower   UserModel `gorm:"foreignKey:FollowerId"`
	FollowedId uint      `gorm:"not null;uniqueIndex:idx_follower_followed"`
	Followed   UserModel `gorm:"foreignKey:FollowedId"`
}

func AutoMigrate(db *gorm.DB) {
	db.AutoMigrate(&UserModel{})
	db.AutoMigrate(&FollowModel{})
}
