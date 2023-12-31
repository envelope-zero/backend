package controllers_test

import (
	"context"
	"log"
	"net/url"
	"os"
	"testing"

	"github.com/envelope-zero/backend/v4/pkg/controllers"
	"github.com/envelope-zero/backend/v4/pkg/database"
	"github.com/envelope-zero/backend/v4/pkg/models"
	"github.com/stretchr/testify/suite"
)

type TestSuiteStandard struct {
	suite.Suite
	controller controllers.Controller
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

	// Create the context and store the API URL
	ctx := context.Background()
	url, _ := url.Parse("http://example.com")
	ctx = context.WithValue(ctx, database.ContextURL, url)

	suite.controller = controllers.Controller{db.WithContext(ctx)}
}

// CloseDB closes the database connection. This enables testing the handling
// of database errors.
func (suite *TestSuiteStandard) CloseDB() {
	sqlDB, err := suite.controller.DB.DB()
	if err != nil {
		suite.Assert().FailNowf("Failed to get database resource for teardown: %v", err.Error())
	}
	sqlDB.Close()
}
