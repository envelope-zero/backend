package models_test

import (
	"log"
	"os"
	"testing"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
)

// TestMain takes care of the test setup for this package.
func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		gin.SetMode("release")
	}

	err := database.ConnectDatabase(sqlite.Open, ":memory:")
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
