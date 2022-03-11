package models

import "github.com/shopspring/decimal"

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

// Transactions returns all transactions for this account
func (a Account) Transactions() []Transaction {
	var transactions []Transaction

	// Get all transactions where the account is either the source or the destination
	DB.Where(
		"source_account_id = ?", a.ID,
	).Or("destination_account_id = ?", a.ID).Find(&transactions)

	return transactions
}

// Balance returns the Balance of the account, including all transactions.
//
// Note that this will produce wrong results with sqlite as of now, see
// https://github.com/go-gorm/gorm/issues/5153 for details.
func (a Account) Balance() decimal.Decimal {
	var sourceSum, destinationSum decimal.Decimal

	DB.Table("transactions").Where("source_account_id = ?", a.ID).Select(
		"SUM(amount)",
	).Row().Scan(&sourceSum)

	DB.Table("transactions").Where("destination_account_id = ?", a.ID).Select(
		"SUM(amount)",
	).Row().Scan(&destinationSum)

	return destinationSum.Sub(sourceSum)
}
