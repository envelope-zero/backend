package models_test

import (
	"context"
	"log"
	"net/url"
	"os"
	"testing"

	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type TestSuiteStandard struct {
	suite.Suite
	db *gorm.DB
}

// Pseudo-Test run by go test that runs the test suite.
func TestSuite(t *testing.T) {
	suite.Run(t, new(TestSuiteStandard))
}

func (suite *TestSuiteStandard) SetupSuite() {
	os.Setenv("LOG_FORMAT", "human")
	os.Setenv("GIN_MODE", "debug")
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

	err = models.Migrate(db)
	if err != nil {
		log.Fatalf("Database migration failed with: %s", err)
	}

	// Create the context and store the API URL
	ctx := context.Background()
	url, _ := url.Parse("https://example.com")
	ctx = context.WithValue(ctx, database.ContextURL, url)

	suite.db = db.WithContext(ctx)
}

// CloseDB closes the database connection. This enables testing the handling
// of database errors.
func (suite *TestSuiteStandard) CloseDB() {
	sqlDB, err := suite.db.DB()
	if err != nil {
		suite.Assert().FailNowf("Failed to get database resource for teardown: %v", err.Error())
	}
	sqlDB.Close()
}

func (suite *TestSuiteStandard) createTestBudget(c models.BudgetCreate) models.Budget {
	budget := models.Budget{
		BudgetCreate: c,
	}
	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().FailNow("Budget could not be saved", "Error: %s, Budget: %#v", err, budget)
	}

	return budget
}

func (suite *TestSuiteStandard) createTestCategory(c models.CategoryCreate) models.Category {
	category := models.Category{
		CategoryCreate: c,
	}
	err := suite.db.Save(&category).Error
	if err != nil {
		suite.Assert().FailNow("category could not be saved", "Error: %s, Budget: %#v", err, category)
	}

	return category
}

func (suite *TestSuiteStandard) createTestEnvelope(c models.EnvelopeCreate) models.Envelope {
	envelope := models.Envelope{
		EnvelopeCreate: c,
	}
	err := suite.db.Save(&envelope).Error
	if err != nil {
		suite.Assert().FailNow("Envelope could not be saved", "Error: %s, Envelope: %#v", err, envelope)
	}

	return envelope
}

func (suite *TestSuiteStandard) createTestMonthConfig(m models.MonthConfig) models.MonthConfig {
	err := suite.db.Save(&m).Error
	if err != nil {
		suite.Assert().FailNow("MonthConfig could not be saved", "Error: %s, MonthConfig: %#v", err, m)
	}

	return m
}

func (suite *TestSuiteStandard) createTestAccount(c models.AccountCreate) models.Account {
	if c.Name == "" {
		c.Name = uuid.New().String()
	}

	account := models.Account{
		AccountCreate: c,
	}
	err := suite.db.Save(&account).Error
	if err != nil {
		suite.Assert().FailNow("Account could not be saved", "Error: %s, Account: %#v", err, account)
	}

	return account
}

func (suite *TestSuiteStandard) createTestAllocation(c models.AllocationCreate) models.Allocation {
	allocation := models.Allocation{
		AllocationCreate: c,
	}
	err := suite.db.Save(&allocation).Error
	if err != nil {
		suite.Assert().FailNow("Allocation could not be saved", "Error: %s, Allocation: %#v", err, allocation)
	}

	return allocation
}

func (suite *TestSuiteStandard) createTestTransaction(c models.TransactionCreate) models.Transaction {
	transaction := models.Transaction{
		TransactionCreate: c,
	}
	err := suite.db.Save(&transaction).Error
	if err != nil {
		suite.Assert().FailNow("Transaction could not be saved", "Error: %s, Transaction: %#v", err, transaction)
	}

	return transaction
}
