package models_test

import (
	"testing"

	"github.com/envelope-zero/backend/internal/models"
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
	models.DB.Save(&account)

	externalAccount := models.Account{
		AccountCreate: models.AccountCreate{
			External: true,
		},
	}
	models.DB.Save(&externalAccount)

	incomingTransaction := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			SourceAccountID:      externalAccount.ID,
			DestinationAccountID: account.ID,
			Reconciled:           true,
			Amount:               decimal.NewFromFloat(32.17),
		},
	}
	models.DB.Save(&incomingTransaction)

	outgoingTransaction := models.Transaction{
		TransactionCreate: models.TransactionCreate{
			SourceAccountID:      account.ID,
			DestinationAccountID: externalAccount.ID,
			Amount:               decimal.NewFromFloat(17.45),
		},
	}
	models.DB.Save(&outgoingTransaction)

	a := account.WithCalculations()

	assert.True(t, a.Balance.Equal(incomingTransaction.Amount.Sub(outgoingTransaction.Amount)), "Balance for account is not correct. Should be: %v but is %v", incomingTransaction.Amount.Sub(outgoingTransaction.Amount), a.Balance)

	assert.True(t, a.ReconciledBalance.Equal(incomingTransaction.Amount), "Reconciled balance for account is not correct. Should be: %v but is %v", incomingTransaction.Amount, a.ReconciledBalance)

	models.DB.Delete(&account)
	models.DB.Delete(&externalAccount)
	models.DB.Delete(&incomingTransaction)
	models.DB.Delete(&outgoingTransaction)
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

	err := account.BeforeSave(models.DB)
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

	err = account.BeforeSave(models.DB)
	if err != nil {
		assert.Fail(t, "account.BeforeSave failed")
	}

	assert.True(t, account.OnBudget, "OnBudget is set to false even though the account is internal")
}
