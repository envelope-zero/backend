package models

import (
	"fmt"
	"strings"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Account represents an asset account, e.g. a bank account.
type AccountV2 struct {
	DefaultModel
	AccountCreate
	Budget            Budget          `json:"-"`
	Balance           decimal.Decimal `json:"balance" gorm:"-" example:"2735.17"`           // Balance of the account, including all transactions referencing it
	ReconciledBalance decimal.Decimal `json:"reconciledBalance" gorm:"-" example:"2539.57"` // Balance of the account, including all reconciled transactions referencing it
	RecentEnvelopes   []*uuid.UUID    `json:"recentEnvelopes" gorm:"-"`                     // Envelopes recently used with this account

	Links struct {
		Self         string `json:"self" example:"https://example.com/api/v1/accounts/af892e10-7e0a-4fb8-b1bc-4b6d88401ed2"`                     // The account itself
		Transactions string `json:"transactions" example:"https://example.com/api/v1/transactions?account=af892e10-7e0a-4fb8-b1bc-4b6d88401ed2"` // Transactions referencing the account
	} `json:"links" gorm:"-"`
}

func (AccountV2) TableName() string {
	return "accounts"
}

func (AccountV2) Self() string {
	return "Account"
}

func (a *AccountV2) WithCalculations(db *gorm.DB) error {
	balance, _, err := a.GetBalanceMonth(db, types.Month{})
	if err != nil {
		return err
	}
	a.Balance = balance

	a.ReconciledBalance, err = a.SumReconciled(db)
	if err != nil {
		return err
	}

	return nil
}

func (a *AccountV2) AfterFind(tx *gorm.DB) (err error) {
	a.links(tx)
	return
}

// AfterSave also sets the links so that we do not need to
// query the resource directly after creating or updating it.
func (a *AccountV2) AfterSave(tx *gorm.DB) (err error) {
	a.links(tx)
	return
}

func (a *AccountV2) links(tx *gorm.DB) {
	url := tx.Statement.Context.Value(database.ContextURL)
	a.Links.Self = fmt.Sprintf("%s/v1/accounts/%s", url, a.ID)
	a.Links.Transactions = fmt.Sprintf("%s/v1/transactions?account=%s", url, a.ID)
}

// BeforeUpdate verifies the state of the account before
// committing an update to the database.
func (a *AccountV2) BeforeUpdate(tx *gorm.DB) (err error) {
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
func (a *AccountV2) BeforeSave(_ *gorm.DB) (err error) {
	if a.External {
		a.OnBudget = false
	}
	return nil
}

// Transactions returns all transactions for this account.
func (a AccountV2) SumReconciled(db *gorm.DB) (balance decimal.Decimal, err error) {
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
func (a AccountV2) GetBalanceMonth(db *gorm.DB, month types.Month) (balance, available decimal.Decimal, err error) {
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
func (a *AccountV2) SetRecentEnvelopes(db *gorm.DB) error {
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
		return err
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

	a.RecentEnvelopes = ids
	return nil
}
