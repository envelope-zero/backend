package models

// ExternalAccount represents an expense account, e.g. the supermarket you
// buy your groceries at
type ExternalAccount struct {
	Model
	Name     string `json:"name"`
	BudgetID int    `json:"budgetId"`
}

// CreateExternalAccount defines all values required to create a new expense account
type CreateExternalAccount struct {
	Name string `json:"name" binding:"required"`
}
