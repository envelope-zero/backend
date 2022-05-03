package models

// Category represents a category of envelopes.
type Category struct {
	Model
	CategoryCreate
	Budget Budget `json:"-"`
}

type CategoryCreate struct {
	Name     string `json:"name,omitempty"`
	BudgetID uint64 `json:"budgetId"`
	Note     string `json:"note,omitempty"`
}
