package models

import (
	"fmt"
	"time"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/database"
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
	Links             struct {
		Self         string `json:"self" example:"https://example.com/api/v1/accounts/af892e10-7e0a-4fb8-b1bc-4b6d88401ed2"`
		Transactions string `json:"transactions" example:"https://example.com/api/v1/transactions?account=af892e10-7e0a-4fb8-b1bc-4b6d88401ed2"`
	} `json:"links" gorm:"-"`
}

type AccountCreate struct {
	Name               string          `json:"name" example:"Cash" default:"" gorm:"uniqueIndex:account_name_budget_id"`
	Note               string          `json:"note" example:"Money in my wallet" default:""`
	BudgetID           uuid.UUID       `json:"budgetId" example:"550dc009-cea6-4c12-b2a5-03446eb7b7cf" gorm:"uniqueIndex:account_name_budget_id"`
	OnBudget           bool            `json:"onBudget" example:"true" default:"false"` // Always false when external: true
	External           bool            `json:"external" example:"false" default:"false"`
	InitialBalance     decimal.Decimal `json:"initialBalance" example:"173.12" default:"0"`
	InitialBalanceDate *time.Time      `json:"initialBalanceDate" example:"2017-05-12T00:00:00Z"`
	Hidden             bool            `json:"hidden" example:"true" default:"false"`
	ImportHash         string          `json:"importHash" example:"867e3a26dc0baf73f4bff506f31a97f6c32088917e9e5cf1a5ed6f3f84a6fa70" default:""` // The SHA256 hash of a unique combination of values to use in duplicate detection
}

func (a Account) Self() string {
	return "Account"
}

func (a Account) WithCalculations(db *gorm.DB) (Account, error) {
	balance, _, err := a.GetBalanceMonth(db, types.Month{})
	if err != nil {
		return Account{}, err
	}
	a.Balance = balance

	a.ReconciledBalance, err = a.SumReconciled(db)
	if err != nil {
		return Account{}, err
	}

	return a, nil
}

func (a *Account) AfterFind(tx *gorm.DB) (err error) {
	a.links(tx)
	return
}

// AfterSave also sets the links so that we do not need to
// query the resource directly after creating or updating it.
func (a *Account) AfterSave(tx *gorm.DB) (err error) {
	a.links(tx)
	return
}

func (a *Account) links(tx *gorm.DB) {
	url := tx.Statement.Context.Value(database.ContextURL)
	a.Links.Self = fmt.Sprintf("%s/v1/accounts/%s", url, a.ID)
	a.Links.Transactions = fmt.Sprintf("%s/v1/transactions?account=%s", url, a.ID)
}

// BeforeSave sets OnBudget to false when External is true.
func (a *Account) BeforeSave(_ *gorm.DB) (err error) {
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
func (a Account) SumReconciled(db *gorm.DB) (balance decimal.Decimal, err error) {
	var transactions []Transaction

	err = db.
		Preload("DestinationAccount").
		Preload("SourceAccount").
		Where(
			db.Where(Transaction{TransactionCreate: TransactionCreate{DestinationAccountID: a.ID, Reconciled: true}}).
				Or(db.Where(Transaction{TransactionCreate: TransactionCreate{SourceAccountID: a.ID, Reconciled: true}}))).
		Find(&transactions).Error

	if err != nil {
		return decimal.Zero, err
	}

	balance = a.InitialBalance

	// Add incoming transactions, subtract outgoing transactions
	for _, t := range transactions {
		if t.DestinationAccountID == a.ID {
			balance = balance.Add(t.Amount)
		} else {
			balance = balance.Sub(t.Amount)
		}
	}

	return
}

// GetBalanceMonth calculates the balance and available sums for a specific month.
//
// The balance Decimal is the actual account balance, factoring in all transactions before the end of the month.
// The available Decimal is the sum that is available for budgeting at the end of the specified month.
func (a Account) GetBalanceMonth(db *gorm.DB, month types.Month) (balance, available decimal.Decimal, err error) {
	var transactions []Transaction

	query := db.
		Preload("DestinationAccount").
		Preload("SourceAccount").
		Where(
			db.Where(Transaction{TransactionCreate: TransactionCreate{DestinationAccountID: a.ID}}).
				Or(db.Where(Transaction{TransactionCreate: TransactionCreate{SourceAccountID: a.ID}})))

	// Limit to the month if it is specified
	if !month.IsZero() {
		query = query.Where("transactions.date < date(?)", month.AddDate(0, 1))
	}

	err = query.Find(&transactions).Error
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}

	if month.IsZero() || (a.InitialBalanceDate != nil && month.AddDate(0, 1).AfterTime(*a.InitialBalanceDate)) {
		balance = a.InitialBalance
		available = a.InitialBalance
	}

	// Add incoming transactions, subtract outgoing transactions
	// For available, only do so if the next month is after the availableFrom date
	for _, t := range transactions {
		if t.DestinationAccountID == a.ID {
			balance = balance.Add(t.Amount)

			// If the transaction is an income transaction, but its AvailableFrom is after this month, skip it
			if !month.AddDate(0, 1).After(t.AvailableFrom) && t.SourceAccount.External && t.EnvelopeID == nil {
				continue
			}
			available = available.Add(t.Amount)
		} else {
			balance = balance.Sub(t.Amount)
			available = available.Sub(t.Amount)
		}
	}

	return
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
