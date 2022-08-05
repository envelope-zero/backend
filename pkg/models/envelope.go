package models

import (
	"time"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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

	return incomingSum.Sub(outgoingSum)
}

// Month calculates the month specific values for an envelope and returns an EnvelopeMonth for them.
func (e Envelope) Month(t time.Time) EnvelopeMonth {
	spent := e.Spent(t)

	var allocation Allocation
	database.DB.First(&allocation, &Allocation{
		AllocationCreate: AllocationCreate{
			Month: time.Date(t.UTC().Year(), t.UTC().Month(), 1, 0, 0, 0, 0, time.UTC),
		},
	})

	balance := allocation.Amount.Add(spent)

	return EnvelopeMonth{
		ID:         e.ID,
		Name:       e.Name,
		Month:      allocation.Month,
		Spent:      spent,
		Balance:    balance,
		Allocation: allocation.Amount,
	}
}
