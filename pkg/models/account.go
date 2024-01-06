package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Account represents an asset account, e.g. a bank account.
type Account struct {
	DefaultModel
	Budget             Budget
	BudgetID           uuid.UUID `gorm:"uniqueIndex:account_name_budget_id"`
	Name               string    `gorm:"uniqueIndex:account_name_budget_id"`
	Note               string
	OnBudget           bool
	External           bool
	InitialBalance     decimal.Decimal `gorm:"type:DECIMAL(20,8)"`
	InitialBalanceDate *time.Time
	Archived           bool
	ImportHash         string // A SHA256 hash of a unique combination of values to use in duplicate detection for imports
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

// BeforeSave ensures consistency for the account
//
// It enforces OnBudget to be false when the account
// is external.
//
// It trims whitespace from all strings
func (a *Account) BeforeSave(_ *gorm.DB) error {
	if a.External {
		a.OnBudget = false
	}

	a.Name = strings.TrimSpace(a.Name)
	a.Note = strings.TrimSpace(a.Note)
	a.ImportHash = strings.TrimSpace(a.ImportHash)

	return nil
}

// Transactions returns all transactions for this account.
func (a Account) Transactions(db *gorm.DB) []Transaction {
	var transactions []Transaction

	// Get all transactions where the account is either the source or the destination
	db.Where(Transaction{SourceAccountID: a.ID}).Or(Transaction{DestinationAccountID: a.ID}).Find(&transactions)
	return transactions
}

// Transactions returns all transactions for this account.
func (a Account) SumReconciled(db *gorm.DB) (balance decimal.Decimal, err error) {
	var transactions []Transaction

	err = db.
		Preload("DestinationAccount").
		Preload("SourceAccount").
		Where(
			db.Where(Transaction{DestinationAccountID: a.ID, ReconciledDestination: true}).
				Or(db.Where(Transaction{SourceAccountID: a.ID, ReconciledSource: true}))).
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
			db.Where(Transaction{DestinationAccountID: a.ID}).
				Or(db.Where(Transaction{SourceAccountID: a.ID})))

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
		Select("envelopes.id as e_id, datetime(envelopes.created_at) as created").
		Where(&Transaction{
			DestinationAccountID: a.ID,
		}).
		Order("datetime(transactions.date) DESC").
		Limit(50)

	// Group by frequency
	err := db.
		Table("(?)", latest).
		// Set the nil UUID as ID if the envelope ID is NULL, since count() only counts non-null values
		Select("IIF(e_id IS NOT NULL, e_id, '00000000-0000-0000-0000-000000000000') as id").
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
