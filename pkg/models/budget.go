package models

import (
	"strings"

	"github.com/envelope-zero/backend/v4/internal/types"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Budget represents a budget
//
// A budget is the highest level of organization in Envelope Zero, all other
// resources reference it directly or transitively.
type Budget struct {
	DefaultModel
	BudgetCreate
}

func (b Budget) Self() string {
	return "Budget"
}

type BudgetCreate struct {
	Name     string `json:"name" example:"Morre's Budget" default:""`       // Name of the budget
	Note     string `json:"note" example:"My personal expenses" default:""` // A longer description of the budget
	Currency string `json:"currency" example:"€" default:""`                // The currency for the budget
}

func (b *Budget) BeforeSave(_ *gorm.DB) error {
	b.Name = strings.TrimSpace(b.Name)
	b.Note = strings.TrimSpace(b.Note)
	b.Currency = strings.TrimSpace(b.Currency)

	return nil
}

// Balance calculates the balance for a budget.
func (b Budget) Balance(tx *gorm.DB) (balance decimal.Decimal, err error) {
	// Get all OnBudget accounts for the budget
	var accounts []Account
	_ = tx.Where(&Account{
		AccountCreate: AccountCreate{
			BudgetID: b.ID,
			OnBudget: true,
		},
	}).Find(&accounts)

	// Add all their balances to the budget's balance
	for _, account := range accounts {
		aBalance, _, err := account.GetBalanceMonth(tx, types.Month{})
		if err != nil {
			return decimal.Zero, err
		}

		balance = balance.Add(aBalance)
	}

	return balance, nil
}

// Income returns the income for a budget in a given month.
func (b Budget) Income(db *gorm.DB, month types.Month) (income decimal.Decimal, err error) {
	var transactions []Transaction

	err = db.
		Joins("JOIN accounts source_account ON transactions.source_account_id = source_account.id AND source_account.deleted_at IS NULL").
		Joins("JOIN accounts destination_account ON transactions.destination_account_id = destination_account.id AND destination_account.deleted_at IS NULL").
		Where("source_account.external = 1").
		Where("destination_account.external = 0").
		Where("transactions.envelope_id IS NULL").
		Where("transactions.available_from >= date(?) AND transactions.available_from < date(?)", month, month.AddDate(0, 1)).
		Where(&Transaction{
			TransactionCreate: TransactionCreate{
				BudgetID: b.ID,
			},
		}).
		Find(&transactions).
		Error
	if err != nil {
		return decimal.Zero, err
	}

	for _, t := range transactions {
		income = income.Add(t.Amount)
	}

	return
}

// Allocated calculates the sum that has been budgeted for a specific month.
func (b Budget) Allocated(db *gorm.DB, month types.Month) (allocated decimal.Decimal, err error) {
	var monthConfigs []MonthConfig
	err = db.
		Joins("JOIN envelopes ON month_configs.envelope_id = envelopes.id AND envelopes.deleted_at IS NULL").
		Joins("JOIN categories ON envelopes.category_id = categories.id AND categories.deleted_at IS NULL").
		Joins("JOIN budgets ON categories.budget_id = budgets.id AND budgets.deleted_at IS NULL").
		Where("budgets.id = ?", b.ID).
		Where("month_configs.month >= date(?)", month).
		Where("month_configs.month < date(?)", month.AddDate(0, 1)).
		Find(&monthConfigs).
		Error
	if err != nil {
		return decimal.Zero, err
	}

	for _, a := range monthConfigs {
		allocated = allocated.Add(a.Allocation)
	}

	return
}
