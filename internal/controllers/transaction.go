package controllers

import (
	"net/http"
	"strconv"

	"github.com/envelope-zero/backend/internal/httputil"
	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// RegisterTransactionRoutes registers the routes for transactions with
// the RouterGroup that is passed.
func RegisterTransactionRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", OptionsTransactionList)
		r.GET("", GetTransactions)
		r.POST("", CreateTransaction)
	}

	// Transaction with ID
	{
		r.OPTIONS("/:transactionId", OptionsTransactionDetail)
		r.GET("/:transactionId", GetTransaction)
		r.PATCH("/:transactionId", UpdateTransaction)
		r.DELETE("/:transactionId", DeleteTransaction)
	}
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Transactions
// @Success      204
// @Param        budgetId  path  uint64  true  "ID of the budget"
// @Router       /v1/budgets/{budgetId}/transactions [options]
func OptionsTransactionList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Transactions
// @Success      204
// @Param        budgetId       path  uint64  true  "ID of the budget"
// @Param        transactionId  path  uint64  true  "ID of the transaction"
// @Router       /v1/budgets/{budgetId}/transactions/{transactionId} [options]
func OptionsTransactionDetail(c *gin.Context) {
	httputil.OptionsGetPatchDelete(c)
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

	// Check if the budget exists at all
	budget, err := getBudget(c)
	if err != nil {
		return
	}

	models.DB.Where(&models.Category{
		BudgetID: budget.ID,
	}).Find(&transactions)

	c.JSON(http.StatusOK, gin.H{"data": transactions})
}

// GetTransaction retrieves an transaction by its ID.
func GetTransaction(c *gin.Context) {
	transaction, err := getTransaction(c)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": transaction})
}

// UpdateTransaction updates an transaction, selected by the ID parameter.
func UpdateTransaction(c *gin.Context) {
	var transaction models.Transaction

	err := models.DB.First(&transaction, c.Param("transactionId")).Error
	if err != nil {
		httputil.FetchErrorHandler(c, err)
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
		httputil.FetchErrorHandler(c, err)
		return
	}

	models.DB.Delete(&transaction)

	c.JSON(http.StatusNoContent, gin.H{})
}
