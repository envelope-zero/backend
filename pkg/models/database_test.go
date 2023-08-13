package models_test

import (
	"github.com/envelope-zero/backend/v3/pkg/models"
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
