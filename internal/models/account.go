package models

// Account represents an asset account, e.g. a bank account
type Account struct {
	Model
	Name     string `json:"name"`
	BudgetID int
	Budget   Budget
}

// CreateAccount defines all values required to create an new account
type CreateAccount struct {
	Name string `json:"name" binding:"required"`
}
