package controllers

import (
	"net/http"

	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// swagger:enum AllocationMode
type AllocationMode string

const (
	AllocateLastMonthBudget AllocationMode = "ALLOCATE_LAST_MONTH_BUDGET"
	AllocateLastMonthSpend  AllocationMode = "ALLOCATE_LAST_MONTH_SPEND"
)

type BudgetAllocationMode struct {
	Mode AllocationMode `json:"mode" example:"ALLOCATE_LAST_MONTH_SPEND"` // Mode to allocate budget with
}

type MonthResponse struct {
	Data models.Month `json:"data"` // Data for the month
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
//	@Deprecated		true
func (co Controller) OptionsMonth(c *gin.Context) {
	httputil.OptionsGetPostDelete(c)
}

// GetMonth returns data for a specific budget and month
//
//	@Summary		Get data about a month
//	@Description	Returns data about a specific month.
//	@Tags			Months
//	@Produce		json
//	@Success		200		{object}	MonthResponse
//	@Failure		400		{object}	httperrors.HTTPError
//	@Failure		404		{object}	httperrors.HTTPError
//	@Failure		500		{object}	httperrors.HTTPError
//	@Param			budget	query		string	true	"ID formatted as string"
//	@Param			month	query		string	true	"The month in YYYY-MM format"
//	@Router			/v1/months [get]
//	@Deprecated		true
func (co Controller) GetMonth(c *gin.Context) {
	qMonth, budget, e := co.parseMonthQuery(c)
	if !e.Nil() {
		c.JSON(e.Status, httperrors.HTTPError{
			Error: e.Error(),
		})
		return
	}

	month, err := budget.Month(co.DB, qMonth)
	if err != nil {
		e = httperrors.Parse(c, err)
		c.JSON(e.Status, httperrors.HTTPError{
			Error: e.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, MonthResponse{Data: month})
}

// DeleteAllocations deletes all allocations for a month
//
//	@Summary		Delete allocations for a month
//	@Description	Deletes all allocation for the specified month
//	@Tags			Months
//	@Success		204
//	@Failure		400		{object}	httperrors.HTTPError
//	@Failure		404		{object}	httperrors.HTTPError
//	@Failure		500		{object}	httperrors.HTTPError
//	@Param			budget	query		string	true	"ID formatted as string"
//	@Param			month	query		string	true	"The month in YYYY-MM format"
//	@Router			/v1/months [delete]
//	@Deprecated		true
func (co Controller) DeleteAllocations(c *gin.Context) {
	month, budget, err := co.parseMonthQuery(c)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	// We query for all allocations here
	var allocations []models.Allocation

	if !queryAndHandleErrors(c, co.DB.
		Joins("JOIN envelopes ON envelopes.id = allocations.envelope_id").
		Joins("JOIN categories ON categories.id = envelopes.category_id").
		Joins("JOIN budgets on budgets.id = categories.budget_id").
		Where(models.Allocation{AllocationCreate: models.AllocationCreate{Month: month}}).
		Where("budgets.id = ?", budget.ID).
		Find(&allocations)) {
		return
	}

	for _, allocation := range allocations {
		if !queryAndHandleErrors(c, co.DB.Unscoped().Delete(&allocation)) {
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
//	@Failure		400		{object}	httperrors.HTTPError
//	@Failure		404		{object}	httperrors.HTTPError
//	@Failure		500		{object}	httperrors.HTTPError
//	@Param			budget	query		string					true	"ID formatted as string"
//	@Param			month	query		string					true	"The month in YYYY-MM format"
//	@Param			mode	body		BudgetAllocationMode	true	"Budget"
//	@Router			/v1/months [post]
//	@Deprecated		true
func (co Controller) SetAllocations(c *gin.Context) {
	month, _, err := co.parseMonthQuery(c)
	if !err.Nil() {
		c.JSON(err.Status, httperrors.HTTPError{
			Error: err.Error(),
		})
		return
	}

	// Get the mode to set new allocations in
	var data BudgetAllocationMode
	if err := httputil.BindDataHandleErrors(c, &data); err != nil {
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
	if !queryAndHandleErrors(c, co.DB.
		Joins("JOIN allocations ON allocations.envelope_id = envelopes.id AND envelopes.hidden IS FALSE AND allocations.month = ? AND NOT EXISTS(?)", pastMonth, queryCurrentMonth).
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
			amount = models.Envelope{DefaultModel: models.DefaultModel{ID: allocation.EnvelopeID}}.Spent(co.DB, pastMonth).Neg()
		}

		// Do not create allocations for an amount of 0
		if amount.IsZero() {
			continue
		}

		if !queryAndHandleErrors(c, co.DB.Create(&models.Allocation{
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
