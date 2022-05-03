package models

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Account represents an asset account, e.g. a bank account.
type Account struct {
	Model
	AccountCreate
	Budget            Budget          `json:"-"`
	Balance           decimal.Decimal `json:"balance" gorm:"-"`
	ReconciledBalance decimal.Decimal `json:"reconciledBalance" gorm:"-"`
}

type AccountCreate struct {
	Name     string `json:"name,omitempty"`
	Note     string `json:"note,omitempty"`
	BudgetID uint64 `json:"budgetId"`
	OnBudget bool   `json:"onBudget"` // Always false when external: true
	External bool   `json:"external"`
}

func (a Account) WithCalculations() Account {
	a.Balance = a.getBalance()
	a.ReconciledBalance = a.SumReconciledTransactions()

	return a
}

// BeforeSave sets OnBudget to false when External is true.
func (a *Account) BeforeSave(tx *gorm.DB) (err error) {
	if a.External {
		a.OnBudget = false
	}
	return nil
}

// Transactions returns all transactions for this account.
func (a Account) Transactions() []Transaction {
	var transactions []Transaction

	// Get all transactions where the account is either the source or the destination
	DB.Where(Transaction{TransactionCreate: TransactionCreate{SourceAccountID: a.ID}}).Or(Transaction{TransactionCreate: TransactionCreate{DestinationAccountID: a.ID}}).Find(&transactions)
	return transactions
}

// Transactions returns all transactions for this account.
func (a Account) SumReconciledTransactions() decimal.Decimal {
	return TransactionsSum(
		Transaction{
			TransactionCreate: TransactionCreate{
				Reconciled:           true,
				DestinationAccountID: a.ID,
			},
		},
		Transaction{
			TransactionCreate: TransactionCreate{
				Reconciled:      true,
				SourceAccountID: a.ID,
			},
		},
	)
}

// GetBalance returns the balance of the account, including all transactions.
func (a Account) getBalance() decimal.Decimal {
	return TransactionsSum(
		Transaction{
			TransactionCreate: TransactionCreate{
				DestinationAccountID: a.ID,
			},
		},
		Transaction{
			TransactionCreate: TransactionCreate{
				SourceAccountID: a.ID,
			},
		},
	)
}
