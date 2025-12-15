package models

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/envelope-zero/backend/v7/internal/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Envelope represents an envelope in your budget.
type Envelope struct {
	DefaultModel
	Category   Category  `json:"-"`
	CategoryID uuid.UUID `gorm:"uniqueIndex:envelope_category_name"`
	Name       string    `gorm:"uniqueIndex:envelope_category_name"`
	Note       string
	Archived   bool
}

var ErrEnvelopeNameNotUnique = errors.New("the envelope name must be unique for the category")

func (e *Envelope) BeforeCreate(tx *gorm.DB) error {
	_ = e.DefaultModel.BeforeCreate(tx)

	toSave := tx.Statement.Dest.(*Envelope)
	return e.checkIntegrity(tx, *toSave)
}

func (e *Envelope) BeforeSave(_ *gorm.DB) error {
	e.Name = strings.TrimSpace(e.Name)
	e.Note = strings.TrimSpace(e.Note)

	return nil
}

// BeforeUpdate verifies the state of the envelope before
// committing an update to the database.
func (e *Envelope) BeforeUpdate(tx *gorm.DB) (err error) {
	toSave := tx.Statement.Dest.(Envelope)
	if tx.Statement.Changed("CategoryID") {
		err := e.checkIntegrity(tx, toSave)
		if err != nil {
			return err
		}
	}

	// If the archival state is updated from archived to unarchived and the category is
	// archived, unarchive the category, too.
	if tx.Statement.Changed("Archived") && !toSave.Archived {
		var category Category
		err = tx.First(&category, e.CategoryID).Error
		if err != nil {
			return err
		}

		if category.Archived {
			tx.Model(&category).Select("Archived").Updates(Category{Archived: false})
		}
	}

	return err
}

// checkIntegrity verifies references to other resources
func (e *Envelope) checkIntegrity(tx *gorm.DB, toSave Envelope) error {
	return tx.First(&Category{}, toSave.CategoryID).Error
}

// Spent returns the amount spent for the month the time.Time instance is in.
func (e Envelope) Spent(db *gorm.DB, month types.Month) decimal.Decimal {
	// All transactions where the Envelope ID matches and that have an external account as source and an internal account as destination
	var incoming []Transaction

	db.Joins("SourceAccount").Joins("DestinationAccount").Where(
		"SourceAccount__on_budget = 0 AND DestinationAccount__on_budget = 1 AND transactions.envelope_id = ?", e.ID,
	).Find(&incoming)

	// Add all incoming transactions that are in the correct month
	incomingSum := decimal.Zero
	for _, transaction := range incoming {
		if month.Contains(transaction.Date) {
			incomingSum = incomingSum.Add(transaction.Amount)
		}
	}

	var outgoing []Transaction
	db.Joins("SourceAccount").Joins("DestinationAccount").Where(
		"SourceAccount__on_budget = 1 AND DestinationAccount__on_budget = 0 AND transactions.envelope_id = ?", e.ID,
	).Find(&outgoing)

	// Add all outgoing transactions that are in the correct month
	outgoingSum := decimal.Zero
	for _, transaction := range outgoing {
		if month.Contains(transaction.Date) {
			outgoingSum = outgoingSum.Add(transaction.Amount)
		}
	}

	return outgoingSum.Neg().Add(incomingSum)
}

type AggregatedTransaction struct {
	Amount                     decimal.Decimal
	Date                       time.Time
	SourceAccountOnBudget      bool
	DestinationAccountOnBudget bool
}

// Balance calculates the balance of an Envelope in a specific month.
func (e Envelope) Balance(db *gorm.DB, month types.Month) (decimal.Decimal, error) {
	// Get all relevant data for rawTransactions
	var rawTransactions []AggregatedTransaction
	err := db.
		Table("transactions").
		Joins("JOIN accounts source_account ON transactions.source_account_id = source_account.id").
		Joins("JOIN accounts destination_account ON transactions.destination_account_id = destination_account.id").
		Where("transactions.date < date(?)", month.AddDate(0, 1)).
		Where("transactions.envelope_id = ?", e.ID).
		Select("transactions.amount AS Amount, transactions.date AS Date, source_account.on_budget AS SourceAccountOnBudget, destination_account.on_budget AS DestinationAccountOnBudget").
		Find(&rawTransactions).Error
	if err != nil {
		return decimal.Zero, err
	}

	// Sort monthTransactions by month
	monthTransactions := make(map[types.Month][]AggregatedTransaction)
	for _, transaction := range rawTransactions {
		tDate := types.NewMonth(transaction.Date.Year(), transaction.Date.Month())
		monthTransactions[tDate] = append(monthTransactions[tDate], transaction)
	}

	// Get MonthConfigs
	var rawConfigs []MonthConfig
	err = db.
		Table("month_configs").
		Where("month_configs.month < date(?)", month.AddDate(0, 1)).
		Where("month_configs.envelope_id = ?", e.ID).
		Find(&rawConfigs).Error
	if err != nil {
		return decimal.Zero, nil
	}

	// Sort MonthConfigs by month
	configMonths := make(map[types.Month]MonthConfig)
	for _, monthConfig := range rawConfigs {
		configMonths[monthConfig.Month] = monthConfig
	}

	// This is a helper map to only add unique months to the
	// monthKeys slice
	monthsWithData := make(map[types.Month]bool)

	// Create a slice of the months that have MonthConfigs
	// to have a sorted list we can iterate over
	monthKeys := make([]types.Month, 0)
	for k := range configMonths {
		if _, ok := monthsWithData[k]; !ok {
			monthKeys = append(monthKeys, k)
			monthsWithData[k] = true
		}
	}

	// Add the months that have transaction data
	for k := range monthTransactions {
		if _, ok := monthsWithData[k]; !ok {
			monthKeys = append(monthKeys, k)
		}
	}

	// Sort by time so that earlier months are first
	sort.Slice(monthKeys, func(i, j int) bool {
		return monthKeys[i].Before(monthKeys[j])
	})

	if len(monthKeys) == 0 {
		return decimal.Zero, nil
	}

	sum := decimal.Zero
	loopMonth := monthKeys[0]
	for i := 0; i < len(monthKeys); i++ {
		currentMonthTransactions, transactionsOk := monthTransactions[loopMonth]
		currentMonthConfig, configOk := configMonths[loopMonth]

		// We always go forward one month until we
		// reach the last one with data
		loopMonth = loopMonth.AddDate(0, 1)

		// If there is no data for the current month,
		// we loop once more and go on to the next month
		//
		// We also reset the balance to 0 if it is negative
		// since with no MonthConfig, the balance starts from 0 again
		if !transactionsOk && !configOk {
			i--
			if sum.IsNegative() {
				sum = decimal.Zero
			}
			continue
		}

		// Initialize the sum for this month
		monthSum := sum

		for _, transaction := range currentMonthTransactions {
			if transaction.SourceAccountOnBudget {
				// Outgoing gets subtracted
				monthSum = monthSum.Sub(transaction.Amount)
			} else {
				// Incoming money gets added to the balance
				monthSum = monthSum.Add(transaction.Amount)
			}
		}

		// The zero value for a decimal is Zero, so we don't need to check
		// if there is an allocation
		monthSum = monthSum.Add(currentMonthConfig.Allocation)

		// If the value is not negative, we're done here.
		if !monthSum.IsNegative() {
			sum = monthSum
			continue
		}

		// If this is the last month, the sum is the monthSum
		if monthSum.IsNegative() && loopMonth.After(month) {
			sum = monthSum
		} else if monthSum.IsNegative() {
			sum = decimal.Zero
		}

		// In cases where the sum is negative and we do not have
		// configuration for the month before the month we are
		// calculating the balance for, we set the balance to 0
		// in the last loop iteration.
		//
		// This stops the rollover of overflow without configuration
		// infinitely far into the future.
		//
		// We check the month before the month we are calculating for
		// because if we do not have configuration for the current month,
		// negative balance from the month before could still roll over.
		if monthSum.IsNegative() && i+1 == len(monthKeys) && loopMonth.Before(month) {
			sum = decimal.Zero
		}
	}

	return sum, nil
}

// EnvelopeMonth contains data about an Envelope for a specific month.
type EnvelopeMonth struct {
	Envelope
	Spent      decimal.Decimal `json:"spent" example:"73.12"`      // The amount spent over the whole month
	Balance    decimal.Decimal `json:"balance" example:"12.32"`    // The balance at the end of the monht
	Allocation decimal.Decimal `json:"allocation" example:"85.44"` // The amount of money allocated
}

// Month calculates the month specific values for an envelope and returns an EnvelopeMonth and allocation ID for them.
func (e Envelope) Month(db *gorm.DB, month types.Month) (EnvelopeMonth, error) {
	spent := e.Spent(db, month)
	envelopeMonth := EnvelopeMonth{
		Envelope:   e,
		Spent:      spent,
		Balance:    decimal.NewFromFloat(0),
		Allocation: decimal.NewFromFloat(0),
	}

	var monthConfig MonthConfig
	err := db.Where(&MonthConfig{
		EnvelopeID: e.ID,
		Month:      month,
	}).Find(&monthConfig).Error

	// If an unexpected error occurs, return
	if err != nil && err != gorm.ErrRecordNotFound {
		return EnvelopeMonth{}, err
	}

	envelopeMonth.Balance, err = e.Balance(db, month)
	if err != nil {
		return EnvelopeMonth{}, err
	}

	envelopeMonth.Allocation = monthConfig.Allocation
	return envelopeMonth, nil
}

// Returns all envelopes on this instance for export
func (Envelope) Export() (json.RawMessage, error) {
	var envelopes []Envelope
	err := DB.Unscoped().Where(&Envelope{}).Find(&envelopes).Error
	if err != nil {
		return nil, err
	}

	j, err := json.Marshal(&envelopes)
	if err != nil {
		return json.RawMessage{}, err
	}
	return json.RawMessage(j), nil
}
