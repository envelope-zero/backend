package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/envelope-zero/backend/pkg/httperrors"
	"github.com/envelope-zero/backend/pkg/httputil"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	Self         string `json:"self" example:"https://example.com/api/v1/budgets/550dc009-cea6-4c12-b2a5-03446eb7b7cf"`
	Accounts     string `json:"accounts" example:"https://example.com/api/v1/accounts?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`
	Categories   string `json:"categories" example:"https://example.com/api/v1/categories?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`
	Envelopes    string `json:"envelopes" example:"https://example.com/api/v1/envelopes?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`
	Transactions string `json:"transactions" example:"https://example.com/api/v1/transactions?budget=550dc009-cea6-4c12-b2a5-03446eb7b7cf"`
	Month        string `json:"month" example:"https://example.com/api/v1/budgets/550dc009-cea6-4c12-b2a5-03446eb7b7cf/YYYY-MM"` // This will always end in 'YYYY-MM' for clients to use replace with actual numbers.
}

type BudgetMonthResponse struct {
	Data models.BudgetMonth `json:"data"`
}

type BudgetQueryFilter struct {
	Name     string `form:"name"`
	Note     string `form:"note"`
	Currency string `form:"currency"`
}

// RegisterBudgetRoutes registers the routes for budgets with
// the RouterGroup that is passed.
func RegisterBudgetRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", OptionsBudgetList)
		r.GET("", GetBudgets)
		r.POST("", CreateBudget)
	}

	// Budget with ID
	{
		r.OPTIONS("/:budgetId", OptionsBudgetDetail)
		r.GET("/:budgetId", GetBudget)
		r.GET("/:budgetId/:month", GetBudgetMonth)
		r.PATCH("/:budgetId", UpdateBudget)
		r.DELETE("/:budgetId", DeleteBudget)
	}
}

// @Summary     Allowed HTTP verbs
// @Description Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags        Budgets
// @Success     204
// @Failure     500 {object} httperrors.HTTPError
// @Router      /v1/budgets [options]
func OptionsBudgetList(c *gin.Context) {
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
func OptionsBudgetDetail(c *gin.Context) {
	p, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	_, err = getBudgetObject(c, p)
	if err != nil {
		return
	}
	httputil.OptionsGetPatchDelete(c)
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
func CreateBudget(c *gin.Context) {
	var budget models.Budget

	if err := httputil.BindData(c, &budget); err != nil {
		return
	}

	database.DB.Create(&budget)

	budgetObject, _ := getBudgetObject(c, budget.ID)
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
func GetBudgets(c *gin.Context) {
	var filter BudgetQueryFilter

	// Every parameter is bound into a string, so this will always succeed
	_ = c.Bind(&filter)

	// Get the fields that we're filtering for
	queryFields := httputil.GetURLFields(c.Request.URL, filter)

	var budgets []models.Budget
	database.DB.Where(&models.Budget{
		BudgetCreate: models.BudgetCreate{
			Name:     filter.Name,
			Note:     filter.Note,
			Currency: filter.Currency,
		},
	}, queryFields...).Find(&budgets)

	// When there are no budgets, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	budgetObjects := make([]Budget, 0)

	for _, budget := range budgets {
		o, _ := getBudgetObject(c, budget.ID)
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
func GetBudget(c *gin.Context) {
	p, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	budgetObject, err := getBudgetObject(c, p)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, BudgetResponse{Data: budgetObject})
}

// @Summary     Get Budget month data
// @Description Returns data about a budget for a for a specific month
// @Tags        Budgets
// @Produce     json
// @Success     200 {object} BudgetMonthResponse
// @Failure     400 {object} httperrors.HTTPError
// @Failure     404
// @Failure     500      {object} httperrors.HTTPError
// @Param       budgetId path     string true "ID formatted as string"
// @Param       month    path     string true "The month in YYYY-MM format"
// @Router      /v1/budgets/{budgetId}/{month} [get]
func GetBudgetMonth(c *gin.Context) {
	p, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	budget, err := getBudgetResource(c, p)
	if err != nil {
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		return
	}

	if month.Month.IsZero() {
		httperrors.New(c, http.StatusBadRequest, "You cannot request data for no month")
		return
	}

	var envelopes []models.Envelope

	categories, _ := getCategoryResources(c, budget.ID)

	// Get envelopes for all categories
	for _, category := range categories {
		var e []models.Envelope

		database.DB.Where(&models.Envelope{
			EnvelopeCreate: models.EnvelopeCreate{
				CategoryID: category.ID,
			},
		}).Find(&e)

		envelopes = append(envelopes, e...)
	}

	var envelopeMonths []models.EnvelopeMonth
	for _, envelope := range envelopes {
		envelopeMonths = append(envelopeMonths, envelope.Month(month.Month))
	}

	c.JSON(http.StatusOK, BudgetMonthResponse{Data: models.BudgetMonth{
		ID:        budget.ID,
		Name:      budget.Name,
		Month:     time.Date(month.Month.UTC().Year(), month.Month.UTC().Month(), 1, 0, 0, 0, 0, time.UTC),
		Envelopes: envelopeMonths,
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
func UpdateBudget(c *gin.Context) {
	p, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	budget, err := getBudgetResource(c, p)
	if err != nil {
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

	err = database.DB.Model(&budget).Select("", updateFields...).Updates(data).Error
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	budgetObject, _ := getBudgetObject(c, budget.ID)
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
func DeleteBudget(c *gin.Context) {
	p, err := uuid.Parse(c.Param("budgetId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	budget, err := getBudgetResource(c, p)
	if err != nil {
		return
	}

	database.DB.Delete(&budget)

	c.JSON(http.StatusNoContent, gin.H{})
}

// getBudgetResource is the internal helper to verify permissions and return a budget.
func getBudgetResource(c *gin.Context, id uuid.UUID) (models.Budget, error) {
	if id == uuid.Nil {
		err := errors.New("No budget ID specified")
		httperrors.New(c, http.StatusBadRequest, err.Error())
		return models.Budget{}, err
	}

	var budget models.Budget

	err := database.DB.Where(&models.Budget{
		Model: models.Model{
			ID: id,
		},
	}).First(&budget).Error
	if err != nil {
		httperrors.New(c, http.StatusNotFound, "No budget found for the specified ID")
		return models.Budget{}, err
	}

	return budget, nil
}

func getBudgetObject(c *gin.Context, id uuid.UUID) (Budget, error) {
	resource, err := getBudgetResource(c, id)
	if err != nil {
		return Budget{}, err
	}

	return Budget{
		resource.WithCalculations(),
		getBudgetLinks(c, resource.ID),
	}, nil
}

// getBudgetLinks returns a BudgetLinks struct.
//
// This function is only needed for getBudgetObject as we cannot create an instance of Budget
// with mixed named and unnamed parameters.
func getBudgetLinks(c *gin.Context, id uuid.UUID) BudgetLinks {
	url := fmt.Sprintf("%s/v1/budgets/%s", c.GetString("baseURL"), id)

	return BudgetLinks{
		Self:         url,
		Accounts:     fmt.Sprintf("%s/v1/accounts?budget=%s", c.GetString("baseURL"), id),
		Categories:   fmt.Sprintf("%s/v1/categories?budget=%s", c.GetString("baseURL"), id),
		Envelopes:    fmt.Sprintf("%s/v1/envelopes?budget=%s", c.GetString("baseURL"), id),
		Transactions: fmt.Sprintf("%s/v1/transactions?budget=%s", c.GetString("baseURL"), id),
		Month:        url + "/YYYY-MM",
	}
}
