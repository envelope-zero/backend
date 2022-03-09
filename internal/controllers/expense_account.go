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

// RegisterExpenseAccountRoutes registers the routes for accounts with
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
		r.OPTIONS("/:accountId", func(c *gin.Context) {
			c.Header("allow", "GET, PATCH, DELETE")
		})
		r.GET("/:accountId", GetExpenseAccount)
		r.PATCH("/:accountId", UpdateExpenseAccount)
		r.DELETE("/:accountId", DeleteExpenseAccount)
	}
}

// CreateExpenseAccount creates a new account
func CreateExpenseAccount(c *gin.Context) {
	var data models.CreateExpenseAccount

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	budgetID, _ := strconv.Atoi(c.Param("budgetId"))
	account := models.ExpenseAccount{Name: data.Name, BudgetID: budgetID}
	database.DB.Create(&account)

	c.JSON(http.StatusOK, gin.H{"data": account})
}

// GetExpenseAccounts retrieves all accounts
func GetExpenseAccounts(c *gin.Context) {
	var accounts []models.ExpenseAccount
	database.DB.Where("budget_id = ?", c.Param("budgetId")).Find(&accounts)

	c.JSON(http.StatusOK, gin.H{"data": accounts})
}

// GetExpenseAccount retrieves a account by its ID
func GetExpenseAccount(c *gin.Context) {
	var account models.ExpenseAccount
	err := database.DB.First(&account, c.Param("accountId")).Error

	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": account})
}

// UpdateExpenseAccount updates a account, selected by the ID parameter
func UpdateExpenseAccount(c *gin.Context) {
	var account models.ExpenseAccount

	err := database.DB.First(&account, c.Param("accountId")).Error

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

	database.DB.Model(&account).Updates(data)
	c.JSON(http.StatusOK, gin.H{"data": account})
}

// DeleteExpenseAccount removes a account, identified by its ID
func DeleteExpenseAccount(c *gin.Context) {
	var account models.ExpenseAccount
	err := database.DB.First(&account, c.Param("accountId")).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	database.DB.Delete(&account)

	c.JSON(http.StatusOK, gin.H{"data": true})
}
