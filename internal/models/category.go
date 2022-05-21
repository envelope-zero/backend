package models

import "github.com/google/uuid"

// Category represents a category of envelopes.
type Category struct {
	Model
	CategoryCreate
	Budget Budget `json:"-"`
}

type CategoryCreate struct {
	Name     string    `json:"name,omitempty"`
	BudgetID uuid.UUID `json:"budgetId"`
	Note     string    `json:"note,omitempty"`
}
