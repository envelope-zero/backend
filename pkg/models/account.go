package models

import (
	"time"

	"github.com/envelope-zero/backend/internal/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Account represents an asset account, e.g. a bank account.
type Account struct {
	DefaultModel
	AccountCreate
	Budget            Budget          `json:"-"`
	Balance           decimal.Decimal `json:"balance" gorm:"-" example:"2735.17"`
	ReconciledBalance decimal.Decimal `json:"reconciledBalance" gorm:"-" example:"2539.57"`
}

type AccountCreate struct {
	Name               string          `json:"name" example:"Cash" default:""`
	Note               string          `json:"note" example:"Money in my wallet" default:""`
	BudgetID           uuid.UUID       `json:"budgetId" example:"550dc009-cea6-4c12-b2a5-03446eb7b7cf"`
	OnBudget           bool            `json:"onBudget" example:"true" default:"false"` // Always false when external: true
	External           bool            `json:"external" example:"false" default:"false"`
	InitialBalance     decimal.Decimal `json:"initialBalance" example:"173.12" default:"0"`
	InitialBalanceDate *time.Time      `json:"initialBalanceDate" example:"2017-05-12T00:00:00Z"`
	Hidden             bool            `json:"hidden" example:"true" default:"false"`
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
	return a.InitialBalance.Add(TransactionsSum(db,
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
	))
}

// GetBalanceMonth calculates the balance at the end of a specific month.
func (a Account) GetBalanceMonth(db *gorm.DB, month types.Month) (decimal.Decimal, error) {
	var transactions []Transaction

	err := db.
		Where("transactions.date < date(?)", month.AddDate(0, 1)).
		Where(
			db.Where(Transaction{TransactionCreate: TransactionCreate{DestinationAccountID: a.ID}}).
				Or(db.Where(Transaction{TransactionCreate: TransactionCreate{SourceAccountID: a.ID}}))).
		Find(&transactions).
		Error
	if err != nil {
		return decimal.Zero, err
	}

	sum := decimal.Zero

	if a.InitialBalanceDate != nil && month.AddDate(0, 1).AfterTime(*a.InitialBalanceDate) {
		sum = a.InitialBalance
	}

	// Add incoming transactions, subtract outgoing transactions
	for _, t := range transactions {
		if t.DestinationAccountID == a.ID {
			sum = sum.Add(t.Amount)
		} else {
			sum = sum.Sub(t.Amount)
		}
	}

	return sum, nil
}

// getBalance returns the balance of the account calculated over all transactions.
func (a Account) getBalance(db *gorm.DB) decimal.Decimal {
	return a.InitialBalance.Add(TransactionsSum(db,
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
	))
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

// RecentEnvelopes returns the most common envelopes used in the last 10
// transactions where the account is the destination account.
//
// The list is sorted by decending frequency of the envelope being used.
func (a Account) RecentEnvelopes(db *gorm.DB) (envelopes []Envelope, err error) {
	err = db.
		Table("transactions").
		Select("envelopes.*, count(envelopes.id) AS count").
		Joins("JOIN envelopes ON envelopes.id = transactions.envelope_id AND envelopes.deleted_at IS NULL").
		Order("count DESC, date(transactions.date) DESC").
		Where(&Transaction{
			TransactionCreate: TransactionCreate{
				DestinationAccountID: a.ID,
			},
		}).
		Limit(10).
		Group("envelopes.id").
		Find(&envelopes).Error

	return
}
