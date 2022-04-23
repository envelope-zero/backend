package models

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// Envelope represents an envelope in your budget.
type Envelope struct {
	Model
	EnvelopeCreate
	Category Category `json:"-"`
}

type EnvelopeCreate struct {
	Name       string `json:"name,omitempty"`
	CategoryID uint64 `json:"categoryId"`
	Note       string `json:"note,omitempty"`
}

// EnvelopeMonth contains data about an Envelope for a specific month.
type EnvelopeMonth struct {
	ID         uint64          `json:"id"`
	Name       string          `json:"name"`
	Month      time.Time       `json:"month"`
	Spent      decimal.Decimal `json:"spent"`
	Balance    decimal.Decimal `json:"balance"`
	Allocation decimal.Decimal `json:"allocation"`
}

// Spent returns the amount spent for the month the time.Time instance is in.
func (e Envelope) Spent(t time.Time) decimal.Decimal {
	// All transactions where the Envelope ID matches and that have an external account as source and an internal account as destination
	incoming, _ := RawTransactions(
		fmt.Sprintf("SELECT transactions.* FROM transactions, accounts AS source_accounts, accounts AS destination_accounts WHERE transactions.source_account_id = source_accounts.id AND source_accounts.external AND transactions.destination_account_id = destination_accounts.id AND NOT destination_accounts.external AND transactions.envelope_id = %v", e.ID),
	)

	// Add all incoming transactions that are in the correct month
	incomingSum := decimal.Zero
	for _, transaction := range incoming {
		if transaction.Date.UTC().Year() == t.UTC().Year() && transaction.Date.UTC().Month() == t.UTC().Month() {
			incomingSum = incomingSum.Add(transaction.Amount)
		}
	}

	outgoing, _ := RawTransactions(
		// All transactions where the envelope ID matches that have an internal account as source and an external account as destination
		fmt.Sprintf("SELECT transactions.* FROM transactions, accounts AS source_accounts, accounts AS destination_accounts WHERE transactions.source_account_id = source_accounts.id AND NOT source_accounts.external AND transactions.destination_account_id = destination_accounts.id AND destination_accounts.external AND transactions.envelope_id = %v", e.ID),
	)

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
	DB.First(&allocation, &Allocation{
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
