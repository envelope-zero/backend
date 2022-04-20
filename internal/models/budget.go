package models

import "time"

// Budget represents a budget
//
// A budget is the highest level of organization in Envelope Zero, all other
// resources reference it directly or transitively.
type Budget struct {
	Model
	Name     string `json:"name,omitempty"`
	Note     string `json:"note,omitempty"`
	Currency string `json:"currency,omitempty"`
}

type BudgetMonth struct {
	ID        uint64          `json:"id"`
	Name      string          `json:"name"`
	Month     time.Time       `json:"month"`
	Envelopes []EnvelopeMonth `json:"envelopes,omitempty"`
}
