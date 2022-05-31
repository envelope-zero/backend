package models_test

import (
	"testing"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestAccountCalculations(t *testing.T) {
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

	assert.True(t, a.Balance.Equal(incomingTransaction.Amount.Sub(outgoingTransaction.Amount)), "Balance for account is not correct. Should be: %v but is %v", incomingTransaction.Amount.Sub(outgoingTransaction.Amount), a.Balance)

	assert.True(t, a.ReconciledBalance.Equal(incomingTransaction.Amount), "Reconciled balance for account is not correct. Should be: %v but is %v", incomingTransaction.Amount, a.ReconciledBalance)

	database.DB.Delete(&account)
	database.DB.Delete(&externalAccount)
	database.DB.Delete(&incomingTransaction)
	database.DB.Delete(&outgoingTransaction)
}

func TestAccountTransactions(t *testing.T) {
	account := models.Account{}

	transactions := account.Transactions()
	assert.Len(t, transactions, 0)
}

func TestAccountOnBudget(t *testing.T) {
	account := models.Account{
		AccountCreate: models.AccountCreate{
			OnBudget: true,
			External: true,
		},
	}

	err := account.BeforeSave(database.DB)
	if err != nil {
		assert.Fail(t, "account.BeforeSave failed")
	}

	assert.False(t, account.OnBudget, "OnBudget is true even though the account is external")

	account = models.Account{
		AccountCreate: models.AccountCreate{
			OnBudget: true,
			External: false,
		},
	}

	err = account.BeforeSave(database.DB)
	if err != nil {
		assert.Fail(t, "account.BeforeSave failed")
	}

	assert.True(t, account.OnBudget, "OnBudget is set to false even though the account is internal")
}
