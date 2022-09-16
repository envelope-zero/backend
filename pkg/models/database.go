package models

import (
	"fmt"

	"gorm.io/gorm"
)

// Migrate migrates all models to the schema defined in the code.
func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(Budget{}, Account{}, Category{}, Envelope{}, Transaction{}, Allocation{})
	if err != nil {
		return fmt.Errorf("error during DB migration: %w", err)
	}

	return nil
}
