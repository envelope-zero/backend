package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Budget represents a budget
//
// A budget is the highest level of organization in Envelope Zero, all other
// resources reference it directly or transitively.
type Budget struct {
	Model
	BudgetCreate
	Balance decimal.Decimal `json:"balance" gorm:"-" example:"3423.42"`
}

type BudgetCreate struct {
	Name     string `json:"name,omitempty" example:"Morre's Budget" default:""`
	Note     string `json:"note,omitempty" example:"My personal expenses" default:""`
	Currency string `json:"currency,omitempty" example:"€" default:""`
}

type BudgetMonth struct {
	ID        uuid.UUID       `json:"id" example:"1e777d24-3f5b-4c43-8000-04f65f895578"` // The ID of the Envelope
	Name      string          `json:"name" example:"Groceries"`                          // The name of the Envelope
	Month     time.Time       `json:"month" example:"2006-05-01T00:00:00.000000Z"`       // This is always set to 00:00 UTC on the first of the month.
	Envelopes []EnvelopeMonth `json:"envelopes,omitempty"`
}
