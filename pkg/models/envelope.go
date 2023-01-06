package models

import (
	"sort"
	"time"

	"github.com/envelope-zero/backend/internal/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Envelope represents an envelope in your budget.
type Envelope struct {
	DefaultModel
	EnvelopeCreate
	Category Category `json:"-"`
}

type EnvelopeCreate struct {
	Name       string    `json:"name" gorm:"uniqueIndex:envelope_category_name" example:"Groceries" default:""`
	CategoryID uuid.UUID `json:"categoryId" gorm:"uniqueIndex:envelope_category_name" example:"878c831f-af99-4a71-b3ca-80deb7d793c1"`
	Note       string    `json:"note" example:"For stuff bought at supermarkets and drugstores" default:""`
	Hidden     bool      `json:"hidden" example:"true" default:"false"`
}

type EnvelopeMonthLinks struct {
	Allocation string `json:"allocation" example:"https://example.com/api/v1/allocations/772d6956-ecba-485b-8a27-46a506c5a2a3"` // This is an empty string when no allocation exists
}

// EnvelopeMonth contains data about an Envelope for a specific month.
type EnvelopeMonth struct {
	ID         uuid.UUID          `json:"id" example:"10b9705d-3356-459e-9d5a-28d42a6c4547"`               // The ID of the Envelope
	Name       string             `json:"name" example:"Groceries"`                                        // The name of the Envelope
	Month      types.Month        `json:"month" example:"1969-06-01T00:00:00.000000Z" hidden:"deprecated"` // This is always set to 00:00 UTC on the first of the month. **This field is deprecated and will be removed in v2**
	Spent      decimal.Decimal    `json:"spent" example:"73.12"`
	Balance    decimal.Decimal    `json:"balance" example:"12.32"`
	Allocation decimal.Decimal    `json:"allocation" example:"85.44"`
	Links      EnvelopeMonthLinks `json:"links"`
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

	return outgoingSum.Sub(incomingSum)
}

type AggregatedTransaction struct {
	Amount                     decimal.Decimal
	Date                       time.Time
	SourceAccountExternal      bool
	DestinationAccountExternal bool
}

type EnvelopeMonthAllocation struct {
	Month      time.Time
	Allocation decimal.Decimal
}

type EnvelopeMonthConfig struct {
	Month         time.Time
	OverspendMode OverspendMode
}

// Balance calculates the balance of an Envelope in a specific month.
func (e Envelope) Balance(db *gorm.DB, month types.Month) (decimal.Decimal, error) {
	// Get all relevant data for rawTransactions
	var rawTransactions []AggregatedTransaction
	err := db.
		Table("transactions").
		Joins("JOIN accounts source_account ON transactions.source_account_id = source_account.id AND source_account.deleted_at IS NULL").
		Joins("JOIN accounts destination_account ON transactions.destination_account_id = destination_account.id AND destination_account.deleted_at IS NULL").
		Where("transactions.date < date(?)", month.AddDate(0, 1)).
		Where("transactions.envelope_id = ?", e.ID).
		Select("transactions.amount AS Amount, transactions.date AS Date, source_account.external AS SourceAccountExternal, destination_account.external AS DestinationAccountExternal").
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

	// Get allocations
	var rawAllocations []Allocation
	err = db.
		Table("allocations").
		Where("allocations.month < date(?)", month.AddDate(0, 1)).
		Where("allocations.envelope_id = ?", e.ID).
		Find(&rawAllocations).Error
	if err != nil {
		return decimal.Zero, nil
	}

	// Sort allocations by month
	allocationMonths := make(map[types.Month]Allocation)
	for _, allocation := range rawAllocations {
		allocationMonths[allocation.Month] = allocation
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

	// Create a slice of the months that have Allocation
	// data to have a sorted list we can iterate over
	monthKeys := make([]types.Month, 0)
	for k := range allocationMonths {
		monthKeys = append(monthKeys, k)
		monthsWithData[k] = true
	}

	// Add the months that have MonthConfigs
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
		currentMonthAllocation, allocationOk := allocationMonths[loopMonth]
		currentMonthConfig, configOk := configMonths[loopMonth]

		// We always go forward one month until we
		// reach the last one with data
		loopMonth = loopMonth.AddDate(0, 1)

		// If there is no data for the current month,
		// we loop once more and go on to the next month
		//
		// We also reset the balance to 0 if it is negative
		// since with no MonthConfig, the balance starts from 0 again
		if !transactionsOk && !allocationOk && !configOk {
			i--
			if sum.IsNegative() {
				sum = decimal.Zero
			}
			continue
		}

		// Initialize the sum for this month
		monthSum := sum

		for _, transaction := range currentMonthTransactions {
			if transaction.SourceAccountExternal {
				// Incoming money gets added to the balance
				monthSum = monthSum.Add(transaction.Amount)
			} else {
				// Outgoing gets subtracted
				monthSum = monthSum.Sub(transaction.Amount)
			}
		}

		// The zero value for a decimal is Zero, so we don't need to check
		// if there is an allocation
		monthSum = monthSum.Add(currentMonthAllocation.Amount)

		// If the value is not negative, we're done here.
		if !monthSum.IsNegative() {
			sum = monthSum
			continue
		}

		// If there is overspend and the overspend should affect the envelope,
		// the sum for the month is subtracted (using decimal.Add since the
		// number is negative)
		if monthSum.IsNegative() && configOk && currentMonthConfig.OverspendMode == AffectEnvelope {
			sum = monthSum
			// If this is the last month, the sum is the monthSum
		} else if monthSum.IsNegative() && loopMonth.After(month) {
			sum = monthSum
			// In all other cases, the overspend affects Available to Budget,
			// not the envelope balance
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

// Month calculates the month specific values for an envelope and returns an EnvelopeMonth and allocation ID for them.
func (e Envelope) Month(db *gorm.DB, month types.Month) (EnvelopeMonth, uuid.UUID, error) {
	spent := e.Spent(db, month)
	envelopeMonth := EnvelopeMonth{
		ID:         e.ID,
		Name:       e.Name,
		Month:      month,
		Spent:      spent,
		Balance:    decimal.NewFromFloat(0),
		Allocation: decimal.NewFromFloat(0),
	}

	var allocation Allocation
	err := db.First(&allocation, &Allocation{
		AllocationCreate: AllocationCreate{
			EnvelopeID: e.ID,
			Month:      month,
		},
	}).Error

	// If an unexpected error occurs, return
	if err != nil && err != gorm.ErrRecordNotFound {
		return EnvelopeMonth{}, uuid.Nil, err
	}

	envelopeMonth.Balance, err = e.Balance(db, month)
	if err != nil {
		return EnvelopeMonth{}, uuid.Nil, err
	}

	envelopeMonth.Allocation = allocation.Amount
	return envelopeMonth, allocation.ID, nil
}
