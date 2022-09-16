package models_test

import (
	"log"
	"os"
	"testing"

	"github.com/envelope-zero/backend/pkg/database"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// Environment for the test suite. Used to save the database connection.
type TestSuiteType struct {
	suite.Suite
	db *gorm.DB
}

type TestSuiteStandard TestSuiteType

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
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
}

// SetupTest is called before each test in the suite.
func (suite *TestSuiteStandard) SetupTest() {
	db, err := database.Connect(":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		log.Fatalf("Database connection failed with: %#v", err)
	}

	models.Migrate(db)
	if err != nil {
		log.Fatalf("Database migration failed with: %#v", err)
	}

	suite.db = db
}

type TestSuiteClosedDB TestSuiteType

// Pseudo-Test run by go test that runs the test suite.
func TestSuiteClosed(t *testing.T) {
	suite.Run(t, new(TestSuiteClosedDB))
}

func (suite *TestSuiteClosedDB) SetupSuite() {
	os.Setenv("LOG_FORMAT", "human")
	os.Setenv("GIN_MODE", "debug")
	os.Setenv("API_URL", "http://example.com")
}

// TearDownTest is called after each test in the suite.
func (suite *TestSuiteClosedDB) TearDownTest() {
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
}

// SetupTest is called before each test in the suite.
func (suite *TestSuiteClosedDB) SetupTest() {
	db, err := database.Connect(":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		log.Fatalf("Database connection failed with: %#v", err)
	}

	suite.db = db

	sqlDB, err := suite.db.DB()
	if err != nil {
		log.Fatalf("Database connection failed with: %#v", err)
	}
	sqlDB.Close()
}
