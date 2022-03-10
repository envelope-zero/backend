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

// RegisterTransactionRoutes registers the routes for transactions with
// the RouterGroup that is passed
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

// CreateTransaction creates a new transaction
func CreateTransaction(c *gin.Context) {
	var data models.CreateTransaction

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert and validate data
	budgetID, _ := strconv.Atoi(c.Param("budgetId"))
	if !decimal.Decimal.IsPositive(data.Amount) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The transaction amount must be positive"})
		return
	}

	transaction := models.Transaction{
		Date:                 data.Date,
		Amount:               data.Amount,
		Note:                 data.Note,
		BudgetID:             budgetID,
		SourceAccountID:      data.SourceAccountID,
		DestinationAccountID: data.DestinationAccountID,
	}
	models.DB.Create(&transaction)

	c.JSON(http.StatusOK, gin.H{"data": transaction})
}

// GetTransactions retrieves all transactions
func GetTransactions(c *gin.Context) {
	var transactions []models.Transaction
	models.DB.Where("budget_id = ?", c.Param("budgetId")).Find(&transactions)

	c.JSON(http.StatusOK, gin.H{"data": transactions})
}

// GetTransaction retrieves an transaction by its ID
func GetTransaction(c *gin.Context) {
	var transaction models.Transaction
	err := models.DB.First(&transaction, c.Param("transactionId")).Error

	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": transaction})
}

// UpdateTransaction updates an transaction, selected by the ID parameter
func UpdateTransaction(c *gin.Context) {
	var transaction models.Transaction

	err := models.DB.First(&transaction, c.Param("transactionId")).Error

	// Return the apporpriate error: 404 if not found, 500 on all others
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	var data models.Transaction
	err = c.ShouldBindJSON(&data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !decimal.Decimal.IsPositive(data.Amount) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The transaction amount must positive"})
		return
	}

	models.DB.Model(&transaction).Updates(data)
	c.JSON(http.StatusOK, gin.H{"data": transaction})
}

// DeleteTransaction removes a transaction, identified by its ID
func DeleteTransaction(c *gin.Context) {
	var transaction models.Transaction
	err := models.DB.First(&transaction, c.Param("transactionId")).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		}
		return
	}

	models.DB.Delete(&transaction)

	c.JSON(http.StatusOK, gin.H{"data": true})
}
