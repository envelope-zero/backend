package models

import (
	"fmt"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Account represents an asset account, e.g. a bank account.
type Account struct {
	Model
	Name              string          `json:"name,omitempty"`
	BudgetID          uint64          `json:"budgetId"`
	Budget            Budget          `json:"-"`
	OnBudget          bool            `json:"onBudget"` // Always false when external: true
	External          bool            `json:"external"`
	Balance           decimal.Decimal `json:"balance" gorm:"-"`
	ReconciledBalance decimal.Decimal `json:"reconciledBalance" gorm:"-"`
}

// BeforeSave sets OnBudget to false when External is true.
func (a *Account) BeforeSave(tx *gorm.DB) (err error) {
	if a.External {
		a.OnBudget = false
	}
	return nil
}

// WithCalculations returns a pointer to the account with the balance calculated.
func (a Account) WithCalculations() (*Account, error) {
	var err error
	a.Balance, err = a.getBalance()
	if err != nil {
		return nil, err
	}

	a.ReconciledBalance, err = a.SumReconciledTransactions()
	if err != nil {
		return nil, err
	}

	return &a, nil
}

// Transactions returns all transactions for this account.
func (a Account) Transactions() []Transaction {
	var transactions []Transaction

	// Get all transactions where the account is either the source or the destination
	DB.Where(
		"source_account_id = ?", a.ID,
	).Or("destination_account_id = ?", a.ID).Find(&transactions)

	return transactions
}

// Transactions returns all transactions for this account.
func (a Account) SumReconciledTransactions() (decimal.Decimal, error) {
	var sourceSum, destinationSum decimal.NullDecimal

	err := DB.Table("transactions").
		Where(&Transaction{
			Reconciled:      true,
			SourceAccountID: a.ID,
		}).
		Select("SUM(amount)").
		Row().
		Scan(&sourceSum)
	if err != nil {
		return decimal.NewFromFloat(0.0), fmt.Errorf("getting transactions for account with id %d (source) failed: %w", a.ID, err)
	}

	err = DB.Table("transactions").
		Where(&Transaction{
			Reconciled:           true,
			DestinationAccountID: a.ID,
		}).
		Select("SUM(amount)").
		Row().
		Scan(&destinationSum)

	if err != nil {
		return decimal.NewFromFloat(0.0), fmt.Errorf("getting transactions for account with id %d (destination) failed: %w", a.ID, err)
	}

	return destinationSum.Decimal.Sub(sourceSum.Decimal), nil
}

// GetBalance returns the balance of the account, including all transactions.
//
// Note that this will produce wrong results with sqlite as of now, see
// https://github.com/go-gorm/gorm/issues/5153 for details.
func (a Account) getBalance() (decimal.Decimal, error) {
	var sourceSum, destinationSum decimal.NullDecimal

	err := DB.Table("transactions").
		Where(&Transaction{SourceAccountID: a.ID}).
		Select("SUM(amount)").
		Row().
		Scan(&sourceSum)
	if err != nil {
		return decimal.NewFromFloat(0.0), fmt.Errorf("getting transactions for account with id %d (source) failed: %w", a.ID, err)
	}

	err = DB.Table("transactions").
		Where(&Transaction{DestinationAccountID: a.ID}).
		Select("SUM(amount)").
		Row().
		Scan(&destinationSum)
	if err != nil {
		return decimal.NewFromFloat(0.0), fmt.Errorf("getting transactions for account with id %d (destination) failed: %w", a.ID, err)
	}

	return destinationSum.Decimal.Sub(sourceSum.Decimal), nil
}
