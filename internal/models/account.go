package models

import (
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
	DB.Where(Transaction{SourceAccountID: a.ID}).Or(Transaction{DestinationAccountID: a.ID}).Find(&transactions)
	return transactions
}

// Transactions returns all transactions for this account.
func (a Account) SumReconciledTransactions() (decimal.Decimal, error) {
	return TransactionsSum(
		Transaction{
			Reconciled:           true,
			DestinationAccountID: a.ID,
		},
		Transaction{
			Reconciled:      true,
			SourceAccountID: a.ID,
		},
	)
}

// GetBalance returns the balance of the account, including all transactions.
func (a Account) getBalance() (decimal.Decimal, error) {
	return TransactionsSum(
		Transaction{DestinationAccountID: a.ID},
		Transaction{SourceAccountID: a.ID},
	)
}
