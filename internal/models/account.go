package models

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Account represents an asset account, e.g. a bank account
type Account struct {
	Model
	Name     string          `json:"name,omitempty"`
	BudgetID int             `json:"budgetId"`
	Budget   Budget          `json:"-"`
	OnBudget bool            `json:"onBudget"`
	Visible  bool            `json:"visible"`
	Note     string          `json:"note,omitempty"`
	Balance  decimal.Decimal `json:"balance" gorm:"-"`
}

// WithBalance returns a pointer to the account with the balance calculated
func (a Account) WithBalance() (*Account, error) {
	var err error
	a.Balance, err = a.getBalance()

	if err != nil {
		return nil, err
	}

	return &a, nil
}

// Transactions returns all transactions for this account
func (a Account) Transactions() []Transaction {
	var transactions []Transaction

	// Get all transactions where the account is either the source or the destination
	DB.Where(
		"source_account_id = ?", a.ID,
	).Or("destination_account_id = ?", a.ID).Find(&transactions)

	return transactions
}

// GetBalance returns the balance of the account, including all transactions.
//
// Note that this will produce wrong results with sqlite as of now, see
// https://github.com/go-gorm/gorm/issues/5153 for details.
func (a Account) getBalance() (decimal.Decimal, error) {
	var sourceSum, destinationSum decimal.NullDecimal

	err := DB.Table("transactions").Where("source_account_id = ?", a.ID).Select(
		"SUM(amount)",
	).Row().Scan(&sourceSum)
	if err != nil {
		return decimal.NewFromFloat(0.0), fmt.Errorf("getting transactions for account with id %d (source) failed: %w", a.ID, err)
	}

	err = DB.Table("transactions").Where("destination_account_id = ?", a.ID).Select(
		"SUM(amount)",
	).Row().Scan(&destinationSum)
	if err != nil {
		return decimal.NewFromFloat(0.0), fmt.Errorf("getting transactions for account with id %d (destination) failed: %w", a.ID, err)
	}

	return destinationSum.Decimal.Sub(sourceSum.Decimal), nil
}
