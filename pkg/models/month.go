package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CategoryEnvelopes struct {
	ID        uuid.UUID       `json:"id" example:"dafd9a74-6aeb-46b9-9f5a-cfca624fea85"`
	Name      string          `json:"name" example:"Rainy Day Funds" default:""`
	Envelopes []EnvelopeMonth `json:"envelopes"`
}

type Month struct {
	ID         uuid.UUID           `json:"id" example:"1e777d24-3f5b-4c43-8000-04f65f895578"` // The ID of the Budget
	Name       string              `json:"name" example:"Zero budget"`                        // The name of the Budget
	Month      time.Time           `json:"month" example:"2006-05-01T00:00:00.000000Z"`       // This is always set to 00:00 UTC on the first of the month.
	Budgeted   decimal.Decimal     `json:"budgeted" example:"2100"`                           // The sum of all allocations for the month
	Income     decimal.Decimal     `json:"income" example:"2317.34"`                          // The total income for the month (sum of all incoming transactions without an Envelope)
	Available  decimal.Decimal     `json:"available" example:"217.34"`                        // The amount available to budget
	Balance    decimal.Decimal     `json:"balance" example:"5231.37"`                         // The sum of all envelope balances
	Spent      decimal.Decimal     `json:"spent" example:"133.70"`                            // The amount of money spent in this month
	Categories []CategoryEnvelopes `json:"categories"`                                        // A list of envelope month calculations grouped by category
}
