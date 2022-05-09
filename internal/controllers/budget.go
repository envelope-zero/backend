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
	Data []models.Budget `json:"data"`
}

type BudgetResponse struct {
	Data  models.Budget `json:"data"`
	Links BudgetLinks   `json:"links"`
}

type BudgetLinks struct {
	Accounts     string `json:"accounts" example:"https://example.com/api/v1/accounts?budget=2"`
	Categories   string `json:"categories" example:"https://example.com/api/v1/budgets/2/categories"`
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

	RegisterCategoryRoutes(r.Group("/:budgetId/categories"))
	RegisterTransactionRoutes(r.Group("/:budgetId/transactions"))
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
	var data models.Budget

	if err := httputil.BindData(c, &data); err != nil {
		return
	}

	models.DB.Create(&data)
	c.JSON(http.StatusCreated, BudgetResponse{Data: data})
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

	c.JSON(http.StatusOK, BudgetListResponse{
		Data: budgets,
	})
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
	_, err := getBudgetResource(c)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, newBudgetResponse(c))
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
	budget, err := getBudgetResource(c)
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

	envelopes, err := getEnvelopeResources(c)
	if err != nil {
		return
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
	budget, err := getBudgetResource(c)
	if err != nil {
		return
	}

	var data models.Budget
	if err := httputil.BindData(c, &data); err != nil {
		return
	}

	models.DB.Model(&budget).Updates(data)
	c.JSON(http.StatusOK, newBudgetResponse(c))
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
	budget, err := getBudgetResource(c)
	if err != nil {
		return
	}

	models.DB.Delete(&budget)

	c.JSON(http.StatusNoContent, gin.H{})
}

// getBudgetResource verifies that the budget from the URL parameters exists and returns it.
func getBudgetResource(c *gin.Context) (models.Budget, error) {
	var budget models.Budget

	budgetID, err := httputil.ParseID(c, "budgetId")
	if err != nil {
		return models.Budget{}, err
	}

	// Check that the budget exists. If not, return a 404
	err = models.DB.Where(&models.Budget{
		Model: models.Model{
			ID: budgetID,
		},
	}).First(&budget).Error
	if err != nil {
		httputil.FetchErrorHandler(c, err)
		return models.Budget{}, err
	}

	return budget, nil
}

// getBudget is the internal helper to verify permissions and return an account.
func getBudget(c *gin.Context, id uint64) (models.Budget, error) {
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

func newBudgetResponse(c *gin.Context) BudgetResponse {
	// When this function is called, the resource has already been validated
	budget, _ := getBudgetResource(c)

	url := httputil.RequestPathV1(c) + fmt.Sprintf("/budgets/%d", budget.ID)

	return BudgetResponse{
		Data: budget,
		Links: BudgetLinks{
			Accounts:     httputil.RequestPathV1(c) + fmt.Sprintf("/accounts?budget=%d", budget.ID),
			Categories:   url + "/transactions",
			Transactions: url + "/transactions",
			Month:        url + "/YYYY-MM",
		},
	}
}
