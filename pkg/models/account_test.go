package models_test

import (
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteStandard) TestAccountCalculations() {
	budget := models.Budget{}
	err := suite.db.Save(&budget).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	account := models.Account{
		AccountCreate: models.AccountCreate{
			BudgetID:       budget.ID,
			OnBudget:       true,
			External:       false,
			InitialBalance: decimal.NewFromFloat(170),
		},
	}
	err = suite.db.Save(&account).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	externalAccount := models.Account{
		AccountCreate: models.AccountCreate{
			BudgetID: budget.ID,
			External: true,
		},
	}
	err = suite.db.Save(&externalAccount).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	category := models.Category{
		CategoryCreate: models.CategoryCreate{
			BudgetID: budget.ID,
		},
	}
	err = suite.db.Save(&category).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	envelope := models.Envelope{
		EnvelopeCreate: models.EnvelopeCreate{
			CategoryID: category.ID,
		},
	}
	err = suite.db.Save(&envelope).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	incomingTransaction := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.ID,
			EnvelopeID:           &envelope.ID,
			SourceAccountID:      externalAccount.ID,
			DestinationAccountID: account.ID,
			Reconciled:           true,
			Amount:               decimal.NewFromFloat(32.17),
		},
	}
	err = suite.db.Save(&incomingTransaction).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	outgoingTransaction := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID:             budget.ID,
			EnvelopeID:           &envelope.ID,
			SourceAccountID:      account.ID,
			DestinationAccountID: externalAccount.ID,
			Amount:               decimal.NewFromFloat(17.45),
		},
	}
	err = suite.db.Save(&outgoingTransaction).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be saved", err)
	}

	a := account.WithCalculations(suite.db)

	assert.True(suite.T(), a.Balance.Equal(incomingTransaction.Amount.Sub(outgoingTransaction.Amount).Add(a.InitialBalance)), "Balance for account is not correct. Should be: %v but is %v", incomingTransaction.Amount.Sub(outgoingTransaction.Amount), a.Balance)

	assert.True(suite.T(), a.ReconciledBalance.Equal(incomingTransaction.Amount.Add(a.InitialBalance)), "Reconciled balance for account is not correct. Should be: %v but is %v", incomingTransaction.Amount, a.ReconciledBalance)

	err = suite.db.Delete(&incomingTransaction).Error
	if err != nil {
		suite.Assert().Fail("Resource could not be deleted", err)
	}

	a = account.WithCalculations(suite.db)
	assert.True(suite.T(), a.Balance.Equal(outgoingTransaction.Amount.Neg().Add(a.InitialBalance)), "Balance for account is not correct. Should be: %v but is %v", outgoingTransaction.Amount.Neg(), a.Balance)
	assert.True(suite.T(), a.ReconciledBalance.Equal(decimal.NewFromFloat(0).Add(a.InitialBalance)), "Reconciled balance for account is not correct. Should be: %v but is %v", decimal.NewFromFloat(0), a.ReconciledBalance)
}

func (suite *TestSuiteStandard) TestAccountTransactions() {
	account := models.Account{}

	transactions := account.Transactions(suite.db)
	assert.Len(suite.T(), transactions, 0)
}

func (suite *TestSuiteStandard) TestAccountOnBudget() {
	account := models.Account{
		AccountCreate: models.AccountCreate{
			OnBudget: true,
			External: true,
		},
	}

	err := account.BeforeSave(suite.db)
	if err != nil {
		assert.Fail(suite.T(), "account.BeforeSave failed")
	}

	assert.False(suite.T(), account.OnBudget, "OnBudget is true even though the account is external")

	account = models.Account{
		AccountCreate: models.AccountCreate{
			OnBudget: true,
			External: false,
		},
	}

	err = account.BeforeSave(suite.db)
	if err != nil {
		assert.Fail(suite.T(), "account.BeforeSave failed")
	}

	assert.True(suite.T(), account.OnBudget, "OnBudget is set to false even though the account is internal")
}
