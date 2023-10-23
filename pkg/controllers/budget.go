package controllers

import (
	"net/http"

	"github.com/envelope-zero/backend/v3/internal/types"
	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type BudgetListResponse struct {
	Data []models.Budget `json:"data"` // List of budgets
}

type BudgetResponse struct {
	Data models.Budget `json:"data"` // Data for the budget
}

type BudgetMonthResponse struct {
	Data models.BudgetMonth `json:"data"` // Data for the budget's month
}

type BudgetQueryFilter struct {
	Name     string `form:"name" filterField:"false"`   // By name
	Note     string `form:"note" filterField:"false"`   // By note
	Currency string `form:"currency"`                   // By currency
	Search   string `form:"search" filterField:"false"` // By string in name or note
}

// swagger:enum AllocationMode
type AllocationMode string

const (
	AllocateLastMonthBudget AllocationMode = "ALLOCATE_LAST_MONTH_BUDGET"
	AllocateLastMonthSpend  AllocationMode = "ALLOCATE_LAST_MONTH_SPEND"
)

type BudgetAllocationMode struct {
	Mode AllocationMode `json:"mode" example:"ALLOCATE_LAST_MONTH_SPEND"` // Mode to allocate budget with
}

// RegisterBudgetRoutes registers the routes for budgets with
// the RouterGroup that is passed.
func (co Controller) RegisterBudgetRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", co.OptionsBudgetList)
		r.GET("", co.GetBudgets)
		r.POST("", co.CreateBudget)
	}

	// Budget with ID
	{
		r.OPTIONS("/:budgetId", co.OptionsBudgetDetail)
		r.GET("/:budgetId", co.GetBudget)
		r.OPTIONS("/:budgetId/:month", co.OptionsBudgetMonth)
		r.GET("/:budgetId/:month", co.GetBudgetMonth)
		r.OPTIONS("/:budgetId/:month/allocations", co.OptionsBudgetMonthAllocations)
		r.POST("/:budgetId/:month/allocations", co.SetAllocationsMonth)
		r.DELETE("/:budgetId/:month/allocations", co.DeleteAllocationsMonth)
		r.PATCH("/:budgetId", co.UpdateBudget)
		r.DELETE("/:budgetId", co.DeleteBudget)
	}
}

// OptionsBudgetList returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Budgets
//	@Success		204
//	@Router			/v1/budgets [options]
func (co Controller) OptionsBudgetList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// OptionsBudgetDetail returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Budgets
//	@Success		204
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			budgetId	path		string	true	"ID formatted as string"
//	@Router			/v1/budgets/{budgetId} [options]
func (co Controller) OptionsBudgetDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	_, ok := getResourceByIDAndHandleErrors[models.Budget](c, co, id)
	if !ok {
		return
	}
	httputil.OptionsGetPatchDelete(c)
}

// OptionsBudgetMonth returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs. **Use OPTIONS /month endpoint with month and budgetId query parameters instead.**
//	@Tags			Budgets
//	@Success		204
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			budgetId	path		string	true	"ID formatted as string"
//	@Param			month		path		string	true	"The month in YYYY-MM format"
//	@Router			/v1/budgets/{budgetId}/{month} [options]
//	@Deprecated		true
func (co Controller) OptionsBudgetMonth(c *gin.Context) {
	id, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		httperrors.InvalidMonth(c)
		return
	}

	_, ok := getResourceByIDAndHandleErrors[models.Budget](c, co, id)
	if !ok {
		return
	}
	httputil.OptionsGet(c)
}

// OptionsBudgetMonthAllocations returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs. **Use OPTIONS /month endpoint with month and budgetId query parameters instead.**
//	@Tags			Budgets
//	@Success		204
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			budgetId	path		string	true	"ID formatted as string"
//	@Param			month		path		string	true	"The month in YYYY-MM format"
//	@Router			/v1/budgets/{budgetId}/{month}/allocations [options]
//	@Deprecated		true
func (co Controller) OptionsBudgetMonthAllocations(c *gin.Context) {
	id, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		httperrors.InvalidMonth(c)
		return
	}

	_, ok := getResourceByIDAndHandleErrors[models.Budget](c, co, id)
	if !ok {
		return
	}
	httputil.OptionsDelete(c)
}

// CreateBudget creates a new budget
//
//	@Summary		Create budget
//	@Description	Creates a new budget
//	@Tags			Budgets
//	@Accept			json
//	@Produce		json
//	@Success		201		{object}	BudgetResponse
//	@Failure		400		{object}	httperrors.HTTPError
//	@Failure		500		{object}	httperrors.HTTPError
//	@Param			budget	body		models.BudgetCreate	true	"Budget"
//	@Router			/v1/budgets [post]
func (co Controller) CreateBudget(c *gin.Context) {
	var budget models.Budget

	if err := httputil.BindData(c, &budget); err != nil {
		return
	}

	if !queryAndHandleErrors(c, co.DB.Create(&budget)) {
		return
	}

	c.JSON(http.StatusCreated, BudgetResponse{Data: budget})
}

// GetBudgets returns data for all budgets filtered by the query parameters
//
//	@Summary		List budgets
//	@Description	Returns a list of budgets
//	@Tags			Budgets
//	@Produce		json
//	@Success		200	{object}	BudgetListResponse
//	@Failure		500	{object}	httperrors.HTTPError
//	@Router			/v1/budgets [get]
//	@Param			name		query	string	false	"Filter by name"
//	@Param			note		query	string	false	"Filter by note"
//	@Param			currency	query	string	false	"Filter by currency"
//	@Param			search		query	string	false	"Search for this text in name and note"
func (co Controller) GetBudgets(c *gin.Context) {
	var filter BudgetQueryFilter

	// Every parameter is bound into a string, so this will always succeed
	_ = c.Bind(&filter)

	// Get the fields that we're filtering for
	queryFields, setFields := httputil.GetURLFields(c.Request.URL, filter)

	var budgets []models.Budget

	query := co.DB.Where(&models.Budget{
		BudgetCreate: models.BudgetCreate{
			Name:     filter.Name,
			Note:     filter.Note,
			Currency: filter.Currency,
		},
	}, queryFields...)

	query = stringFilters(co.DB, query, setFields, filter.Name, filter.Note, filter.Search)

	if !queryAndHandleErrors(c, query.Find(&budgets)) {
		return
	}

	// When there are no budgets, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	if len(budgets) == 0 {
		budgets = make([]models.Budget, 0)
	}

	c.JSON(http.StatusOK, BudgetListResponse{Data: budgets})
}

// GetBudget returns data for a single budget
//
//	@Summary		Get budget
//	@Description	Returns a specific budget
//	@Tags			Budgets
//	@Produce		json
//	@Success		200			{object}	BudgetResponse
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			budgetId	path		string	true	"ID formatted as string"
//	@Router			/v1/budgets/{budgetId} [get]
func (co Controller) GetBudget(c *gin.Context) {
	id, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	budgetObject, ok := getResourceByIDAndHandleErrors[models.Budget](c, co, id)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, BudgetResponse{Data: budgetObject})
}

// GetBudgetMonth returns data for a month for a specific budget
//
//	@Summary		Get Budget month data
//	@Description	Returns data about a budget for a for a specific month. **Use GET /month endpoint with month and budgetId query parameters instead.**
//	@Tags			Budgets
//	@Produce		json
//	@Success		200			{object}	BudgetMonthResponse
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			budgetId	path		string	true	"ID formatted as string"
//	@Param			month		path		string	true	"The month in YYYY-MM format"
//	@Router			/v1/budgets/{budgetId}/{month} [get]
//	@Deprecated		true
func (co Controller) GetBudgetMonth(c *gin.Context) {
	id, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	budget, ok := getResourceByIDAndHandleErrors[models.Budget](c, co, id)
	if !ok {
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		httperrors.InvalidMonth(c)
		return
	}

	if month.Month.IsZero() {
		httperrors.New(c, http.StatusBadRequest, "You cannot request data for no month")
		return
	}

	var envelopes []models.Envelope

	categories, _ := co.getCategoryResources(c, budget.ID)

	// Get envelopes for all categories
	for _, category := range categories {
		var e []models.Envelope

		if !queryAndHandleErrors(c, co.DB.Where(&models.Envelope{
			EnvelopeCreate: models.EnvelopeCreate{
				CategoryID: category.ID,
			},
		}).Find(&e)) {
			return
		}

		envelopes = append(envelopes, e...)
	}

	var envelopeMonths []models.EnvelopeMonth
	for _, envelope := range envelopes {
		envelopeMonth, _, err := envelope.Month(co.DB, types.MonthOf(month.Month))
		if err != nil {
			httperrors.Handler(c, err)
			return
		}
		envelopeMonths = append(envelopeMonths, envelopeMonth)
	}

	// Get all allocations for all Envelopes for the month
	var allocations []models.Allocation
	for _, envelope := range envelopes {
		var a models.Allocation

		if !queryAndHandleErrors(c, co.DB.Where(&models.Allocation{
			AllocationCreate: models.AllocationCreate{
				EnvelopeID: envelope.ID,
				Month:      types.MonthOf(month.Month),
			},
		}).Find(&a)) {
			return
		}

		allocations = append(allocations, a)
	}

	// Calculate the budgeted sum
	var budgeted decimal.Decimal
	for _, allocation := range allocations {
		budgeted = budgeted.Add(allocation.Amount)
	}

	// Calculate the income
	income, err := budget.Income(co.DB, types.MonthOf(month.Month))
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	// Get the available sum for budgeting
	bMonth, err := budget.Month(co.DB, types.MonthOf(month.Month))
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	c.JSON(http.StatusOK, BudgetMonthResponse{Data: models.BudgetMonth{
		ID:        budget.ID,
		Name:      budget.Name,
		Month:     types.MonthOf(month.Month),
		Income:    income,
		Budgeted:  budgeted,
		Envelopes: envelopeMonths,
		Available: bMonth.Available,
	}})
}

// UpdateBudget updates data for a budget
//
//	@Summary		Update budget
//	@Description	Update an existing budget. Only values to be updated need to be specified.
//	@Tags			Budgets
//	@Accept			json
//	@Produce		json
//	@Success		200			{object}	BudgetResponse
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			budgetId	path		string				true	"ID formatted as string"
//	@Param			budget		body		models.BudgetCreate	true	"Budget"
//	@Router			/v1/budgets/{budgetId} [patch]
func (co Controller) UpdateBudget(c *gin.Context) {
	id, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	budget, ok := getResourceByIDAndHandleErrors[models.Budget](c, co, id)
	if !ok {
		return
	}

	updateFields, err := httputil.GetBodyFields(c, models.BudgetCreate{})
	if err != nil {
		return
	}

	var data models.Budget
	if err := httputil.BindData(c, &data); err != nil {
		return
	}

	if !queryAndHandleErrors(c, co.DB.Model(&budget).Select("", updateFields...).Updates(data)) {
		return
	}

	c.JSON(http.StatusOK, BudgetResponse{Data: budget})
}

// Do stuff
//
//	@Summary		Delete budget
//	@Description	Deletes a budget
//	@Tags			Budgets
//	@Success		204
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			budgetId	path		string	true	"ID formatted as string"
//	@Router			/v1/budgets/{budgetId} [delete]
func (co Controller) DeleteBudget(c *gin.Context) {
	id, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	budget, ok := getResourceByIDAndHandleErrors[models.Budget](c, co, id)
	if !ok {
		return
	}

	if !queryAndHandleErrors(c, co.DB.Delete(&budget)) {
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

// DeleteAllocationsMonth deletes all allocations for a specific month
//
//	@Summary		Delete allocations for a month
//	@Description	Deletes all allocation for the specified month. **Use DELETE /month endpoint with month and budgetId query parameters instead.**
//	@Tags			Budgets
//	@Success		204
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			month		path		string	true	"The month in YYYY-MM format"
//	@Param			budgetId	path		string	true	"Budget ID formatted as string"
//	@Router			/v1/budgets/{budgetId}/{month}/allocations [delete]
//	@Deprecated		true
func (co Controller) DeleteAllocationsMonth(c *gin.Context) {
	id, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	// If the budget does not exist, abort the request
	_, ok := getResourceByIDAndHandleErrors[models.Budget](c, co, id)
	if !ok {
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		httperrors.InvalidMonth(c)
		return
	}

	// We query for all allocations here
	var allocations []models.Allocation

	if !queryAndHandleErrors(c, co.DB.
		Joins("JOIN envelopes ON envelopes.id = allocations.envelope_id").
		Joins("JOIN categories ON categories.id = envelopes.category_id").
		Joins("JOIN budgets on budgets.id = categories.budget_id").
		Where(models.Allocation{AllocationCreate: models.AllocationCreate{Month: types.MonthOf(month.Month)}}).
		Where("budgets.id = ?", id).
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

// SetAllocationsMonth sets all allocations for a specific month
//
//	@Summary		Set allocations for a month
//	@Description	Sets allocations for a month for all envelopes that do not have an allocation yet. **Deprecated. Use POST /month endpoint with month and budgetId query parameters instead.**
//	@Tags			Budgets
//	@Success		204
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			month		path		string					true	"The month in YYYY-MM format"
//	@Param			budgetId	path		string					true	"Budget ID formatted as string"
//	@Param			mode		body		BudgetAllocationMode	true	"Budget"
//	@Router			/v1/budgets/{budgetId}/{month}/allocations [post]
//	@Deprecated		true
func (co Controller) SetAllocationsMonth(c *gin.Context) {
	id, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	// If the budget does not exist, abort the request
	_, ok := getResourceByIDAndHandleErrors[models.Budget](c, co, id)
	if !ok {
		return
	}

	// Verify the month
	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		httperrors.InvalidMonth(c)
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

	pastMonth := types.MonthOf(month.Month.AddDate(0, -1, 0))

	queryCurrentMonth := co.DB.Select("id").Table("allocations").Where("allocations.envelope_id = envelopes.id AND allocations.month = ?", month.Month)

	// Get all envelopes that do not have an allocation for the target month
	// but for the month before
	var envelopesAmount []struct {
		EnvelopeID uuid.UUID `gorm:"column:id"`
		Amount     decimal.Decimal
	}

	// Get all envelope IDs and allocation amounts where there is no allocation
	// for the request month, but one for the last month
	if !queryAndHandleErrors(c, co.DB.
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

		if !queryAndHandleErrors(c, co.DB.Create(&models.Allocation{
			AllocationCreate: models.AllocationCreate{
				EnvelopeID: allocation.EnvelopeID,
				Amount:     amount,
				Month:      types.MonthOf(month.Month),
			},
		})) {
			return
		}
	}

	c.JSON(http.StatusNoContent, gin.H{})
}
