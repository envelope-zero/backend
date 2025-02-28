package v4_test

import (
	"log"
	"os"
	"testing"

	"github.com/envelope-zero/backend/v7/internal/models"
	"github.com/envelope-zero/backend/v7/test"
	"github.com/stretchr/testify/suite"
)

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
	sqlDB, err := models.DB.DB()
	if err != nil {
		log.Fatalf("Database connection for teardown failed with: %#v", err)
	}
	sqlDB.Close()
}

// SetupTest is called before each test in the suite.
func (suite *TestSuiteStandard) SetupTest() {
	err := models.Connect(test.TmpFile(suite.T()))
	if err != nil {
		log.Fatalf("Database initialization failed with: %#v", err)
	}
}

// CloseDB closes the database connection. This enables testing the handling
// of database errors.
func (suite *TestSuiteStandard) CloseDB() {
	sqlDB, err := models.DB.DB()
	if err != nil {
		suite.Assert().FailNowf("Failed to get database resource for teardown: %v", err.Error())
	}
	sqlDB.Close()
}
