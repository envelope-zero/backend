package models

// Account represents an asset account, e.g. a bank account
type Account struct {
	Model
	Name     string `json:"name"`
	BudgetID int    `json:"budgetId"`
	Budget   Budget `json:"-"`
	OnBudget bool   `json:"onBudget"`
	Visible  bool   `json:"visible"`
}

// CreateAccount defines all values required to create a new asset account
type CreateAccount struct {
	Name     string `json:"name" binding:"required"`
	OnBudget bool   `json:"onBudget"`
	Visible  bool   `json:"visible"`
}
