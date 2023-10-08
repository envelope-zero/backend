package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AccountCreate struct {
	Name               string          `json:"name" example:"Cash" default:"" gorm:"uniqueIndex:account_name_budget_id"`                          // Name of the account
	Note               string          `json:"note" example:"Money in my wallet" default:""`                                                      // A longer description for the account
	BudgetID           uuid.UUID       `json:"budgetId" example:"550dc009-cea6-4c12-b2a5-03446eb7b7cf" gorm:"uniqueIndex:account_name_budget_id"` // ID of the budget this account belongs to
	OnBudget           bool            `json:"onBudget" example:"true" default:"false"`                                                           // Does the account factor into the available budget? Always false when external: true
	External           bool            `json:"external" example:"false" default:"false"`                                                          // Does the account belong to the budget owner or not?
	InitialBalance     decimal.Decimal `json:"initialBalance" example:"173.12" default:"0"`                                                       // Balance of the account before any transactions were recorded
	InitialBalanceDate *time.Time      `json:"initialBalanceDate" example:"2017-05-12T00:00:00Z"`                                                 // Date of the initial balance
	Hidden             bool            `json:"hidden" example:"true" default:"false"`                                                             // Is the account archived?
	ImportHash         string          `json:"importHash" example:"867e3a26dc0baf73f4bff506f31a97f6c32088917e9e5cf1a5ed6f3f84a6fa70" default:""`  // The SHA256 hash of a unique combination of values to use in duplicate detection
}
