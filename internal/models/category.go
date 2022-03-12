package models

// Category represents a category of envelopes
type Category struct {
	Model
	Name     string `json:"name"`
	BudgetID int    `json:"budgetId"`
	Budget   Budget `json:"-"`
	Note     string `json:"note,omitempty"`
}
