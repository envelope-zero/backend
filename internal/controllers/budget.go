package controllers

import (
	"net/http"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
)

// RegisterBudgetRoutes registers the routes for budgets with
// the RouterGroup that is passed.
func RegisterBudgetRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", func(c *gin.Context) {
			c.Header("allow", "GET, POST")
		})
		r.GET("", GetBudgets)
		r.POST("", CreateBudget)
	}

	// Transaction with ID
	{
		r.OPTIONS("/:budgetId", func(c *gin.Context) {
			c.Header("allow", "GET, PATCH, DELETE")
		})
		r.GET("/:budgetId", GetBudget)
		r.PATCH("/:budgetId", UpdateBudget)
		r.DELETE("/:budgetId", DeleteBudget)
	}

	// Register the routes for dependent resources
	RegisterAccountRoutes(r.Group("/:budgetId/accounts"))
	RegisterCategoryRoutes(r.Group("/:budgetId/categories"))
	RegisterTransactionRoutes(r.Group("/:budgetId/transactions"))
}

// CreateBudget creates a new budget.
func CreateBudget(c *gin.Context) {
	var data models.Budget

	if status, err := bindData(c, &data); err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	models.DB.Create(&data)
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

// GetBudgets retrieves all budgets.
func GetBudgets(c *gin.Context) {
	var budgets []models.Budget
	models.DB.Find(&budgets)

	c.JSON(http.StatusOK, gin.H{"data": budgets})
}

// GetBudget retrieves a budget by its ID.
func GetBudget(c *gin.Context) {
	budget, err := getBudget(c)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": budget, "links": map[string]string{
		"accounts":     requestURL(c) + "/accounts",
		"categories":   requestURL(c) + "/categories",
		"transactions": requestURL(c) + "/transactions",
	}})
}

// UpdateBudget updates a budget, selected by the ID parameter.
func UpdateBudget(c *gin.Context) {
	var budget models.Budget

	err := models.DB.First(&budget, c.Param("budgetId")).Error
	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		FetchErrorHandler(c, err)
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

// DeleteBudget removes a budget, identified by its ID.
func DeleteBudget(c *gin.Context) {
	var budget models.Budget
	err := models.DB.First(&budget, c.Param("budgetId")).Error
	if err != nil {
		FetchErrorHandler(c, err)
		return
	}

	models.DB.Delete(&budget)

	c.JSON(http.StatusNoContent, gin.H{})
}
