package controllers_test

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
func TestStandard(t *testing.T) {
	suite.Run(t, new(TestSuiteStandard))
}

func (suite *TestSuiteStandard) SetupSuite() {
	os.Setenv("LOG_FORMAT", "human")
	os.Setenv("GIN_MODE", "debug")
	os.Setenv("API_URL", "http://example.com")
}

// TearDownTest is called after each test in the suite.
func (suite *TestSuiteStandard) TearDownTest() {
	sqlDB, err := database.DB.DB()
	if err != nil {
		log.Fatalf("Database connection for teardown failed with: %s", err.Error())
	}
	sqlDB.Close()
}

// SetupTest is called before each test in the suite.
func (suite *TestSuiteStandard) SetupTest() {
	err := database.ConnectDatabase(sqlite.Open, ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		log.Fatalf("Database initialization failed with: %s", err.Error())
	}

	// Migrate all models so that the schema is correct
	err = database.DB.AutoMigrate(models.Budget{}, models.Account{}, models.Category{}, models.Envelope{}, models.Transaction{}, models.Allocation{})
	if err != nil {
		log.Fatalf("Database migration failed with: %s", err.Error())
	}
}

// TestSuiteClosedDB is used for tests against an already
// closed database connection.
type TestSuiteClosedDB struct {
	suite.Suite
}

// Pseudo-Test run by go test that runs the test suite.
func TestClosedDB(t *testing.T) {
	suite.Run(t, new(TestSuiteClosedDB))
}

func (suite *TestSuiteClosedDB) SetupSuite() {
	os.Setenv("LOG_FORMAT", "human")
	os.Setenv("GIN_MODE", "debug")
	os.Setenv("API_URL", "http://example.com")
}

// SetupTest is called before each test in the suite.
func (suite *TestSuiteClosedDB) SetupTest() {
	err := database.ConnectDatabase(sqlite.Open, ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		log.Fatalf("Database initialization failed with: %s", err.Error())
	}

	sqlDB, err := database.DB.DB()
	if err != nil {
		log.Fatalf("Database connection failed with: %s", err.Error())
	}
	sqlDB.Close()
}

// TearDownTest is called after each test in the suite.
func (suite *TestSuiteClosedDB) TearDownTest() {
	sqlDB, err := database.DB.DB()
	if err != nil {
		log.Fatalf("Database connection for teardown failed with: %s", err.Error())
	}
	sqlDB.Close()
}
