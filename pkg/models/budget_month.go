package models

import (
	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type BudgetMonth struct {
	ID        uuid.UUID       `json:"id" example:"1e777d24-3f5b-4c43-8000-04f65f895578"` // The ID of the Budget
	Name      string          `json:"name" example:"Groceries"`                          // The name of the Budget
	Month     types.Month     `json:"month" example:"2006-05-01T00:00:00.000000Z"`       // Month these calculations are made for
	Budgeted  decimal.Decimal `json:"budgeted" example:"2100"`                           // Amount of money that has been allocated to envelopes
	Income    decimal.Decimal `json:"income" example:"2317.34"`                          // Income. This is all money that is sent from off-budget to on-budget accounts without an envelope set.
	Available decimal.Decimal `json:"available" example:"217.34"`                        // The amount of money still available to budget.
	Envelopes []EnvelopeMonth `json:"envelopes"`                                         // The envelopes this budget has, with detailed calculations
}

func (b BudgetMonth) Self() string {
	return "BudgetMonth"
}

type CategoryEnvelopes struct {
	Category
	Envelopes  []EnvelopeMonth `json:"envelopes"`                // Slice of all envelopes
	Balance    decimal.Decimal `json:"balance" example:"-10.13"` // Sum of the balances of the envelopes
	Allocation decimal.Decimal `json:"allocation" example:"90"`  // Sum of allocations for the envelopes
	Spent      decimal.Decimal `json:"spent" example:"100.13"`   // Sum spent for all envelopes
}

type Month struct {
	ID         uuid.UUID           `json:"id" example:"1e777d24-3f5b-4c43-8000-04f65f895578"` // The ID of the Budget
	Name       string              `json:"name" example:"Zero budget"`                        // The name of the Budget
	Month      types.Month         `json:"month" example:"2006-05-01T00:00:00.000000Z"`       // The month
	Budgeted   decimal.Decimal     `json:"budgeted" example:"2100"`                           // The sum of all allocations for the month. **Deprecated, please use the `allocation` field**
	Income     decimal.Decimal     `json:"income" example:"2317.34"`                          // The total income for the month (sum of all incoming transactions without an Envelope)
	Available  decimal.Decimal     `json:"available" example:"217.34"`                        // The amount available to budget
	Balance    decimal.Decimal     `json:"balance" example:"5231.37"`                         // The sum of all envelope balances
	Spent      decimal.Decimal     `json:"spent" example:"133.70"`                            // The amount of money spent in this month
	Allocation decimal.Decimal     `json:"allocation" example:"1200.50"`                      // The sum of all allocations for this month
	Categories []CategoryEnvelopes `json:"categories"`                                        // A list of envelope month calculations grouped by category
}
