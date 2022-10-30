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

type BudgetListResponse struct {
	Data []Budget `json:"data"`
}

type BudgetResponse struct {
	Data Budget `json:"data"`
}

type Budget struct {
	models.Budget
	Links BudgetLinks `json:"links"`
}

type BudgetLinks struct {
	Self             string `json:"self" example:"https://example.com/api/v1/budgets/550dc009-cea6-4c12-b2a5-03446eb7b7cf"`
	Accounts         string `json:"accounts" example:"https://example.com/api/v1/accounts?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`
	Categories       string `json:"categories" example:"https://example.com/api/v1/categories?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`
	Envelopes        string `json:"envelopes" example:"https://example.com/api/v1/envelopes?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`
	Transactions     string `json:"transactions" example:"https://example.com/api/v1/transactions?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`
	Month            string `json:"month" example:"https://example.com/api/v1/budgets/550dc009-cea6-4c12-b2a5-03446eb7b7cf/YYYY-MM"`                        // This uses 'YYYY-MM' for clients to replace with the actual year and month.
	GroupedMonth     string `json:"groupedMonth" example:"https://example.com/api/v1/months?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf&month=YYYY-MM"`     // This uses 'YYYY-MM' for clients to replace with the actual year and month.
	MonthAllocations string `json:"monthAllocations" example:"https://example.com/api/v1/months?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf&month=YYYY-MM"` // This uses 'YYYY-MM' for clients to replace with the actual year and month.
}

type BudgetMonthResponse struct {
	Data models.BudgetMonth `json:"data"`
}

type BudgetQueryFilter struct {
	Name     string `form:"name"`
	Note     string `form:"note"`
	Currency string `form:"currency"`
}

// swagger:enum AllocationMode
type AllocationMode string

const (
	AllocateLastMonthBudget AllocationMode = "ALLOCATE_LAST_MONTH_BUDGET"
	AllocateLastMonthSpend  AllocationMode = "ALLOCATE_LAST_MONTH_SPEND"
)

type BudgetAllocationMode struct {
	Mode AllocationMode `json:"mode" example:"ALLOCATE_LAST_MONTH_SPEND"`
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

// @Summary     Allowed HTTP verbs
// @Description Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags        Budgets
// @Success     204
// @Failure     500 {object} httperrors.HTTPError
// @Router      /v1/budgets [options]
func (co Controller) OptionsBudgetList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary     Allowed HTTP verbs
// @Description Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags        Budgets
// @Success     204
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Failure     500      {object} httperrors.HTTPError
// @Param       budgetId path     string true "ID formatted as string"
// @Router      /v1/budgets/{budgetId} [options]
func (co Controller) OptionsBudgetDetail(c *gin.Context) {
	p, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	_, ok := co.getBudgetObject(c, p)
	if !ok {
		return
	}
	httputil.OptionsGetPatchDelete(c)
}

// @Summary     Allowed HTTP verbs
// @Description Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs. **Use OPTIONS /month endpoint with month and budgetId query parameters instead.**
// @Tags        Budgets
// @Success     204
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Failure     500      {object} httperrors.HTTPError
// @Param       budgetId path     string true "ID formatted as string"
// @Param       month    path     string true "The month in YYYY-MM format"
// @Router      /v1/budgets/{budgetId}/{month} [options]
// @Deprecated  true
func (co Controller) OptionsBudgetMonth(c *gin.Context) {
	p, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		httperrors.InvalidMonth(c)
		return
	}

	_, ok := co.getBudgetObject(c, p)
	if !ok {
		return
	}
	httputil.OptionsGet(c)
}

// @Summary     Allowed HTTP verbs
// @Description Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs. **Use OPTIONS /month endpoint with month and budgetId query parameters instead.**
// @Tags        Budgets
// @Success     204
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Failure     500      {object} httperrors.HTTPError
// @Failure     500      {object} httperrors.HTTPError
// @Param       budgetId path     string true "ID formatted as string"
// @Param       month    path     string true "The month in YYYY-MM format"
// @Router      /v1/budgets/{budgetId}/{month}/allocations [options]
// @Deprecated  true
func (co Controller) OptionsBudgetMonthAllocations(c *gin.Context) {
	p, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		httperrors.InvalidMonth(c)
		return
	}

	_, ok := co.getBudgetObject(c, p)
	if !ok {
		return
	}
	httputil.OptionsDelete(c)
}

// @Summary     Create budget
// @Description Creates a new budget
// @Tags        Budgets
// @Accept      json
// @Produce     json
// @Success     201    {object} BudgetResponse
// @Failure     400    {object} httperrors.HTTPError
// @Failure     500    {object} httperrors.HTTPError
// @Param       budget body     models.BudgetCreate true "Budget"
// @Router      /v1/budgets [post]
func (co Controller) CreateBudget(c *gin.Context) {
	var budget models.Budget

	if err := httputil.BindData(c, &budget); err != nil {
		return
	}

	if !queryWithRetry(c, co.DB.Create(&budget)) {
		return
	}

	budgetObject, ok := co.getBudgetObject(c, budget.ID)
	if !ok {
		return
	}

	c.JSON(http.StatusCreated, BudgetResponse{Data: budgetObject})
}

// @Summary     List budgets
// @Description Returns a list of budgets
// @Tags        Budgets
// @Produce     json
// @Success     200 {object} BudgetListResponse
// @Failure     500 {object} httperrors.HTTPError
// @Router      /v1/budgets [get]
// @Router      /v1/budgets [get]
// @Param       name     query string false "Filter by name"
// @Param       note     query string false "Filter by note"
// @Param       currency query string false "Filter by currency"
func (co Controller) GetBudgets(c *gin.Context) {
	var filter BudgetQueryFilter

	// Every parameter is bound into a string, so this will always succeed
	_ = c.Bind(&filter)

	// Get the fields that we're filtering for
	queryFields := httputil.GetURLFields(c.Request.URL, filter)

	var budgets []models.Budget

	if !queryWithRetry(c, co.DB.Where(&models.Budget{
		BudgetCreate: models.BudgetCreate{
			Name:     filter.Name,
			Note:     filter.Note,
			Currency: filter.Currency,
		},
	}, queryFields...).Find(&budgets)) {
		return
	}

	// When there are no budgets, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	budgetObjects := make([]Budget, 0)

	for _, budget := range budgets {
		o, _ := co.getBudgetObject(c, budget.ID)
		budgetObjects = append(budgetObjects, o)
	}

	c.JSON(http.StatusOK, BudgetListResponse{Data: budgetObjects})
}

// @Summary     Get budget
// @Description Returns a specific budget
// @Tags        Budgets
// @Produce     json
// @Success     200 {object} BudgetResponse
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Failure     500      {object} httperrors.HTTPError
// @Param       budgetId path     string true "ID formatted as string"
// @Router      /v1/budgets/{budgetId} [get]
func (co Controller) GetBudget(c *gin.Context) {
	p, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	budgetObject, ok := co.getBudgetObject(c, p)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, BudgetResponse{Data: budgetObject})
}

// @Summary     Get Budget month data
// @Description Returns data about a budget for a for a specific month. **Use GET /month endpoint with month and budgetId query parameters instead.**
// @Tags        Budgets
// @Produce     json
// @Success     200 {object} BudgetMonthResponse
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Failure     500      {object} httperrors.HTTPError
// @Param       budgetId path     string true "ID formatted as string"
// @Param       month    path     string true "The month in YYYY-MM format"
// @Router      /v1/budgets/{budgetId}/{month} [get]
// @Deprecated  true
func (co Controller) GetBudgetMonth(c *gin.Context) {
	p, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	budget, ok := co.getBudgetResource(c, p)
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
	// Set the month to the first of the month at midnight
	month.Month = time.Date(month.Month.Year(), month.Month.Month(), 1, 0, 0, 0, 0, time.UTC)

	var envelopes []models.Envelope

	categories, _ := co.getCategoryResources(c, budget.ID)

	// Get envelopes for all categories
	for _, category := range categories {
		var e []models.Envelope

		if !queryWithRetry(c, co.DB.Where(&models.Envelope{
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
		envelopeMonth, _, err := envelope.Month(co.DB, month.Month)
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

		if !queryWithRetry(c, co.DB.Where(&models.Allocation{
			AllocationCreate: models.AllocationCreate{
				EnvelopeID: envelope.ID,
				Month:      month.Month,
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
	income, err := budget.Income(co.DB, month.Month)
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	// Get the available sum for budgeting
	available, err := budget.Available(co.DB, month.Month)
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	c.JSON(http.StatusOK, BudgetMonthResponse{Data: models.BudgetMonth{
		ID:        budget.ID,
		Name:      budget.Name,
		Month:     month.Month,
		Income:    income,
		Budgeted:  budgeted,
		Envelopes: envelopeMonths,
		Available: available,
	}})
}

// @Summary     Update budget
// @Description Update an existing budget. Only values to be updated need to be specified.
// @Tags        Budgets
// @Accept      json
// @Produce     json
// @Success     200 {object} BudgetResponse
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Failure     500      {object} httperrors.HTTPError
// @Param       budgetId path     string              true "ID formatted as string"
// @Param       budget   body     models.BudgetCreate true "Budget"
// @Router      /v1/budgets/{budgetId} [patch]
func (co Controller) UpdateBudget(c *gin.Context) {
	p, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	budget, ok := co.getBudgetResource(c, p)
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

	if !queryWithRetry(c, co.DB.Model(&budget).Select("", updateFields...).Updates(data)) {
		return
	}

	budgetObject, ok := co.getBudgetObject(c, budget.ID)
	if !ok {
		httperrors.Handler(c, err)
		return
	}
	c.JSON(http.StatusOK, BudgetResponse{Data: budgetObject})
}

// @Summary     Delete budget
// @Description Deletes a budget
// @Tags        Budgets
// @Success     204
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Failure     500      {object} httperrors.HTTPError
// @Param       budgetId path     string true "ID formatted as string"
// @Router      /v1/budgets/{budgetId} [delete]
func (co Controller) DeleteBudget(c *gin.Context) {
	p, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	budget, ok := co.getBudgetResource(c, p)
	if !ok {
		return
	}

	if !queryWithRetry(c, co.DB.Delete(&budget)) {
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

// @Summary     Delete allocations for a month
// @Description Deletes all allocation for the specified month. **Use DELETE /month endpoint with month and budgetId query parameters instead.**
// @Tags        Budgets
// @Success     204
// @Failure     400      {object} httperrors.HTTPError
// @Failure     500      {object} httperrors.HTTPError
// @Param       month    path     string true "The month in YYYY-MM format"
// @Param       budgetId path     string true "Budget ID formatted as string"
// @Router      /v1/budgets/{budgetId}/{month}/allocations [delete]
// @Deprecated  true.
func (co Controller) DeleteAllocationsMonth(c *gin.Context) {
	budgetID, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	// If the budget does not exist, abort the request
	_, ok := co.getBudgetResource(c, budgetID)
	if !ok {
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		httperrors.InvalidMonth(c)
		return
	}

	// As URIMonth has a time_format of YYYY-MM, it is parsed without timezone
	// by gorm. Therefore, we need to create a new time.Time object.
	queryMonth := time.Date(month.Month.Year(), month.Month.Month(), 1, 0, 0, 0, 0, time.UTC)

	// We query for all allocations here
	var allocations []models.Allocation

	if !queryWithRetry(c, co.DB.
		Joins("JOIN envelopes ON envelopes.id = allocations.envelope_id").
		Joins("JOIN categories ON categories.id = envelopes.category_id").
		Joins("JOIN budgets on budgets.id = categories.budget_id").
		Where(models.Allocation{AllocationCreate: models.AllocationCreate{Month: queryMonth}}).
		Where("budgets.id = ?", budgetID).
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

// @Summary     Set allocations for a month
// @Description Sets allocations for a month for all envelopes that do not have an allocation yet. **Use POST /month endpoint with month and budgetId query parameters instead.**
// @Tags        Budgets
// @Success     204
// @Failure     400      {object} httperrors.HTTPError
// @Failure     500      {object} httperrors.HTTPError
// @Param       month    path     string               true "The month in YYYY-MM format"
// @Param       budgetId path     string               true "Budget ID formatted as string"
// @Param       mode     body     BudgetAllocationMode true "Budget"
// @Router      /v1/budgets/{budgetId}/{month}/allocations [post]
// @Deprecated  true.
func (co Controller) SetAllocationsMonth(c *gin.Context) {
	budgetID, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	// If the budget does not exist, abort the request
	_, ok := co.getBudgetResource(c, budgetID)
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

	// As URIMonth has a time_format of YYYY-MM, it is parsed without timezone
	// by gorm. Therefore, we need to create a new time.Time object.
	requestMonth := time.Date(month.Month.Year(), month.Month.Month(), 1, 0, 0, 0, 0, time.UTC)
	pastMonth := requestMonth.AddDate(0, -1, 0)

	queryCurrentMonth := co.DB.Select("id").Table("allocations").Where("allocations.envelope_id = envelopes.id AND allocations.month = ?", requestMonth)

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
				Month:      requestMonth,
			},
		})) {
			return
		}
	}

	c.JSON(http.StatusNoContent, gin.H{})
}

// getBudgetResource is the internal helper to verify permissions and return a budget.
//
// It returns a Budget and a boolean indicating success.
func (co Controller) getBudgetResource(c *gin.Context, id uuid.UUID) (models.Budget, bool) {
	if id == uuid.Nil {
		httperrors.New(c, http.StatusBadRequest, "No budget ID specified")
		return models.Budget{}, false
	}

	var budget models.Budget

	if !queryWithRetry(c, co.DB.Where(&models.Budget{
		DefaultModel: models.DefaultModel{
			ID: id,
		},
	}).First(&budget), "No budget found for the specified ID") {
		return models.Budget{}, false
	}

	return budget.WithCalculations(co.DB), true
}

func (co Controller) getBudgetObject(c *gin.Context, id uuid.UUID) (Budget, bool) {
	resource, ok := co.getBudgetResource(c, id)
	if !ok {
		return Budget{}, false
	}

	url := fmt.Sprintf("%s/v1/budgets/%s", c.GetString("baseURL"), id)

	return Budget{
		resource,
		BudgetLinks{
			Self:             url,
			Accounts:         fmt.Sprintf("%s/v1/accounts?budget=%s", c.GetString("baseURL"), resource.ID),
			Categories:       fmt.Sprintf("%s/v1/categories?budget=%s", c.GetString("baseURL"), resource.ID),
			Envelopes:        fmt.Sprintf("%s/v1/envelopes?budget=%s", c.GetString("baseURL"), resource.ID),
			Transactions:     fmt.Sprintf("%s/v1/transactions?budget=%s", c.GetString("baseURL"), resource.ID),
			Month:            url + "/YYYY-MM",
			GroupedMonth:     fmt.Sprintf("%s/v1/months?budget=%s&month=YYYY-MM", c.GetString("baseURL"), resource.ID),
			MonthAllocations: fmt.Sprintf("%s/v1/months?budget=%s&month=YYYY-MM", c.GetString("baseURL"), resource.ID),
		},
	}, true
}
