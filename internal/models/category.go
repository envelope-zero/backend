package models

// Category represents a category of envelopes
type Category struct {
	Model
	Name     string `json:"name"`
	BudgetID int    `json:"budgetId"`
	Budget   Budget `json:"-"`
}

// CreateCategory defines all values required to create a new category
type CreateCategory struct {
	Name string `json:"name" binding:"required"`
}
