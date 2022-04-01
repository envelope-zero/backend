package controllers

import (
	"net/http"
	"strconv"

	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// RegisterTransactionRoutes registers the routes for transactions with
// the RouterGroup that is passed.
func RegisterTransactionRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", func(c *gin.Context) {
			c.Header("allow", "GET, POST")
		})
		r.GET("", GetTransactions)
		r.POST("", CreateTransaction)
	}

	// Transaction with ID
	{
		r.OPTIONS("/:transactionId", func(c *gin.Context) {
			c.Header("allow", "GET, PATCH, DELETE")
		})
		r.GET("/:transactionId", GetTransaction)
		r.PATCH("/:transactionId", UpdateTransaction)
		r.DELETE("/:transactionId", DeleteTransaction)
	}
}

// CreateTransaction creates a new transaction.
func CreateTransaction(c *gin.Context) {
	var data models.Transaction

	if status, err := bindData(c, &data); err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// Convert and validate data
	data.BudgetID, _ = strconv.ParseUint(c.Param("budgetId"), 10, 0)
	if !decimal.Decimal.IsPositive(data.Amount) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The transaction amount must be positive"})
		return
	}

	models.DB.Create(&data)

	c.JSON(http.StatusCreated, gin.H{"data": data})
}

// GetTransactions retrieves all transactions.
func GetTransactions(c *gin.Context) {
	var transactions []models.Transaction
	models.DB.Where("budget_id = ?", c.Param("budgetId")).Find(&transactions)

	c.JSON(http.StatusOK, gin.H{"data": transactions})
}

// GetTransaction retrieves an transaction by its ID.
func GetTransaction(c *gin.Context) {
	var transaction models.Transaction
	err := models.DB.First(&transaction, c.Param("transactionId")).Error
	if err != nil {
		fetchErrorHandler(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": transaction})
}

// UpdateTransaction updates an transaction, selected by the ID parameter.
func UpdateTransaction(c *gin.Context) {
	var transaction models.Transaction

	err := models.DB.First(&transaction, c.Param("transactionId")).Error
	if err != nil {
		fetchErrorHandler(c, err)
		return
	}

	var data models.Transaction
	if status, err := bindData(c, &data); err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// If the amount set via the API request is not existant or
	// is 0, we use the old amount
	if data.Amount.IsZero() {
		data.Amount = transaction.Amount
	}

	if !decimal.Decimal.IsPositive(data.Amount) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The transaction amount must positive"})
		return
	}

	models.DB.Model(&transaction).Updates(data)
	c.JSON(http.StatusOK, gin.H{"data": transaction})
}

// DeleteTransaction removes a transaction, identified by its ID.
func DeleteTransaction(c *gin.Context) {
	var transaction models.Transaction
	err := models.DB.First(&transaction, c.Param("transactionId")).Error
	if err != nil {
		fetchErrorHandler(c, err)
		return
	}

	models.DB.Delete(&transaction)

	c.JSON(http.StatusNoContent, gin.H{})
}
