package models

import (
	"github.com/envelope-zero/backend/pkg/database"
)

// MigrateDatabase migrates all models.
func MigrateDatabase() error {
	// Migrate all models so that the schema is correct
	err := database.DB.AutoMigrate(Budget{}, Account{}, Category{}, Envelope{}, Transaction{}, Allocation{})
	if err != nil {
		return err
	}

	return nil
}
