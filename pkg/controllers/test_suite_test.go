package controllers_test

import (
	"log"
	"os"
	"testing"

	"github.com/envelope-zero/backend/pkg/controllers"
	"github.com/envelope-zero/backend/pkg/database"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/stretchr/testify/suite"
)

type TestSuiteType struct {
	suite.Suite
	controller controllers.Controller
}

// Environment for the test suite. Used to save the database connection.
type TestSuiteStandard TestSuiteType

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
	sqlDB, err := suite.controller.DB.DB()
	if err != nil {
		log.Fatalf("Database connection for teardown failed with: %#v", err)
	}
	sqlDB.Close()
}

// SetupTest is called before each test in the suite.
func (suite *TestSuiteStandard) SetupTest() {
	db, err := database.Connect(":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		log.Fatalf("Database initialization failed with: %#v", err)
	}

	models.Migrate(db)
	if err != nil {
		log.Fatalf("Database migration failed with: %#v", err)
	}

	suite.controller = controllers.Controller{db}
}

// TestSuiteClosedDB is used for tests against an already
// closed database connection.
type TestSuiteClosedDB TestSuiteType

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
	db, err := database.Connect(":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		log.Fatalf("Database initialization failed with: %#v", err)
	}

	models.Migrate(db)
	if err != nil {
		log.Fatalf("Database migration failed with: %#v", err)
	}

	suite.controller = controllers.Controller{db}

	sqlDB, err := suite.controller.DB.DB()
	if err != nil {
		log.Fatalf("Database connection failed with: %#v", err)
	}
	sqlDB.Close()
}

// TearDownTest is called after each test in the suite.
func (suite *TestSuiteClosedDB) TearDownTest() {
	sqlDB, err := suite.controller.DB.DB()
	if err != nil {
		log.Fatalf("Database connection for teardown failed with: %#v", err)
	}
	sqlDB.Close()
}
