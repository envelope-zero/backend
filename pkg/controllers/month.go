package controllers

import (
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/internal/types"
	"github.com/envelope-zero/backend/pkg/httperrors"
	"github.com/envelope-zero/backend/pkg/httputil"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type MonthResponse struct {
	Data Month `json:"data"`
}

type CategoryEnvelopes struct {
	ID         uuid.UUID              `json:"id" example:"dafd9a74-6aeb-46b9-9f5a-cfca624fea85"` // ID of the category
	Name       string                 `json:"name" example:"Rainy Day Funds" default:""`         // Name of the category
	Envelopes  []models.EnvelopeMonth `json:"envelopes"`                                         // Slice of all envelopes
	Balance    decimal.Decimal        `json:"balance" example:"-10.13"`                          // Sum of the balances of the envelopes
	Allocation decimal.Decimal        `json:"allocation" example:"90"`                           // Sum of allocations for the envelopes
	Spent      decimal.Decimal        `json:"spent" example:"100.13"`                            // Sum spent for all envelopes
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

// parseMonthQuery takes in the context and parses the request
//
// It verifies that the requested budget exists and parses the ID to return
// the budget resource itself.
func (co Controller) parseMonthQuery(c *gin.Context) (types.Month, models.Budget, bool) {
	var query struct {
		QueryMonth
		BudgetID string `form:"budget" example:"81b0c9ce-6fd3-4e1e-becc-106055898a2a"`
	}

	if err := c.Bind(&query); err != nil {
		httperrors.Handler(c, err)
		return types.Month{}, models.Budget{}, false
	}

	if query.Month.IsZero() {
		httperrors.New(c, http.StatusBadRequest, "The month query parameter must be set")
		return types.Month{}, models.Budget{}, false
	}

	budgetID, err := uuid.Parse(query.BudgetID)
	if err != nil {
		httperrors.InvalidUUID(c)
		return types.Month{}, models.Budget{}, false
	}

	budget, ok := co.getBudgetResource(c, budgetID)
	if !ok {
		return types.Month{}, models.Budget{}, false
	}

	return types.MonthOf(query.Month), budget, true
}

// RegisterMonthRoutes registers the routes for months with
// the RouterGroup that is passed.
func (co Controller) RegisterMonthRoutes(r *gin.RouterGroup) {
	{
		r.OPTIONS("", co.OptionsMonth)
		r.GET("", co.GetMonth)
		r.POST("", co.SetAllocations)
		r.DELETE("", co.DeleteAllocations)
	}
}

// OptionsMonth returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs.
//	@Tags			Months
//	@Success		204
//	@Router			/v1/months [options]
func (co Controller) OptionsMonth(c *gin.Context) {
	httputil.OptionsGetPostDelete(c)
}

// GetMonth returns data for a specific budget and month
//
//	@Summary		Get data about a month
//	@Description	Returns data about a specific month.
//	@Tags			Months
//	@Produce		json
//	@Success		200	{object}	MonthResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500		{object}	httperrors.HTTPError
//	@Param			budget	query		string	true	"ID formatted as string"
//	@Param			month	query		string	true	"The month in YYYY-MM format"
//	@Router			/v1/months [get]
func (co Controller) GetMonth(c *gin.Context) {
	qMonth, budget, ok := co.parseMonthQuery(c)
	if !ok {
		return
	}

	// Initialize the response object
	month := Month{
		ID:    budget.ID,
		Name:  budget.Name,
		Month: qMonth,
	}

	// Add budgeted sum to response
	budgeted, err := budget.Budgeted(co.DB, month.Month)
	if err != nil {
		httperrors.Handler(c, err)
		return
	}
	month.Budgeted = budgeted
	month.Allocation = budgeted

	// Add income to response
	income, err := budget.Income(co.DB, month.Month)
	if err != nil {
		httperrors.Handler(c, err)
		return
	}
	month.Income = income

	// Add available sum to response
	available, err := budget.Available(co.DB, month.Month)
	if err != nil {
		httperrors.Handler(c, err)
		return
	}
	month.Available = available

	// Get all categories to iterate over
	categories, ok := co.getCategoryResources(c, budget.ID)
	if !ok {
		return
	}

	month.Categories = make([]CategoryEnvelopes, 0)
	month.Balance = decimal.Zero

	// Get envelopes for all categories
	for _, category := range categories {
		var categoryEnvelopes CategoryEnvelopes

		// Set the basic category values
		categoryEnvelopes.ID = category.ID
		categoryEnvelopes.Name = category.Name
		categoryEnvelopes.Envelopes = make([]models.EnvelopeMonth, 0)

		var envelopes []models.Envelope

		if !queryWithRetry(c, co.DB.Where(&models.Envelope{
			EnvelopeCreate: models.EnvelopeCreate{
				CategoryID: category.ID,
			},
		}).Find(&envelopes)) {
			return
		}

		for _, envelope := range envelopes {
			envelopeMonth, allocationID, err := envelope.Month(co.DB, month.Month)
			if err != nil {
				httperrors.Handler(c, err)
				return
			}

			// Update the month's summarized data
			month.Balance = month.Balance.Add(envelopeMonth.Balance)
			month.Spent = month.Spent.Add(envelopeMonth.Spent)

			// Update the category's summarized data
			categoryEnvelopes.Balance = categoryEnvelopes.Balance.Add(envelopeMonth.Balance)
			categoryEnvelopes.Spent = categoryEnvelopes.Spent.Add(envelopeMonth.Spent)
			categoryEnvelopes.Allocation = categoryEnvelopes.Allocation.Add(envelopeMonth.Allocation)

			// Set the allocation link. If there is no allocation, we send the collection endpoint.
			// With this, any client will be able to see that the "Budgeted" amount is 0 and therefore
			// send a HTTP POST for creation instead of a patch.
			envelopeMonth.Links.Allocation = fmt.Sprintf("%s/v1/allocations", c.GetString("baseURL"))
			if allocationID != uuid.Nil {
				envelopeMonth.Links.Allocation = fmt.Sprintf("%s/%s", envelopeMonth.Links.Allocation, allocationID)
			}

			categoryEnvelopes.Envelopes = append(categoryEnvelopes.Envelopes, envelopeMonth)
		}

		month.Categories = append(month.Categories, categoryEnvelopes)
	}

	c.JSON(http.StatusOK, MonthResponse{Data: month})
}

// DeleteAllocations deletes all allocations for a month
//
//	@Summary		Delete allocations for a month
//	@Description	Deletes all allocation for the specified month
//	@Tags			Months
//	@Success		204
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500		{object}	httperrors.HTTPError
//	@Param			budget	query		string	true	"ID formatted as string"
//	@Param			month	query		string	true	"The month in YYYY-MM format"
//	@Router			/v1/months [delete]
func (co Controller) DeleteAllocations(c *gin.Context) {
	month, budget, ok := co.parseMonthQuery(c)
	if !ok {
		return
	}

	// We query for all allocations here
	var allocations []models.Allocation

	if !queryWithRetry(c, co.DB.
		Joins("JOIN envelopes ON envelopes.id = allocations.envelope_id").
		Joins("JOIN categories ON categories.id = envelopes.category_id").
		Joins("JOIN budgets on budgets.id = categories.budget_id").
		Where(models.Allocation{AllocationCreate: models.AllocationCreate{Month: month}}).
		Where("budgets.id = ?", budget.ID).
		Find(&allocations)) {
		return
	}

	for _, allocation := range allocations {
		if !queryWithRetry(c, co.DB.Unscoped().Delete(&allocation)) {
			return
		}
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

// SetAllocations sets all allocations for a month
//
//	@Summary		Set allocations for a month
//	@Description	Sets allocations for a month for all envelopes that do not have an allocation yet
//	@Tags			Months
//	@Success		204
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		404
//	@Failure		500		{object}	httperrors.HTTPError
//	@Param			budget	query		string					true	"ID formatted as string"
//	@Param			month	query		string					true	"The month in YYYY-MM format"
//	@Param			mode	body		BudgetAllocationMode	true	"Budget"
//	@Router			/v1/months [post]
func (co Controller) SetAllocations(c *gin.Context) {
	month, _, ok := co.parseMonthQuery(c)
	if !ok {
		return
	}

	// Get the mode to set new allocations in
	var data BudgetAllocationMode
	if err := httputil.BindData(c, &data); err != nil {
		return
	}

	if data.Mode != AllocateLastMonthBudget && data.Mode != AllocateLastMonthSpend {
		httperrors.New(c, http.StatusBadRequest, "The mode must be %s or %s", AllocateLastMonthBudget, AllocateLastMonthSpend)
		return
	}

	pastMonth := month.AddDate(0, -1)
	queryCurrentMonth := co.DB.Select("id").Table("allocations").Where("allocations.envelope_id = envelopes.id AND allocations.month = ?", month)

	// Get all envelopes that do not have an allocation for the target month
	// but for the month before
	var envelopesAmount []struct {
		EnvelopeID uuid.UUID `gorm:"column:id"`
		Amount     decimal.Decimal
	}

	// Get all envelope IDs and allocation amounts where there is no allocation
	// for the request month, but one for the last month
	if !queryWithRetry(c, co.DB.
		Joins("JOIN allocations ON allocations.envelope_id = envelopes.id AND allocations.month = ? AND NOT EXISTS(?)", pastMonth, queryCurrentMonth).
		Select("envelopes.id, allocations.amount").
		Table("envelopes").
		Find(&envelopesAmount)) {
		return
	}

	// Create all new allocations
	for _, allocation := range envelopesAmount {
		// If the mode is the spend of last month, calculate and set it
		amount := allocation.Amount
		if data.Mode == AllocateLastMonthSpend {
			amount = models.Envelope{DefaultModel: models.DefaultModel{ID: allocation.EnvelopeID}}.Spent(co.DB, pastMonth)
		}

		if !queryWithRetry(c, co.DB.Create(&models.Allocation{
			AllocationCreate: models.AllocationCreate{
				EnvelopeID: allocation.EnvelopeID,
				Amount:     amount,
				Month:      month,
			},
		})) {
			return
		}
	}

	c.JSON(http.StatusNoContent, gin.H{})
}
