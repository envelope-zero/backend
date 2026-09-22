package models_test

import (
	"log"
	"os"
	"testing"

	"github.com/envelope-zero/backend/v7/internal/models"
	"github.com/envelope-zero/backend/v7/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

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
}

// TearDownTest is called after each test in the suite.
func (suite *TestSuiteStandard) TearDownTest() {
	sqlDB, _ := models.DB.DB()
	sqlDB.Close()
}

// SetupTest is called before each test in the suite.
func (suite *TestSuiteStandard) SetupTest() {
	err := models.Connect(test.TmpFile(suite.T()))
	if err != nil {
		log.Fatalf("Database connection failed with: %#v", err)
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

func (suite *TestSuiteStandard) createTestBudget(budget models.Budget) models.Budget {
	err := models.DB.Create(&budget).Error
	if err != nil {
		suite.Assert().FailNow("Budget could not be saved", "Error: %s, Budget: %#v", err, budget)
	}

	return budget
}

func (suite *TestSuiteStandard) createTestCategory(category models.Category) models.Category {
	err := models.DB.Create(&category).Error
	if err != nil {
		suite.Assert().FailNow("category could not be saved", "Error: %s, Budget: %#v", err, category)
	}

	return category
}

func (suite *TestSuiteStandard) createTestEnvelope(envelope models.Envelope) models.Envelope {
	err := models.DB.Create(&envelope).Error
	if err != nil {
		suite.Assert().FailNow("Envelope could not be saved", "Error: %s, Envelope: %#v", err, envelope)
	}

	return envelope
}

func (suite *TestSuiteStandard) createTestAccount(account models.Account) models.Account {
	if account.Name == "" {
		account.Name = uuid.New().String()
	}

	err := models.DB.Create(&account).Error
	if err != nil {
		suite.Assert().FailNow("Account could not be saved", "Error: %s, Account: %#v", err, account)
	}

	return account
}

func (suite *TestSuiteStandard) createTestMatchRule(matchRule models.MatchRule) models.MatchRule {
	err := models.DB.Create(&matchRule).Error
	if err != nil {
		suite.Assert().FailNow("MatchRule could not be saved", "Error: %s, MatchRule: %#v", err, matchRule)
	}

	return matchRule
}

func (suite *TestSuiteStandard) createTestTransaction(transaction models.Transaction) models.Transaction {
	err := models.DB.Create(&transaction).Error
	if err != nil {
		suite.Assert().FailNow("Transaction could not be saved", "Error: %s, Transaction: %#v", err, transaction)
	}

	return transaction
}

func (suite *TestSuiteStandard) createTestMonthConfig(monthConfig models.MonthConfig) models.MonthConfig {
	err := models.DB.Create(&monthConfig).Error
	if err != nil {
		suite.Assert().FailNow("MonthConfig could not be saved", "Error: %s, Transaction: %#v", err, monthConfig)
	}

	return monthConfig
}

func (suite *TestSuiteStandard) createTestGoal(goal models.Goal) models.Goal {
	err := models.DB.Create(&goal).Error
	if err != nil {
		suite.Assert().FailNow("Goal could not be saved", "Error: %s, Goal: %#v", err, goal)
	}

	return goal
}
