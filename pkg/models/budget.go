package models

import (
	"fmt"
	"strings"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/google/uuid"
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
	var allocations []Allocation
	err = db.
		Joins("JOIN envelopes ON allocations.envelope_id = envelopes.id AND envelopes.deleted_at IS NULL").
		Joins("JOIN categories ON envelopes.category_id = categories.id AND categories.deleted_at IS NULL").
		Joins("JOIN budgets ON categories.budget_id = budgets.id AND budgets.deleted_at IS NULL").
		Where("budgets.id = ?", b.ID).
		Where("allocations.month >= date(?)", month).
		Where("allocations.month < date(?)", month.AddDate(0, 1)).
		Find(&allocations).
		Error
	if err != nil {
		return decimal.Zero, err
	}

	for _, a := range allocations {
		allocated = allocated.Add(a.Amount)
	}

	return
}

// Month calculates the month overview for this month.
func (b Budget) Month(db *gorm.DB, month types.Month) (Month, error) {
	result := Month{
		ID:    b.ID,
		Name:  b.Name,
		Month: month,
	}

	// Add budgeted sum to response
	budgeted, err := b.Allocated(db, result.Month)
	if err != nil {
		return Month{}, err
	}
	result.Budgeted = budgeted
	result.Allocation = budgeted

	// Add income to response
	income, err := b.Income(db, result.Month)
	if err != nil {
		return Month{}, err
	}
	result.Income = income

	// Get all categories for the budget
	var categories []Category
	err = db.Where(&Category{CategoryCreate: CategoryCreate{BudgetID: b.ID}}).Find(&categories).Error
	if err != nil {
		return Month{}, err
	}

	result.Categories = make([]CategoryEnvelopes, 0)
	result.Balance = decimal.Zero

	// Get envelopes for all categories
	for _, category := range categories {
		var categoryEnvelopes CategoryEnvelopes

		// Set the basic category values
		categoryEnvelopes.Category = category
		categoryEnvelopes.Envelopes = make([]EnvelopeMonth, 0)

		var envelopes []Envelope

		err = db.Where(&Envelope{
			EnvelopeCreate: EnvelopeCreate{
				CategoryID: category.ID,
			},
		}).Find(&envelopes).Error
		if err != nil {
			return Month{}, err
		}

		for _, envelope := range envelopes {
			envelopeMonth, allocationID, err := envelope.Month(db, result.Month)
			if err != nil {
				return Month{}, err
			}

			// Update the month's summarized data
			result.Balance = result.Balance.Add(envelopeMonth.Balance)
			result.Spent = result.Spent.Add(envelopeMonth.Spent)

			// Update the category's summarized data
			categoryEnvelopes.Balance = categoryEnvelopes.Balance.Add(envelopeMonth.Balance)
			categoryEnvelopes.Spent = categoryEnvelopes.Spent.Add(envelopeMonth.Spent)
			categoryEnvelopes.Allocation = categoryEnvelopes.Allocation.Add(envelopeMonth.Allocation)

			// TODO: The remove this with the integration of allocations into MonthConfigs.
			url := db.Statement.Context.Value(database.ContextURL)

			// Set the allocation link. If there is no allocation, we send the collection endpoint.
			// With this, any client will be able to see that the "Budgeted" amount is 0 and therefore
			// send a HTTP POST for creation instead of a patch.
			envelopeMonth.Links.Allocation = fmt.Sprintf("%s/v1/allocations", url)
			if allocationID != uuid.Nil {
				envelopeMonth.Links.Allocation = fmt.Sprintf("%s/%s", envelopeMonth.Links.Allocation, allocationID)
			}

			categoryEnvelopes.Envelopes = append(categoryEnvelopes.Envelopes, envelopeMonth)
		}

		result.Categories = append(result.Categories, categoryEnvelopes)
	}

	// Available amount is the sum of balances of all on-budget accounts, then subtract the sum of all envelope balances
	result.Available = result.Balance.Neg()

	// Get all on budget accounts for the budget
	var accounts []Account
	err = db.Where(&Account{AccountCreate: AccountCreate{BudgetID: b.ID, OnBudget: true}}).Find(&accounts).Error
	if err != nil {
		return Month{}, err
	}

	// Add all on-balance accounts to the available sum
	for _, a := range accounts {
		_, available, err := a.GetBalanceMonth(db, month)
		if err != nil {
			return Month{}, err
		}
		result.Available = result.Available.Add(available)
	}

	return result, nil
}
