package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/envelope-zero/backend/internal/httputil"
	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
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
	Self         string `json:"self" example:"https://example.com/api/v1/budgets/4"`
	Accounts     string `json:"accounts" example:"https://example.com/api/v1/accounts?budget=2"`
	Categories   string `json:"categories" example:"https://example.com/api/v1/categories?budget=2"`
	Transactions string `json:"transactions" example:"https://example.com/api/v1/budgets/2/transactions"`
	Month        string `json:"month" example:"https://example.com/api/v1/budgets/2/2022-03"`
}

type BudgetMonthResponse struct {
	Data models.BudgetMonth `json:"data"`
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

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Budgets
// @Success      204
// @Failure      500       {object}  httputil.HTTPError
// @Router       /v1/budgets [options]
func OptionsBudgetList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Budgets
// @Success      204
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500       {object}  httputil.HTTPError
// @Param        budgetId  path      uint64  true  "ID of the budget"
// @Router       /v1/budgets/{budgetId} [options]
func OptionsBudgetDetail(c *gin.Context) {
	httputil.OptionsGetPatchDelete(c)
}

// @Summary      Create a budget
// @Description  Creates a new budget
// @Tags         Budgets
// @Accept       json
// @Produce      json
// @Success      201     {object}  BudgetResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      500       {object}  httputil.HTTPError
// @Param        budget    body      models.BudgetCreate  true  "Budget"
// @Router       /v1/budgets [post]
func CreateBudget(c *gin.Context) {
	var budget models.Budget

	if err := httputil.BindData(c, &budget); err != nil {
		return
	}

	models.DB.Create(&budget)

	budgetObject, _ := getBudgetObject(c, budget.ID)
	c.JSON(http.StatusCreated, BudgetResponse{Data: budgetObject})
}

// @Summary      List all budgets
// @Description  Returns list of budgets
// @Tags         Budgets
// @Produce      json
// @Success      200  {object}  BudgetListResponse
// @Failure      500       {object}  httputil.HTTPError
// @Router       /v1/budgets [get]
func GetBudgets(c *gin.Context) {
	var budgets []models.Budget
	models.DB.Find(&budgets)

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

// @Summary      Get a budget
// @Description  Returns a specific budget
// @Tags         Budgets
// @Produce      json
// @Success      200  {object}  BudgetResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500       {object}  httputil.HTTPError
// @Param        budgetId  path      uint64  true  "ID of the budget"
// @Router       /v1/budgets/{budgetId} [get]
func GetBudget(c *gin.Context) {
	id, err := httputil.ParseID(c, "budgetId")
	if err != nil {
		return
	}

	budgetObject, err := getBudgetObject(c, id)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, BudgetResponse{Data: budgetObject})
}

// @Summary      Get Budget month data
// @Description  Returns data about a budget for a for a specific month
// @Tags         Budgets
// @Produce      json
// @Success      200  {object}  BudgetMonthResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500  {object}  httputil.HTTPError
// @Param        budgetId  path      uint64  true  "ID of the budget"
// @Param        month     path      string  true  "The month in YYYY-MM format"
// @Router       /v1/budgets/{budgetId}/{month} [get]
func GetBudgetMonth(c *gin.Context) {
	id, err := httputil.ParseID(c, "budgetId")
	if err != nil {
		return
	}

	budget, err := getBudgetResource(c, id)
	if err != nil {
		return
	}

	var month URIMonth
	if err := c.BindUri(&month); err != nil {
		return
	}

	if month.Month.IsZero() {
		httputil.NewError(c, http.StatusBadRequest, errors.New("You cannot request data for no month"))
		return
	}

	var envelopes []models.Envelope

	categories, _ := getCategoryResources(c, budget.ID)

	// Get envelopes for all categories
	for _, category := range categories {
		var e []models.Envelope

		models.DB.Where(&models.Envelope{
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

// @Summary      Update a budget
// @Description  Update an existing budget. Only values to be updated need to be specified.
// @Tags         Budgets
// @Accept       json
// @Produce      json
// @Success      200  {object}  BudgetResponse
// @Failure      400     {object}  httputil.HTTPError
// @Failure      404
// @Failure      500  {object}  httputil.HTTPError
// @Param        budgetId  path      uint64               true  "ID of the budget"
// @Param        budget  body      models.BudgetCreate  true  "Budget"
// @Router       /v1/budgets/{budgetId} [patch]
func UpdateBudget(c *gin.Context) {
	id, err := httputil.ParseID(c, "budgetId")
	if err != nil {
		return
	}

	budget, err := getBudgetResource(c, id)
	if err != nil {
		return
	}

	var data models.Budget
	if err := httputil.BindData(c, &data); err != nil {
		return
	}

	models.DB.Model(&budget).Updates(data)
	budgetObject, _ := getBudgetObject(c, budget.ID)
	c.JSON(http.StatusOK, BudgetResponse{Data: budgetObject})
}

// @Summary      Delete a budget
// @Description  Deletes an existing budget
// @Tags         Budgets
// @Success      204
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500     {object}  httputil.HTTPError
// @Param        budgetId  path      uint64  true  "ID of the budget"
// @Router       /v1/budgets/{budgetId} [delete]
func DeleteBudget(c *gin.Context) {
	id, err := httputil.ParseID(c, "budgetId")
	if err != nil {
		return
	}

	budget, err := getBudgetResource(c, id)
	if err != nil {
		return
	}

	models.DB.Delete(&budget)

	c.JSON(http.StatusNoContent, gin.H{})
}

// getBudgetResource is the internal helper to verify permissions and return a budget.
func getBudgetResource(c *gin.Context, id uint64) (models.Budget, error) {
	var budget models.Budget

	err := models.DB.Where(&models.Budget{
		Model: models.Model{
			ID: id,
		},
	}).First(&budget).Error
	if err != nil {
		httputil.FetchErrorHandler(c, err)
		return models.Budget{}, err
	}

	return budget, nil
}

func getBudgetObject(c *gin.Context, id uint64) (Budget, error) {
	resource, err := getBudgetResource(c, id)
	if err != nil {
		return Budget{}, err
	}

	return Budget{
		resource,
		getBudgetLinks(c, resource.ID),
	}, nil
}

// getBudgetLinks returns a BudgetLinks struct.
//
// This function is only needed for getBudgetObject as we cannot create an instance of Budget
// with mixed named and unnamed parameters.
func getBudgetLinks(c *gin.Context, id uint64) BudgetLinks {
	url := httputil.RequestPathV1(c) + fmt.Sprintf("/budgets/%d", id)

	return BudgetLinks{
		Self:         url,
		Accounts:     httputil.RequestPathV1(c) + fmt.Sprintf("/accounts?budget=%d", id),
		Categories:   httputil.RequestPathV1(c) + fmt.Sprintf("/categories?budget=%d", id),
		Transactions: httputil.RequestPathV1(c) + fmt.Sprintf("/transactions?budget=%d", id),
		Month:        url + "/YYYY-MM",
	}
}
