package database

import (
	"github.com/envelope-zero/backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// DB is the database used by the backend
var DB *gorm.DB

// ConnectDatabase connects to the database DB
func ConnectDatabase() {
	db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})

	if err != nil {
		panic("Failed to connect to database!")
	}

	db.AutoMigrate(&models.Budget{})
	db.AutoMigrate(&models.Account{})

	DB = db
}
