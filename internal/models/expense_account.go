package models

// ExpenseAccount represents an asset account, e.g. a bank account
type ExpenseAccount struct {
	Model
	Name     string `json:"name"`
	BudgetID int
	Budget   Budget
}

// CreateExpenseAccount defines all values required to create an new account
type CreateExpenseAccount struct {
	Name string `json:"name" binding:"required"`
}
