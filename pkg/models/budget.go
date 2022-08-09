package models

import (
	"fmt"
	"time"

	"github.com/envelope-zero/backend/internal/database"
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
	Name     string `json:"name" example:"Morre's Budget" default:""`
	Note     string `json:"note" example:"My personal expenses" default:""`
	Currency string `json:"currency" example:"€" default:""`
}

type BudgetMonth struct {
	ID        uuid.UUID       `json:"id" example:"1e777d24-3f5b-4c43-8000-04f65f895578"` // The ID of the Envelope
	Name      string          `json:"name" example:"Groceries"`                          // The name of the Envelope
	Month     time.Time       `json:"month" example:"2006-05-01T00:00:00.000000Z"`       // This is always set to 00:00 UTC on the first of the month.
	Budgeted  decimal.Decimal `json:"budgeted" example:"2100"`
	Envelopes []EnvelopeMonth `json:"envelopes"`
}

// WithCalculations computes all the calculated values.
func (b Budget) WithCalculations() Budget {
	// Get all OnBudget accounts for the budget
	var accounts []Account
	_ = database.DB.Where(&Account{
		AccountCreate: AccountCreate{
			BudgetID: b.ID,
			OnBudget: true,
		},
	}).Find(&accounts)

	// Add all their balances to the budget's balance
	for _, account := range accounts {
		fmt.Println(account.WithCalculations().Balance)
		b.Balance = b.Balance.Add(account.WithCalculations().Balance)
	}

	return b
}
