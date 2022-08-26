package models

import (
	"fmt"
	"time"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Budget represents a budget
//
// A budget is the highest level of organization in Envelope Zero, all other
// resources reference it directly or transitively.
type Budget struct {
	Model
	BudgetCreate
	Balance decimal.Decimal `json:"balance" gorm:"-" example:"3423.42"`
}

type BudgetCreate struct {
	Name     string `json:"name" example:"Morre's Budget" default:""`
	Note     string `json:"note" example:"My personal expenses" default:""`
	Currency string `json:"currency" example:"€" default:""`
}

type BudgetMonth struct {
	ID        uuid.UUID       `json:"id" example:"1e777d24-3f5b-4c43-8000-04f65f895578"` // The ID of the Envelope
	Name      string          `json:"name" example:"Groceries"`                          // The name of the Envelope
	Month     time.Time       `json:"month" example:"2006-05-01T00:00:00.000000Z"`       // This is always set to 00:00 UTC on the first of the month.
	Budgeted  decimal.Decimal `json:"budgeted" example:"2100"`
	Income    decimal.Decimal `json:"income" example:"2317.34"`
	Envelopes []EnvelopeMonth `json:"envelopes"`
}

// WithCalculations computes all the calculated values.
func (b Budget) WithCalculations() Budget {
	// Get all OnBudget accounts for the budget
	var accounts []Account
	_ = database.DB.Where(&Account{
		AccountCreate: AccountCreate{
			BudgetID: b.ID,
			OnBudget: true,
		},
	}).Find(&accounts)

	// Add all their balances to the budget's balance
	for _, account := range accounts {
		fmt.Println(account.WithCalculations().Balance)
		b.Balance = b.Balance.Add(account.WithCalculations().Balance)
	}

	return b
}

// Income returns the income for a budget in a given month.
func (b Budget) Income(t time.Time) (decimal.Decimal, error) {
	var income decimal.NullDecimal

	err := database.DB.
		Select("SUM(amount)").
		Joins("JOIN accounts source_account ON transactions.source_account_id = source_account.id AND source_account.deleted_at IS NULL").
		Joins("JOIN accounts destination_account ON transactions.destination_account_id = destination_account.id AND destination_account.deleted_at IS NULL").
		Where("source_account.external = 1").
		Where("destination_account.external = 0").
		Where("transactions.envelope_id IS NULL").
		Where("strftime('%m', transactions.date) = ?", fmt.Sprintf("%02d", t.Month())).
		Where("strftime('%Y', transactions.date) = ?", fmt.Sprintf("%d", t.Year())).
		Where(&Transaction{
			TransactionCreate: TransactionCreate{
				BudgetID: b.ID,
			},
		}).
		Table("transactions").
		Find(&income).
		Error
	if err != nil {
		return decimal.Zero, err
	}

	// If no transactions are found, the value is nil
	if !income.Valid {
		return decimal.NewFromFloat(0), nil
	}

	return income.Decimal, nil
}

// TotalIncome calculates the total income over all time.
func (b Budget) TotalIncome() (decimal.Decimal, error) {
	var income decimal.NullDecimal
	err := database.DB.
		Select("SUM(amount)").
		Joins("JOIN accounts source_account ON transactions.source_account_id = source_account.id AND source_account.deleted_at IS NULL").
		Joins("JOIN accounts destination_account ON transactions.destination_account_id = destination_account.id AND destination_account.deleted_at IS NULL").
		Where("source_account.external = 1").
		Where("destination_account.external = 0").
		Where("transactions.envelope_id IS NULL").
		Where(&Transaction{
			TransactionCreate: TransactionCreate{
				BudgetID: b.ID,
			},
		}).
		Table("transactions").
		Find(&income).
		Error
	if err != nil {
		return decimal.Zero, err
	}

	// If no transactions are found, the value is nil
	if !income.Valid {
		return decimal.NewFromFloat(0), nil
	}

	return income.Decimal, nil
}

// TotalBudgeted calculates the total sum that has been budgeted over all time.
func (b Budget) TotalBudgeted() (decimal.Decimal, error) {
	var budgeted decimal.NullDecimal
	err := database.DB.
		Select("SUM(amount)").
		Joins("JOIN envelopes ON allocations.envelope_id = envelopes.id AND envelopes.deleted_at IS NULL").
		Joins("JOIN categories ON envelopes.category_id = categories.id AND categories.deleted_at IS NULL").
		Joins("JOIN budgets ON categories.budget_id = budgets.id AND budgets.deleted_at IS NULL").
		Where("budgets.id = ?", b.ID).
		Table("allocations").
		Find(&budgeted).
		Error
	if err != nil {
		return decimal.Zero, err
	}

	// If no transactions are found, the value is nil
	if !budgeted.Valid {
		return decimal.NewFromFloat(0), nil
	}

	return budgeted.Decimal, nil
}
