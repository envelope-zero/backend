package models

import "github.com/google/uuid"

// Category represents a category of envelopes.
type Category struct {
	Model
	CategoryCreate
	Budget Budget `json:"-"`
}

type CategoryCreate struct {
	Name     string    `json:"name,omitempty" example:"Saving" default:""`
	BudgetID uuid.UUID `json:"budgetId" example:"52d967d3-33f4-4b04-9ba7-772e5ab9d0ce"`
	Note     string    `json:"note,omitempty" example:"All envelopes for long-term saving" default:""`
}
