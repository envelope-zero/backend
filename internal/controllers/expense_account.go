package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterExpenseAccountRoutes registers the routes for expenseAccounts with
// the RouterGroup that is passed
func RegisterExpenseAccountRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", func(c *gin.Context) {
			c.Header("allow", "GET, POST")
		})
		r.GET("", GetExpenseAccounts)
		r.POST("", CreateExpenseAccount)
	}

	// Transaction with ID
	{
		r.OPTIONS("/:expenseAccountId", func(c *gin.Context) {
			c.Header("allow", "GET, PATCH, DELETE")
		})
		r.GET("/:expenseAccountId", GetExpenseAccount)
		r.PATCH("/:expenseAccountId", UpdateExpenseAccount)
		r.DELETE("/:expenseAccountId", DeleteExpenseAccount)
	}
}

// CreateExpenseAccount creates a new expenseAccount
func CreateExpenseAccount(c *gin.Context) {
	var data models.CreateExpenseAccount

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	budgetID, _ := strconv.Atoi(c.Param("budgetId"))
	expenseAccount := models.ExpenseAccount{Name: data.Name, BudgetID: budgetID}
	database.DB.Create(&expenseAccount)

	c.JSON(http.StatusOK, gin.H{"data": expenseAccount})
}

// GetExpenseAccounts retrieves all expenseAccounts
func GetExpenseAccounts(c *gin.Context) {
	var expenseAccounts []models.ExpenseAccount
	database.DB.Where("budget_id = ?", c.Param("budgetId")).Find(&expenseAccounts)

	c.JSON(http.StatusOK, gin.H{"data": expenseAccounts})
}

// GetExpenseAccount retrieves an expenseAccount by its ID
func GetExpenseAccount(c *gin.Context) {
	var expenseAccount models.ExpenseAccount
	err := database.DB.First(&expenseAccount, c.Param("expenseAccountId")).Error

	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": expenseAccount})
}

// UpdateExpenseAccount updates an expenseAccount, selected by the ID parameter
func UpdateExpenseAccount(c *gin.Context) {
	var expenseAccount models.ExpenseAccount

	err := database.DB.First(&expenseAccount, c.Param("expenseAccountId")).Error

	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	var data models.ExpenseAccount
	err = c.ShouldBindJSON(&data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	database.DB.Model(&expenseAccount).Updates(data)
	c.JSON(http.StatusOK, gin.H{"data": expenseAccount})
}

// DeleteExpenseAccount removes an expenseAccount, identified by its ID
func DeleteExpenseAccount(c *gin.Context) {
	var expenseAccount models.ExpenseAccount
	err := database.DB.First(&expenseAccount, c.Param("expenseAccountId")).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	database.DB.Delete(&expenseAccount)

	c.JSON(http.StatusOK, gin.H{"data": true})
}
