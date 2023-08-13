package models_test

import (
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/shopspring/decimal"
)

func (suite *TestSuiteStandard) TestMigrate() {
	suite.CloseDB()
	err := models.Migrate(suite.db)
	suite.Assert().NotNil(err)
	suite.Assert().Contains(err.Error(), "error during DB migration")
}

func (suite *TestSuiteStandard) TestMigrateWithExistingDB() {
	// Initialize the database to have all tables
	err := suite.db.AutoMigrate()
	suite.Assert().Nil(err, err)

	// Execute the migration again
	err = models.Migrate(suite.db)
	suite.Assert().Nil(err)
}

func (suite *TestSuiteStandard) TestMigrateDuplicateAccountNames() {
	// Initialize the database to have all tables
	err := suite.db.AutoMigrate()
	suite.Assert().Nil(err, err)

	// Drop the unique constraint so that we can add non-unique account names
	err = suite.db.Migrator().DropConstraint(&models.Account{}, "account_name_budget_id")
	suite.Assert().Nil(err, err)

	name := "Non-unique name"
	budget := suite.createTestBudget(models.BudgetCreate{})
	_ = suite.createTestAccount(models.AccountCreate{
		BudgetID: budget.ID,
		Name:     name,
	})

	_ = suite.createTestAccount(models.AccountCreate{
		BudgetID: budget.ID,
		Name:     name,
	})

	// Execute the migration again
	err = models.Migrate(suite.db)
	suite.Assert().Nil(err)
}

func (suite *TestSuiteStandard) TestUnsetEnvelope() {
	// Initialize the database to have all tables
	err := suite.db.AutoMigrate()
	suite.Assert().Nil(err, err)

	budget := suite.createTestBudget(models.BudgetCreate{})
	sourceAccount := suite.createTestAccount(models.AccountCreate{
		BudgetID: budget.ID,
		Name:     "TestUnsetEnvelope: Source",
		OnBudget: true,
	})

	destinationAccount := suite.createTestAccount(models.AccountCreate{
		BudgetID: budget.ID,
		Name:     "TestUnsetEnvelope: Destination",
		OnBudget: true,
	})

	envelope := suite.createTestEnvelope(models.EnvelopeCreate{
		CategoryID: suite.createTestCategory(models.CategoryCreate{BudgetID: budget.ID}).ID,
		Name:       "TestUnsetEnvelope: Envelope",
	})

	transaction := suite.createTestTransaction(models.TransactionCreate{
		BudgetID:             budget.ID,
		SourceAccountID:      sourceAccount.ID,
		DestinationAccountID: destinationAccount.ID,
		Amount:               decimal.NewFromFloat(17.36),
		Note:                 "This can only be created for this test - the controllers prevent creating this already",
		EnvelopeID:           &envelope.ID,
	})

	// Execute the migration again
	err = models.Migrate(suite.db)
	suite.Assert().Nil(err)

	// Reload the transaction
	var checkTransaction models.Transaction
	err = suite.db.First(&checkTransaction, transaction.ID).Error
	suite.Assert().Nil(err)

	// Test thet the envelope has been set to nil by the migration
	suite.Assert().Nil(checkTransaction.EnvelopeID)
}
