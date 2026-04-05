package common

import (
	"fmt"
	"os"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func GetDBPath() string {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/gorm.db"
	}
	return dbPath
}

func InitDatabase() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(GetDBPath()), &gorm.Config{})
	if err != nil {
		panic(fmt.Errorf("failed to connect database: %w", err))
	}
	DB = db
	return db
}
