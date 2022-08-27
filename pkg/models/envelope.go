package models

import (
	"time"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Envelope represents an envelope in your budget.
type Envelope struct {
	Model
	EnvelopeCreate
	Category Category `json:"-"`
}

type EnvelopeCreate struct {
	Name       string    `json:"name" gorm:"uniqueIndex:envelope_category_name" example:"Groceries" default:""`
	CategoryID uuid.UUID `json:"categoryId" gorm:"uniqueIndex:envelope_category_name" example:"878c831f-af99-4a71-b3ca-80deb7d793c1"`
	Note       string    `json:"note" example:"For stuff bought at supermarkets and drugstores" default:""`
}

// EnvelopeMonth contains data about an Envelope for a specific month.
type EnvelopeMonth struct {
	ID         uuid.UUID       `json:"id" example:"10b9705d-3356-459e-9d5a-28d42a6c4547"` // The ID of the Envelope
	Name       string          `json:"name" example:"Groceries"`                          // The name of the Envelope
	Month      time.Time       `json:"month" example:"1969-06-01T00:00:00.000000Z"`       // This is always set to 00:00 UTC on the first of the month.
	Spent      decimal.Decimal `json:"spent" example:"73.12"`
	Balance    decimal.Decimal `json:"balance" example:"12.32"`
	Allocation decimal.Decimal `json:"allocation" example:"85.44"`
}

// Spent returns the amount spent for the month the time.Time instance is in.
func (e Envelope) Spent(t time.Time) decimal.Decimal {
	// All transactions where the Envelope ID matches and that have an external account as source and an internal account as destination
	var incoming []Transaction

	database.DB.Joins("SourceAccount").Joins("DestinationAccount").Where(
		"SourceAccount__external = 1 AND DestinationAccount__external = 0 AND transactions.envelope_id = ?", e.ID,
	).Find(&incoming)

	// Add all incoming transactions that are in the correct month
	incomingSum := decimal.Zero
	for _, transaction := range incoming {
		if transaction.Date.UTC().Year() == t.UTC().Year() && transaction.Date.UTC().Month() == t.UTC().Month() {
			incomingSum = incomingSum.Add(transaction.Amount)
		}
	}

	var outgoing []Transaction
	database.DB.Joins("SourceAccount").Joins("DestinationAccount").Where(
		"SourceAccount__external = 0 AND DestinationAccount__external = 1 AND transactions.envelope_id = ?", e.ID,
	).Find(&outgoing)

	// Add all outgoing transactions that are in the correct month
	outgoingSum := decimal.Zero
	for _, transaction := range outgoing {
		if transaction.Date.UTC().Year() == t.UTC().Year() && transaction.Date.UTC().Month() == t.UTC().Month() {
			outgoingSum = outgoingSum.Add(transaction.Amount)
		}
	}

	return outgoingSum.Sub(incomingSum)
}

// Balance calculates the balance of an Envelope in a specific month
// This code performs negative and positive rollover. See also
// https://github.com/envelope-zero/backend/issues/327
func (e Envelope) Balance(month time.Time) (decimal.Decimal, error) {
	// We add one month as the balance should include all transactions and the allocation for the present month
	// With that, we can query for all resources where the date/month is < the month
	month = time.Date(month.Year(), month.AddDate(0, 1, 0).Month(), 1, 0, 0, 0, 0, time.UTC)

	// Sum of incoming transactions
	var incoming decimal.NullDecimal
	err := database.DB.
		Table("transactions").
		Select("SUM(amount)").
		Joins("JOIN accounts source_account ON transactions.source_account_id = source_account.id AND source_account.deleted_at IS NULL").
		Joins("JOIN accounts destination_account ON transactions.destination_account_id = destination_account.id AND destination_account.deleted_at IS NULL").
		Where("source_account.external = 1 AND destination_account.external = 0 AND transactions.envelope_id = ?", e.ID).
		Where("transactions.date < date(?) ", month).
		Find(&incoming).Error
	if err != nil {
		return decimal.Zero, err
	}

	// If no transactions are found, the value is nil
	if !incoming.Valid {
		incoming.Decimal = decimal.Zero
	}

	// Sum of outgoing transactions
	var outgoing decimal.NullDecimal
	err = database.DB.
		Table("transactions").
		Select("SUM(amount)").
		Joins("JOIN accounts source_account ON transactions.source_account_id = source_account.id AND source_account.deleted_at IS NULL").
		Joins("JOIN accounts destination_account ON transactions.destination_account_id = destination_account.id AND destination_account.deleted_at IS NULL").
		Where("source_account.external = 0 AND destination_account.external = 1 AND transactions.envelope_id = ?", e.ID).
		Where("transactions.date < date(?) ", month).
		Find(&outgoing).Error
	if err != nil {
		return decimal.Zero, err
	}

	// If no transactions are found, the value is nil
	if !outgoing.Valid {
		outgoing.Decimal = decimal.Zero
	}

	var budgeted decimal.NullDecimal
	err = database.DB.
		Select("SUM(amount)").
		Where("allocations.envelope_id = ?", e.ID).
		Where("allocations.month < date(?) ", month).
		Table("allocations").
		Find(&budgeted).
		Error
	if err != nil {
		return decimal.Zero, err
	}

	// If no transactions are found, the value is nil
	if !budgeted.Valid {
		budgeted.Decimal = decimal.Zero
	}

	return budgeted.Decimal.Add(incoming.Decimal).Sub(outgoing.Decimal), nil
}

// Month calculates the month specific values for an envelope and returns an EnvelopeMonth for them.
func (e Envelope) Month(t time.Time) (EnvelopeMonth, error) {
	spent := e.Spent(t)
	month := time.Date(t.UTC().Year(), t.UTC().Month(), 1, 0, 0, 0, 0, time.UTC)

	envelopeMonth := EnvelopeMonth{
		ID:         e.ID,
		Name:       e.Name,
		Month:      month,
		Spent:      spent,
		Balance:    decimal.NewFromFloat(0),
		Allocation: decimal.NewFromFloat(0),
	}

	var allocation Allocation
	err := database.DB.First(&allocation, &Allocation{
		AllocationCreate: AllocationCreate{
			EnvelopeID: e.ID,
			Month:      month,
		},
	}).Error

	// If an unexpected error occurs, return
	if err != nil && err != gorm.ErrRecordNotFound {
		return EnvelopeMonth{}, err
	}

	envelopeMonth.Balance, err = e.Balance(month)
	if err != nil {
		return EnvelopeMonth{}, err
	}

	envelopeMonth.Allocation = allocation.Amount
	return envelopeMonth, nil
}
