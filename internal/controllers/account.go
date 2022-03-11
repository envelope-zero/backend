package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// RegisterAccountRoutes registers the routes for accounts with
// the RouterGroup that is passed
func RegisterAccountRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", func(c *gin.Context) {
			c.Header("allow", "GET, POST")
		})
		r.GET("", GetAccounts)
		r.POST("", CreateAccount)
	}

	// Account with ID
	{
		r.OPTIONS("/:accountId", func(c *gin.Context) {
			c.Header("allow", "GET, PATCH, DELETE")
		})
		r.GET("/:accountId", GetAccount)
		r.PATCH("/:accountId", UpdateAccount)
		r.DELETE("/:accountId", DeleteAccount)
	}

	// Transactions
	{
		r.OPTIONS("/:accountId/transactions", func(c *gin.Context) {
			c.Header("allow", "GET")
		})
		r.GET("/:accountId/transactions", GetAccountTransactions)
	}
}

// GetAccountTransactions returns all transactions for the account
func GetAccountTransactions(c *gin.Context) {
	var account models.Account
	err := models.DB.First(&account, c.Param("accountId")).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "No record found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": account.Transactions()})
}

// CreateAccount creates a new account
func CreateAccount(c *gin.Context) {
	var data models.CreateAccount

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	budgetID, _ := strconv.Atoi(c.Param("budgetId"))
	account := models.Account{Name: data.Name, BudgetID: budgetID, OnBudget: data.OnBudget, Visible: data.Visible}
	models.DB.Create(&account)

	c.JSON(http.StatusOK, gin.H{"data": account})
}

// GetAccounts retrieves all accounts
func GetAccounts(c *gin.Context) {
	var accounts []models.Account
	models.DB.Where("budget_id = ?", c.Param("budgetId")).Find(&accounts)

	c.JSON(http.StatusOK, gin.H{"data": accounts})
}

// GetAccount retrieves an account by its ID
func GetAccount(c *gin.Context) {
	var account models.Account
	err := models.DB.First(&account, c.Param("accountId")).Error
	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	balance, err := account.Balance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": struct {
		Account models.Account
		Balance decimal.Decimal `json:"balance"`
	}{
		Account: account,
		Balance: balance,
	}, "links": map[string]string{
		"transactions": "/transactions",
	}})
}

// UpdateAccount updates an account, selected by the ID parameter
func UpdateAccount(c *gin.Context) {
	var account models.Account

	err := models.DB.First(&account, c.Param("accountId")).Error
	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	var data models.Account
	err = c.ShouldBindJSON(&data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	models.DB.Model(&account).Updates(data)
	c.JSON(http.StatusOK, gin.H{"data": account})
}

// DeleteAccount removes a account, identified by its ID
func DeleteAccount(c *gin.Context) {
	var account models.Account
	err := models.DB.First(&account, c.Param("accountId")).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	models.DB.Delete(&account)

	c.JSON(http.StatusOK, gin.H{"data": true})
}
