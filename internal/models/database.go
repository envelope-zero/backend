package models

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// DB is the database used by the backend
var DB *gorm.DB

// ConnectDatabase connects to the database DB
func ConnectDatabase() error {
	db, err := gorm.Open(sqlite.Open("data/gorm.db"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database!")
	}

	err = db.AutoMigrate(Budget{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(Account{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(Category{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(Envelope{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(Transaction{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(Allocation{})
	if err != nil {
		return err
	}

	DB = db
	return nil
}
