package models

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// Envelope represents an envelope in your budget.
type Envelope struct {
	Model
	Name       string   `json:"name,omitempty"`
	CategoryID uint64   `json:"categoryId"`
	Category   Category `json:"-"`
	Note       string   `json:"note,omitempty"`
}

// Spent returns the amount spent for the month the time.Time instance is in.
func (e Envelope) Spent(t time.Time) (decimal.Decimal, error) {
	// All transactions where the Envelope ID matches and that have an external account as source and an internal account as destination
	incoming, err := RawTransactions(
		fmt.Sprintf("SELECT transactions.* FROM transactions, accounts AS source_accounts, accounts AS destination_accounts WHERE transactions.source_account_id = source_accounts.id AND source_accounts.external AND transactions.destination_account_id = destination_accounts.id AND NOT destination_accounts.external AND transactions.envelope_id = %v", e.ID),
	)
	if err != nil {
		return decimal.Zero, err
	}

	// Add all incoming transactions that are in the correct month
	incomingSum := decimal.Zero
	for _, transaction := range incoming {
		if transaction.Date.UTC().Year() == t.UTC().Year() && transaction.Date.UTC().Month() == t.UTC().Month() {
			incomingSum = incomingSum.Add(transaction.Amount)
		}
	}

	outgoing, err := RawTransactions(
		// All transactions where the envelope ID matches that have an internal account as source and an external account as destination
		fmt.Sprintf("SELECT transactions.* FROM transactions, accounts AS source_accounts, accounts AS destination_accounts WHERE transactions.source_account_id = source_accounts.id AND NOT source_accounts.external AND transactions.destination_account_id = destination_accounts.id AND destination_accounts.external AND transactions.envelope_id = %v", e.ID),
	)
	if err != nil {
		return decimal.Zero, err
	}

	// Add all outgoing transactions that are in the correct month
	outgoingSum := decimal.Zero
	for _, transaction := range outgoing {
		if transaction.Date.UTC().Year() == t.UTC().Year() && transaction.Date.UTC().Month() == t.UTC().Month() {
			outgoingSum = outgoingSum.Add(transaction.Amount)
		}
	}

	return incomingSum.Sub(outgoingSum), nil
}
