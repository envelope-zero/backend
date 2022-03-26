package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
)

// RegisterAccountRoutes registers the routes for accounts with
// the RouterGroup that is passed.
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

// GetAccountTransactions returns all transactions for the account.
func GetAccountTransactions(c *gin.Context) {
	var account models.Account
	err := models.DB.First(&account, c.Param("accountId")).Error
	if err != nil {
		fetchErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": account.Transactions()})
}

// CreateAccount creates a new account.
func CreateAccount(c *gin.Context) {
	var data models.Account

	if status, err := bindData(c, &data); err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	data.BudgetID, _ = strconv.ParseUint(c.Param("budgetId"), 10, 0)
	models.DB.Create(&data)

	c.JSON(http.StatusCreated, gin.H{"data": data})
}

// GetAccounts retrieves all accounts.
func GetAccounts(c *gin.Context) {
	var accounts, apiResponses []models.Account

	models.DB.Where("budget_id = ?", c.Param("budgetId")).Find(&accounts)

	for _, account := range accounts {
		response, _ := account.WithBalance()
		apiResponses = append(apiResponses, *response)
	}

	c.JSON(http.StatusOK, gin.H{"data": apiResponses})
}

// GetAccount retrieves an account by its ID.
func GetAccount(c *gin.Context) {
	var account models.Account
	err := models.DB.First(&account, c.Param("accountId")).Error
	if err != nil {
		fetchErrorHandler(c, err)
		return
	}

	apiResponse, err := account.WithBalance()
	if err != nil {
		fetchErrorHandler(c, fmt.Errorf("could not get values for account %v: %v", account.Name, err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": apiResponse,
		"links": map[string]string{
			"transactions": requestURL(c) + "/transactions",
		},
	})
}

// UpdateAccount updates an account, selected by the ID parameter.
func UpdateAccount(c *gin.Context) {
	var account models.Account

	err := models.DB.First(&account, c.Param("accountId")).Error
	if err != nil {
		fetchErrorHandler(c, err)
		return
	}

	var data models.Account
	if status, err := bindData(c, &data); err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	models.DB.Model(&account).Updates(data)
	c.JSON(http.StatusOK, gin.H{"data": account})
}

// DeleteAccount removes a account, identified by its ID.
func DeleteAccount(c *gin.Context) {
	var account models.Account
	err := models.DB.First(&account, c.Param("accountId")).Error
	if err != nil {
		fetchErrorHandler(c, err)
		return
	}

	models.DB.Delete(&account)

	c.JSON(http.StatusNoContent, gin.H{})
}
