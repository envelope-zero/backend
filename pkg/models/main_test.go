package models_test

import (
	"log"
	"os"
	"testing"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/gin-gonic/gin"
)

// TestMain takes care of the test setup for this package.
func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	// Always remove the DB after running tests
	defer os.Remove("data/gorm.db")

	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		gin.SetMode("release")
	}

	err := database.ConnectDatabase()
	if err != nil {
		log.Fatalf("Database connection failed with: %s", err.Error())
	}

	// Migrate all models so that the schema is correct
	err = database.DB.AutoMigrate(models.Budget{}, models.Account{}, models.Category{}, models.Envelope{}, models.Transaction{}, models.Allocation{})
	if err != nil {
		log.Fatalf("Database migration failed with: %s", err.Error())
	}

	return m.Run()
}
