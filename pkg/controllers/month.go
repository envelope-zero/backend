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

type MonthQuery struct {
	Month    time.Time `form:"month" time_format:"2006-01" time_utc:"1" example:"2022-07"`
	BudgetID string    `form:"budget" example:"81b0c9ce-6fd3-4e1e-becc-106055898a2a"`
}

// RegisterMonthRoutes registers the routes for months with
// the RouterGroup that is passed.
func (co Controller) RegisterMonthRoutes(r *gin.RouterGroup) {
	{
		r.OPTIONS("", co.OptionsMonth)
		r.GET("", co.GetMonth)
	}
}

// @Summary     Allowed HTTP verbs
// @Description Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs.
// @Tags        Months
// @Success     204
// @Router      /v1/months [options]
func (co Controller) OptionsMonth(c *gin.Context) {
	httputil.OptionsGet(c)
}

// @Summary     Get data about a month
// @Description Returns data about a specific month.
// @Tags        Months
// @Produce     json
// @Success     200 {object} MonthResponse
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Failure     500    {object} httperrors.HTTPError
// @Param       budget query    string true "ID formatted as string"
// @Param       month  query    string true "The month in YYYY-MM format"
// @Router      /v1/months [get]
func (co Controller) GetMonth(c *gin.Context) {
	var query MonthQuery
	if err := c.Bind(&query); err != nil {
		httperrors.Handler(c, err)
		return
	}

	budgetID, err := uuid.Parse(query.BudgetID)
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	budget, ok := co.getBudgetResource(c, budgetID)
	if !ok {
		return
	}

	if query.Month.IsZero() {
		httperrors.New(c, http.StatusBadRequest, "You cannot request data for no month")
		return
	}
	// Set the month to the first of the month at midnight
	query.Month = time.Date(query.Month.Year(), query.Month.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Initialize the response object
	month := models.Month{
		ID:    budget.ID,
		Name:  budget.Name,
		Month: query.Month,
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
			// Set the allocation link if it is not empty.
			if allocationID != uuid.Nil {
				envelopeMonth.Links.Allocation = fmt.Sprintf("%s/v1/allocations/%s", c.GetString("baseURL"), allocationID)
			}

			categoryEnvelopes.Envelopes = append(categoryEnvelopes.Envelopes, envelopeMonth)
		}

		month.Categories = append(month.Categories, categoryEnvelopes)
	}

	c.JSON(http.StatusOK, MonthResponse{Data: month})
}
