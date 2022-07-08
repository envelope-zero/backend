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
	Name       string    `json:"name,omitempty"`
	CategoryID uuid.UUID `json:"categoryId"`
	Note       string    `json:"note,omitempty"`
}

// EnvelopeMonth contains data about an Envelope for a specific month.
type EnvelopeMonth struct {
	ID         uuid.UUID       `json:"id"`
	Name       string          `json:"name"`
	Month      time.Time       `json:"month"`
	Spent      decimal.Decimal `json:"spent"`
	Balance    decimal.Decimal `json:"balance"`
	Allocation decimal.Decimal `json:"allocation"`
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
			Month: uint8(t.UTC().Month()),
			Year:  uint(t.UTC().Year()),
		},
	})

	balance := allocation.Amount.Add(spent)

	return EnvelopeMonth{
		ID:         e.ID,
		Name:       e.Name,
		Month:      time.Date(t.UTC().Year(), t.UTC().Month(), 1, 0, 0, 0, 0, time.UTC),
		Spent:      spent,
		Balance:    balance,
		Allocation: allocation.Amount,
	}
}
