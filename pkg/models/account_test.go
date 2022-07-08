package models_test

import (
	"github.com/envelope-zero/backend/internal/database"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func (suite *TestSuiteEnv) TestAccountCalculations() {
	account := models.Account{
		AccountCreate: models.AccountCreate{
			OnBudget: true,
			External: false,
		},
	}
	database.DB.Save(&account)

	externalAccount := models.Account{
		AccountCreate: models.AccountCreate{
			External: true,
		},
	}
	database.DB.Save(&externalAccount)

	incomingTransaction := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			SourceAccountID:      externalAccount.ID,
			DestinationAccountID: account.ID,
			Reconciled:           true,
			Amount:               decimal.NewFromFloat(32.17),
		},
	}
	database.DB.Save(&incomingTransaction)

	outgoingTransaction := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			SourceAccountID:      account.ID,
			DestinationAccountID: externalAccount.ID,
			Amount:               decimal.NewFromFloat(17.45),
		},
	}
	database.DB.Save(&outgoingTransaction)

	a := account.WithCalculations()

	assert.True(suite.T(), a.Balance.Equal(incomingTransaction.Amount.Sub(outgoingTransaction.Amount)), "Balance for account is not correct. Should be: %v but is %v", incomingTransaction.Amount.Sub(outgoingTransaction.Amount), a.Balance)

	assert.True(suite.T(), a.ReconciledBalance.Equal(incomingTransaction.Amount), "Reconciled balance for account is not correct. Should be: %v but is %v", incomingTransaction.Amount, a.ReconciledBalance)

	database.DB.Delete(&account)
	database.DB.Delete(&externalAccount)
	database.DB.Delete(&incomingTransaction)
	database.DB.Delete(&outgoingTransaction)
}

func (suite *TestSuiteEnv) TestAccountTransactions() {
	account := models.Account{}

	transactions := account.Transactions()
	assert.Len(suite.T(), transactions, 0)
}

func (suite *TestSuiteEnv) TestAccountOnBudget() {
	account := models.Account{
		AccountCreate: models.AccountCreate{
			OnBudget: true,
			External: true,
		},
	}

	err := account.BeforeSave(database.DB)
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

	err = account.BeforeSave(database.DB)
	if err != nil {
		assert.Fail(suite.T(), "account.BeforeSave failed")
	}

	assert.True(suite.T(), account.OnBudget, "OnBudget is set to false even though the account is internal")
}
