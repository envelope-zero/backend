package models_test

import (
	"log"
	"os"
	"testing"

	"github.com/envelope-zero/backend/internal/models"
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

	err := models.ConnectDatabase()
	if err != nil {
		log.Fatalf("Database migration failed with: %s", err.Error())
	}

	return m.Run()
}
