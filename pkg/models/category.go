package models

import "github.com/google/uuid"

// Category represents a category of envelopes.
type Category struct {
	Model
	CategoryCreate
	Budget Budget `json:"-"`
}

type CategoryCreate struct {
	Name     string    `json:"name" gorm:"uniqueIndex:category_budget_name" example:"Saving" default:""`
	BudgetID uuid.UUID `json:"budgetId" gorm:"uniqueIndex:category_budget_name" example:"52d967d3-33f4-4b04-9ba7-772e5ab9d0ce"`
	Note     string    `json:"note" example:"All envelopes for long-term saving" default:""`
	Hidden   bool      `json:"hidden" example:"true" default:"false"`
}
