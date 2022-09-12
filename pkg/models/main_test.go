package models_test

import (
	"log"
	"os"
	"testing"

	"github.com/envelope-zero/backend/pkg/database"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/suite"
)

// Environment for the test suite. Used to save the database connection.
type TestSuiteStandard struct {
	suite.Suite
}

// Pseudo-Test run by go test that runs the test suite.
func TestSuite(t *testing.T) {
	suite.Run(t, new(TestSuiteStandard))
}

func (suite *TestSuiteStandard) SetupSuite() {
	os.Setenv("LOG_FORMAT", "human")
	os.Setenv("GIN_MODE", "debug")
	os.Setenv("API_URL", "http://example.com")
}

// TearDownTest is called after each test in the suite.
func (suite *TestSuiteStandard) TearDownTest() {
	sqlDB, _ := database.DB.DB()
	sqlDB.Close()
}

// SetupTest is called before each test in the suite.
func (suite *TestSuiteStandard) SetupTest() {
	err := database.ConnectDatabase(sqlite.Open, ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		log.Fatalf("Database connection failed with: %s", err.Error())
	}

	// Migrate all models so that the schema is correct
	err = database.DB.AutoMigrate(models.Budget{}, models.Account{}, models.Category{}, models.Envelope{}, models.Transaction{}, models.Allocation{})
	if err != nil {
		log.Fatalf("Database migration failed with: %s", err.Error())
	}
}
