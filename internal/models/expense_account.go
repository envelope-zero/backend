package models

// ExpenseAccount represents an expense account, e.g. the supermarket you
// buy your groceries at
type ExpenseAccount struct {
	Model
	Name     string `json:"name"`
	BudgetID int
	Budget   Budget
}

// CreateExpenseAccount defines all values required to create a new expense account
type CreateExpenseAccount struct {
	Name string `json:"name" binding:"required"`
}
