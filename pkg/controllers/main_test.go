package controllers_test

import (
	"log"
	"os"
	"testing"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/suite"
)

// Environment for the test suite. Used to save the database connection.
type TestSuiteEnv struct {
	suite.Suite
}

// Pseudo-Test run by go test that runs the test suite.
func TestSuite(t *testing.T) {
	suite.Run(t, new(TestSuiteEnv))
}

func (suite *TestSuiteEnv) SetupSuite() {
	os.Setenv("LOG_FORMAT", "human")
	os.Setenv("GIN_MODE", "debug")
	os.Setenv("API_HOST_PROTOCOL", "http://example.com")
}

// TearDownTest is called after each test in the suite.
func (suite *TestSuiteEnv) TearDownTest() {
	sqlDB, _ := database.DB.DB()
	sqlDB.Close()
}

// SetupTest is called before each test in the suite.
func (suite *TestSuiteEnv) SetupTest() {
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
