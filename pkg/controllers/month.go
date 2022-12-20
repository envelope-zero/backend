package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/envelope-zero/backend/pkg/httperrors"
	"github.com/envelope-zero/backend/pkg/httputil"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type MonthResponse struct {
	Data models.Month `json:"data"`
}

// parseQuery takes in the context and parses the request
//
// It verifies that the requested budget exists and parses the ID to return
// the budget resource itself.
func (co Controller) parseMonthQuery(c *gin.Context) (time.Time, models.Budget, bool) {
	var query struct {
		QueryMonth
		BudgetID string `form:"budget" example:"81b0c9ce-6fd3-4e1e-becc-106055898a2a"`
	}

	if err := c.Bind(&query); err != nil {
		httperrors.Handler(c, err)
		return time.Time{}, models.Budget{}, false
	}

	// For a month, we always use the first day at 00:00 UTC
	query.Month = time.Date(query.Month.Year(), query.Month.Month(), 1, 0, 0, 0, 0, time.UTC)

	if query.Month.IsZero() {
		httperrors.New(c, http.StatusBadRequest, "The month query parameter must be set")
		return time.Time{}, models.Budget{}, false
	}

	budgetID, err := uuid.Parse(query.BudgetID)
	if err != nil {
		httperrors.InvalidUUID(c)
		return time.Time{}, models.Budget{}, false
	}

	budget, ok := co.getBudgetResource(c, budgetID)
	if !ok {
		return time.Time{}, models.Budget{}, false
	}

	return query.Month, budget, true
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
	month := models.Month{
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

	month.Categories = make([]models.CategoryEnvelopes, 0)
	month.Balance = decimal.Zero

	// Get envelopes for all categories
	for _, category := range categories {
		var categoryEnvelopes models.CategoryEnvelopes

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

			// Update the month's balance
			month.Balance = month.Balance.Add(envelopeMonth.Balance)
			month.Spent = month.Spent.Add(envelopeMonth.Spent)

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

	pastMonth := month.AddDate(0, -1, 0)
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
