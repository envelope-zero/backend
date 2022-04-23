package models

import (
	"time"
)

// Budget represents a budget
//
// A budget is the highest level of organization in Envelope Zero, all other
// resources reference it directly or transitively.
type Budget struct {
	Model
	BudgetCreate
}

type BudgetCreate struct {
	Name     string `json:"name,omitempty" example:"My First Budget"`
	Note     string `json:"note,omitempty" example:"A description so I remember what this was for"`
	Currency string `json:"currency,omitempty" example:"€"`
}

type BudgetMonth struct {
	ID        uint64          `json:"id" example:"23"`
	Name      string          `json:"name" example:"A test envelope"`
	Month     time.Time       `json:"month" example:"2006-05-04T15:02:01.000000Z"`
	Envelopes []EnvelopeMonth `json:"envelopes,omitempty"`
}
