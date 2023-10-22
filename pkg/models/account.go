package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// AccountCreate represents all parameters of an Account that are configurable by the user.
type AccountCreate struct {
	Name               string          `json:"name" example:"Cash" default:"" gorm:"uniqueIndex:account_name_budget_id"`                          // Name of the account
	Note               string          `json:"note" example:"Money in my wallet" default:""`                                                      // A longer description for the account
	BudgetID           uuid.UUID       `json:"budgetId" example:"550dc009-cea6-4c12-b2a5-03446eb7b7cf" gorm:"uniqueIndex:account_name_budget_id"` // ID of the budget this account belongs to
	OnBudget           bool            `json:"onBudget" example:"true" default:"false"`                                                           // Does the account factor into the available budget? Always false when external: true
	External           bool            `json:"external" example:"false" default:"false"`                                                          // Does the account belong to the budget owner or not?
	InitialBalance     decimal.Decimal `json:"initialBalance" example:"173.12" default:"0"`                                                       // Balance of the account before any transactions were recorded
	InitialBalanceDate *time.Time      `json:"initialBalanceDate" example:"2017-05-12T00:00:00Z"`                                                 // Date of the initial balance
	Hidden             bool            `json:"hidden" example:"true" default:"false"`                                                             // Is the account archived?
	ImportHash         string          `json:"importHash" example:"867e3a26dc0baf73f4bff506f31a97f6c32088917e9e5cf1a5ed6f3f84a6fa70" default:""`  // The SHA256 hash of a unique combination of values to use in duplicate detection
}

// Account represents an asset account, e.g. a bank account.
type Account struct {
	DefaultModel
	AccountCreate
	Budget Budget `json:"-"`
}

func (Account) Self() string {
	return "Account"
}

// BeforeUpdate verifies the state of the account before
// committing an update to the database.
func (a Account) BeforeUpdate(tx *gorm.DB) (err error) {
	// Account is being set to be on budget, verify that no transactions
	// with this account as destination has an envelope set
	if tx.Statement.Changed("OnBudget") && !a.OnBudget {
		var transactions []Transaction
		err = tx.Model(&Transaction{}).
			Joins("JOIN accounts ON transactions.source_account_id = accounts.id").
			Where("destination_account_id = ? AND accounts.on_budget AND envelope_id not null", a.ID).
			Find(&transactions).Error
		if err != nil {
			return
		}

		if len(transactions) > 0 {
			strs := make([]string, len(transactions))
			for i, t := range transactions {
				strs[i] = t.ID.String()
			}

			ids := strings.Join(strs, ",")
			return fmt.Errorf("the account cannot be set to on budget because the following transactions have an envelope set: %s", ids)
		}
	}

	return nil
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

// SetRecentEnvelopes returns the most common envelopes used in the last 50
// transactions where the account is the destination account.
//
// The list is sorted by decending frequency of the envelope being used.
// If two envelopes appear with the same frequency, the order is undefined
// since sqlite does not have ordering by datetime with more than second precision.
//
// If creation times are more than a second apart, ordering is well defined.
func (a Account) RecentEnvelopes(db *gorm.DB) ([]*uuid.UUID, error) {
	var envelopeIDs []uuid.UUID

	// Get the Envelope IDs for the 50 latest transactions
	latest := db.
		Model(&Transaction{}).
		Joins("LEFT JOIN envelopes ON envelopes.id = transactions.envelope_id AND envelopes.deleted_at IS NULL").
		Select("envelopes.id as id, datetime(envelopes.created_at) as created").
		Where(&Transaction{
			TransactionCreate: TransactionCreate{
				DestinationAccountID: a.ID,
			},
		}).
		Order("datetime(transactions.date) DESC").
		Limit(50)

	// Group by frequency
	err := db.
		Table("(?)", latest).
		Select("id").
		Group("id").
		Order("count(id) DESC").
		Order("created ASC").
		Limit(5).
		Find(&envelopeIDs).Error
	if err != nil {
		return []*uuid.UUID{}, err
	}

	var ids []*uuid.UUID
	for _, envelopeID := range envelopeIDs {
		eID := envelopeID
		if eID == uuid.Nil {
			ids = append(ids, nil)
			continue
		}

		ids = append(ids, &eID)
	}

	return ids, nil
}
