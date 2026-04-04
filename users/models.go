package users

import "gorm.io/gorm"

type UserModel struct {
    gorm.Model
    Username string `gorm:"uniqueIndex"`
    Email    string `gorm:"uniqueIndex"`
    Password string 
}

func AutoMigrate(db *gorm.DB) {
    db.AutoMigrate(&UserModel{})
}