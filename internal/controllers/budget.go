package controllers

import (
	"net/http"
	"time"

	"github.com/envelope-zero/backend/internal/httputil"
	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
)

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
		r.PATCH("/:budgetId", UpdateBudget)
		r.DELETE("/:budgetId", DeleteBudget)
	}

	// Register the routes for dependent resources
	RegisterAccountRoutes(r.Group("/:budgetId/accounts"))
	RegisterCategoryRoutes(r.Group("/:budgetId/categories"))
	RegisterTransactionRoutes(r.Group("/:budgetId/transactions"))
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Budgets
// @Success      204
// @Failure      500  {object}  httputil.HTTPError
// @Router       /v1/budgets [options]
func OptionsBudgetList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Budgets
// @Success      204
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
// @Param        budget    body      models.BudgetCreate  true  "Budget"
// @Success      201     {object}  models.BudgetResponse
// @Failure      400       {object}  httputil.HTTPError
// @Failure      500  {object}  httputil.HTTPError
// @Router       /v1/budgets [post]
func CreateBudget(c *gin.Context) {
	var data models.Budget

	if status, err := bindData(c, &data); err != nil {
		httputil.NewError(c, status, err)
		return
	}

	models.DB.Create(&data)
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

// @Summary      List all budgets
// @Description  Returns list of budgets
// @Tags         Budgets
// @Produce      json
// @Success      200  {object}  models.BudgetListResponse
// @Failure      500  {object}  httputil.HTTPError
// @Router       /v1/budgets [get]
func GetBudgets(c *gin.Context) {
	var budgets []models.Budget
	models.DB.Find(&budgets)

	c.JSON(http.StatusOK, models.BudgetListResponse{
		Data: budgets,
	})
}

// @Summary      Get a budget
// @Description  Returns a specific budget
// @Tags         Budgets
// @Produce      json
// @Param        budgetId  path      uint64  true  "ID of the budget"
// @Success      200       {object}  models.BudgetResponse
// @Failure      404
// @Failure      500  {object}  httputil.HTTPError
// @Router       /v1/budgets/{budgetId} [get]
func GetBudget(c *gin.Context) {
	budget, err := getBudget(c)
	if err != nil {
		return
	}

	// Parse month from the request
	var month Month
	if err := c.ShouldBind(&month); err != nil {
		httputil.FetchErrorHandler(c, err)
		return
	}

	if !month.Month.IsZero() {
		envelopes, err := getEnvelopes(c)
		if err != nil {
			return
		}

		var envelopeMonths []models.EnvelopeMonth
		for _, envelope := range envelopes {
			envelopeMonths = append(envelopeMonths, envelope.Month(month.Month))
		}

		c.JSON(http.StatusOK, gin.H{"data": models.BudgetMonth{
			ID:        budget.ID,
			Name:      budget.Name,
			Month:     time.Date(month.Month.UTC().Year(), month.Month.UTC().Month(), 1, 0, 0, 0, 0, time.UTC),
			Envelopes: envelopeMonths,
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": budget, "links": map[string]string{
		"accounts":     requestURL(c) + "/accounts",
		"categories":   requestURL(c) + "/categories",
		"transactions": requestURL(c) + "/transactions",
		"month":        requestURL(c) + "?month=YYYY-MM",
	}})
}

// @Summary      Update a budget
// @Description  Update an existing budget
// @Tags         Budgets
// @Accept       json
// @Produce      json
// @Param        budgetId  path      uint64               true  "ID of the budget"
// @Param        budget  body      models.BudgetCreate  true  "Budget"
// @Success      200       {object}  models.BudgetResponse
// @Failure      400     {object}  httputil.HTTPError
// @Failure      404
// @Failure      500  {object}  httputil.HTTPError
// @Router       /v1/budgets/{budgetId} [patch]
func UpdateBudget(c *gin.Context) {
	var budget models.Budget

	err := models.DB.First(&budget, c.Param("budgetId")).Error
	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		httputil.FetchErrorHandler(c, err)
		return
	}

	var data models.Budget
	if status, err := bindData(c, &data); err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	models.DB.Model(&budget).Updates(data)
	c.JSON(http.StatusOK, gin.H{"data": budget})
}

// @Summary      Delete a budget
// @Description  Deletes an existing budget
// @Tags         Budgets
// @Param        budgetId  path  uint64  true  "ID of the budget"
// @Success      204
// @Failure      404
// @Failure      500     {object}  httputil.HTTPError
// @Router       /v1/budgets/{budgetId} [delete]
func DeleteBudget(c *gin.Context) {
	var budget models.Budget
	err := models.DB.First(&budget, c.Param("budgetId")).Error
	if err != nil {
		httputil.FetchErrorHandler(c, err)
		return
	}

	models.DB.Delete(&budget)

	c.JSON(http.StatusNoContent, gin.H{})
}
