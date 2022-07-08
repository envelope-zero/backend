package models

import (
	"github.com/envelope-zero/backend/internal/database"
	"github.com/google/uuid"
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
	Name     string    `json:"name,omitempty" example:"Checking"`
	Note     string    `json:"note,omitempty" example:"My bank account"`
	BudgetID uuid.UUID `json:"budgetId" example:"550dc009-cea6-4c12-b2a5-03446eb7b7cf"`
	OnBudget bool      `json:"onBudget" example:"false"` // Always false when external: true
	External bool      `json:"external" example:"true"`
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
	database.DB.Where(Transaction{TransactionCreate: TransactionCreate{SourceAccountID: a.ID}}).Or(Transaction{TransactionCreate: TransactionCreate{DestinationAccountID: a.ID}}).Find(&transactions)
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
