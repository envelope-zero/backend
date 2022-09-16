package models

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Account represents an asset account, e.g. a bank account.
type Account struct {
	Model
	AccountCreate
	Budget            Budget          `json:"-"`
	Balance           decimal.Decimal `json:"balance" gorm:"-" example:"2735.17"`
	ReconciledBalance decimal.Decimal `json:"reconciledBalance" gorm:"-" example:"2539.57"`
}

type AccountCreate struct {
	Name     string    `json:"name" example:"Cash" default:""`
	Note     string    `json:"note" example:"Money in my wallet" default:""`
	BudgetID uuid.UUID `json:"budgetId" example:"550dc009-cea6-4c12-b2a5-03446eb7b7cf"`
	OnBudget bool      `json:"onBudget" example:"true" default:"false"` // Always false when external: true
	External bool      `json:"external" example:"false" default:"false"`
}

func (a Account) WithCalculations(db *gorm.DB) Account {
	a.Balance = a.getBalance(db)
	a.ReconciledBalance = a.SumReconciledTransactions(db)

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
func (a Account) Transactions(db *gorm.DB) []Transaction {
	var transactions []Transaction

	// Get all transactions where the account is either the source or the destination
	db.Where(Transaction{TransactionCreate: TransactionCreate{SourceAccountID: a.ID}}).Or(Transaction{TransactionCreate: TransactionCreate{DestinationAccountID: a.ID}}).Find(&transactions)
	return transactions
}

// Transactions returns all transactions for this account.
func (a Account) SumReconciledTransactions(db *gorm.DB) decimal.Decimal {
	return TransactionsSum(db,
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
func (a Account) getBalance(db *gorm.DB) decimal.Decimal {
	return TransactionsSum(db,
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

// TransactionSums returns the sum of all transactions matching two Transaction structs
//
// The incoming Transactions fields is used to add the amount of all matching transactions to the overall sum
// The outgoing Transactions fields is used to subtract the amount of all matching transactions from the overall sum.
func TransactionsSum(db *gorm.DB, incoming, outgoing Transaction) decimal.Decimal {
	var outgoingSum, incomingSum decimal.NullDecimal

	_ = db.Table("transactions").
		Where(&outgoing).
		Where("deleted_at is NULL").
		Select("SUM(amount)").
		Row().
		Scan(&outgoingSum)

	_ = db.Table("transactions").
		Where(&incoming).
		Where("deleted_at is NULL").
		Select("SUM(amount)").
		Row().
		Scan(&incomingSum)

	return incomingSum.Decimal.Sub(outgoingSum.Decimal)
}
