package models

import (
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

	db.AutoMigrate(Budget{})
	db.AutoMigrate(Account{})
	db.AutoMigrate(Category{})
	db.AutoMigrate(Envelope{})
	db.AutoMigrate(Transaction{})

	DB = db
}
