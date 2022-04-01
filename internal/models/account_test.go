package models_test

import (
	"testing"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestAccountBalance(t *testing.T) {
	account := models.Account{}

	_, err := account.WithBalance()

	assert.Nil(t, err)

	if !decimal.NewFromFloat(0).Equal(account.Balance) {
		assert.Fail(t, "Account balance is not 0", "Actual balance: %v", account.Balance)
	}
}

func TestAccountTransactions(t *testing.T) {
	account := models.Account{}

	transactions := account.Transactions()
	assert.Len(t, transactions, 0)
}

func TestAccountOnBudget(t *testing.T) {
	account := models.Account{
		OnBudget: true,
		External: true,
	}

	err := account.BeforeSave(models.DB)
	if err != nil {
		assert.Fail(t, "account.BeforeSave failed")
	}

	assert.False(t, account.OnBudget, "OnBudget is true even though the account is external")

	account = models.Account{
		OnBudget: true,
		External: false,
	}

	err = account.BeforeSave(models.DB)
	if err != nil {
		assert.Fail(t, "account.BeforeSave failed")
	}

	assert.True(t, account.OnBudget, "OnBudget is set to false even though the account is internal")
}
