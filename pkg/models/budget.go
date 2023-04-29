package models

import (
	"fmt"

	"github.com/envelope-zero/backend/v2/internal/types"
	"github.com/envelope-zero/backend/v2/pkg/database"
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
	Balance decimal.Decimal `json:"balance" gorm:"-" example:"3423.42"` // DEPRECATED. Will be removed in API v2, see https://github.com/envelope-zero/backend/issues/526.
	Links   struct {
		Self             string `json:"self" example:"https://example.com/api/v1/budgets/550dc009-cea6-4c12-b2a5-03446eb7b7cf"`
		Accounts         string `json:"accounts" example:"https://example.com/api/v1/accounts?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`
		Categories       string `json:"categories" example:"https://example.com/api/v1/categories?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`
		Envelopes        string `json:"envelopes" example:"https://example.com/api/v1/envelopes?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`
		Transactions     string `json:"transactions" example:"https://example.com/api/v1/transactions?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`
		Month            string `json:"month" example:"https://example.com/api/v1/budgets/550dc009-cea6-4c12-b2a5-03446eb7b7cf/YYYY-MM"`                        // This uses 'YYYY-MM' for clients to replace with the actual year and month.
		GroupedMonth     string `json:"groupedMonth" example:"https://example.com/api/v1/months?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf&month=YYYY-MM"`     // This uses 'YYYY-MM' for clients to replace with the actual year and month.
		MonthAllocations string `json:"monthAllocations" example:"https://example.com/api/v1/months?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf&month=YYYY-MM"` // This uses 'YYYY-MM' for clients to replace with the actual year and month.
	} `json:"links" gorm:"-"`
}

func (b Budget) Self() string {
	return "Budget"
}

type BudgetCreate struct {
	Name     string `json:"name" example:"Morre's Budget" default:""`
	Note     string `json:"note" example:"My personal expenses" default:""`
	Currency string `json:"currency" example:"€" default:""`
}

type BudgetMonth struct {
	ID        uuid.UUID       `json:"id" example:"1e777d24-3f5b-4c43-8000-04f65f895578"` // The ID of the Budget
	Name      string          `json:"name" example:"Groceries"`                          // The name of the Budget
	Month     types.Month     `json:"month" example:"2006-05-01T00:00:00.000000Z"`
	Budgeted  decimal.Decimal `json:"budgeted" example:"2100"`
	Income    decimal.Decimal `json:"income" example:"2317.34"`
	Available decimal.Decimal `json:"available" example:"217.34"`
	Envelopes []EnvelopeMonth `json:"envelopes"`
}

func (b BudgetMonth) Self() string {
	return "BudgetMonth"
}

func (b *Budget) AfterFind(tx *gorm.DB) (err error) {
	b.links(tx)

	b.Balance = decimal.Zero

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
		if account, err = account.WithCalculations(tx); err != nil {
			return err
		}

		b.Balance = b.Balance.Add(account.Balance)
	}

	return
}

// AfterSave does the same as AfterFind, so we just call it.
func (b *Budget) AfterSave(tx *gorm.DB) (err error) {
	return b.AfterFind(tx)
}

func (b *Budget) links(tx *gorm.DB) {
	url := tx.Statement.Context.Value(database.ContextURL)

	b.Links.Self = fmt.Sprintf("%s/v1/budgets/%s", url, b.ID)
	b.Links.Month = b.Links.Self + "/YYYY-MM"
	b.Links.Accounts = fmt.Sprintf("%s/v1/accounts?budget=%s", url, b.ID)
	b.Links.Categories = fmt.Sprintf("%s/v1/categories?budget=%s", url, b.ID)
	b.Links.Envelopes = fmt.Sprintf("%s/v1/envelopes?budget=%s", url, b.ID)
	b.Links.Transactions = fmt.Sprintf("%s/v1/transactions?budget=%s", url, b.ID)
	b.Links.GroupedMonth = fmt.Sprintf("%s/v1/months?budget=%s&month=YYYY-MM", url, b.ID)
	b.Links.MonthAllocations = fmt.Sprintf("%s/v1/months?budget=%s&month=YYYY-MM", url, b.ID)
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

type CategoryEnvelopes struct {
	Category
	Envelopes  []EnvelopeMonth `json:"envelopes"`                // Slice of all envelopes
	Balance    decimal.Decimal `json:"balance" example:"-10.13"` // Sum of the balances of the envelopes
	Allocation decimal.Decimal `json:"allocation" example:"90"`  // Sum of allocations for the envelopes
	Spent      decimal.Decimal `json:"spent" example:"100.13"`   // Sum spent for all envelopes
}

type Month struct {
	ID         uuid.UUID           `json:"id" example:"1e777d24-3f5b-4c43-8000-04f65f895578"` // The ID of the Budget
	Name       string              `json:"name" example:"Zero budget"`                        // The name of the Budget
	Month      types.Month         `json:"month" example:"2006-05-01T00:00:00.000000Z"`       // The month
	Budgeted   decimal.Decimal     `json:"budgeted" example:"2100"`                           // The sum of all allocations for the month. **Deprecated, please use the `allocation` field**
	Income     decimal.Decimal     `json:"income" example:"2317.34"`                          // The total income for the month (sum of all incoming transactions without an Envelope)
	Available  decimal.Decimal     `json:"available" example:"217.34"`                        // The amount available to budget
	Balance    decimal.Decimal     `json:"balance" example:"5231.37"`                         // The sum of all envelope balances
	Spent      decimal.Decimal     `json:"spent" example:"133.70"`                            // The amount of money spent in this month
	Allocation decimal.Decimal     `json:"allocation" example:"1200.50"`                      // The sum of all allocations for this month
	Categories []CategoryEnvelopes `json:"categories"`                                        // A list of envelope month calculations grouped by category
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

			// FIXME: The remove this with the integration of allocations into MonthConfigs.
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
