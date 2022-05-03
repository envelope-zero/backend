package controllers

import (
	"errors"
	"net/http"

	"github.com/envelope-zero/backend/internal/httputil"
	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type TransactionListResponse struct {
	Data []models.Transaction `json:"data"`
}

type TransactionResponse struct {
	Data models.Transaction `json:"data"`
}

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
// @Param        budgetId       path      uint64  true  "ID of the budget"
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

// @Summary      Create transaction
// @Description  Create a new transaction for the specified budget
// @Tags         Transactions
// @Produce      json
// @Success      201  {object}  TransactionResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500          {object}  httputil.HTTPError
// @Param        budgetId     path      uint64                    true  "ID of the budget"
// @Param        transaction  body      models.TransactionCreate  true  "Transaction"
// @Router       /v1/budgets/{budgetId}/transactions [post]
func CreateTransaction(c *gin.Context) {
	var data models.Transaction

	status, err := httputil.BindData(c, &data)
	if err != nil {
		httputil.NewError(c, status, err)
		return
	}

	// Convert and validate data
	data.BudgetID, err = httputil.ParseID(c, "budgetId")
	if err != nil {
		return
	}

	if !decimal.Decimal.IsPositive(data.Amount) {
		httputil.NewError(c, http.StatusBadRequest, errors.New("The transaction amount must be positive"))
		return
	}

	models.DB.Create(&data)
	c.JSON(http.StatusCreated, TransactionResponse{Data: data})
}

// @Summary      Get all transactions
// @Description  Returns all transactions for a specific budget
// @Tags         Transactions
// @Produce      json
// @Success      200  {object}  TransactionListResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500       {object}  httputil.HTTPError
// @Param        budgetId       path      uint64  true  "ID of the budget"
// @Router       /v1/budgets/{budgetId}/transactions [get]
func GetTransactions(c *gin.Context) {
	var transactions []models.Transaction

	// Check if the budget exists at all
	budget, err := getBudgetResource(c)
	if err != nil {
		return
	}

	models.DB.Where(&models.Category{
		CategoryCreate: models.CategoryCreate{
			BudgetID: budget.ID,
		},
	}).Find(&transactions)

	c.JSON(http.StatusOK, TransactionListResponse{Data: transactions})
}

// @Summary      Get transaction
// @Description  Returns a transaction by its ID
// @Tags         Transactions
// @Produce      json
// @Success      200  {object}  TransactionResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500            {object}  httputil.HTTPError
// @Param        budgetId  path      uint64  true  "ID of the budget"
// @Param        transactionId  path      uint64  true  "ID of the transaction"
// @Router       /v1/budgets/{budgetId}/transactions/{transactionId} [get]
func GetTransaction(c *gin.Context) {
	t, err := getTransactionResource(c)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, TransactionResponse{Data: t})
}

// @Summary      Update a transaction
// @Description  Update an existing transaction. Only values to be updated need to be specified.
// @Tags         Transactions
// @Accept       json
// @Produce      json
// @Success      200  {object}  TransactionResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500            {object}  httputil.HTTPError
// @Param        budgetId       path      uint64                    true  "ID of the budget"
// @Param        transactionId  path      uint64                    true  "ID of the transaction"
// @Param        transaction    body      models.TransactionCreate  true  "Transaction"
// @Router       /v1/budgets/{budgetId}/transactions/{transactionId} [patch]
func UpdateTransaction(c *gin.Context) {
	transaction, err := getTransactionResource(c)
	if err != nil {
		return
	}

	var data models.Transaction
	if status, err := httputil.BindData(c, &data); err != nil {
		httputil.NewError(c, status, err)
		return
	}

	// If the amount set via the API request is not existant or
	// is 0, we use the old amount
	if data.Amount.IsZero() {
		data.Amount = transaction.Amount
	}

	if !decimal.Decimal.IsPositive(data.Amount) {
		httputil.NewError(c, http.StatusBadRequest, errors.New("The transaction amount must positive"))
		return
	}

	models.DB.Model(&transaction).Updates(data)
	c.JSON(http.StatusOK, TransactionResponse{Data: transaction})
}

// @Summary      Delete a transaction
// @Description  Deletes an existing transaction
// @Tags         Transactions
// @Success      204
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500            {object}  httputil.HTTPError
// @Param        budgetId  path  uint64  true  "ID of the budget"
// @Param        transactionId  path      uint64  true  "ID of the transaction"
// @Router       /v1/budgets/{budgetId}/transactions/{transactionId} [delete]
func DeleteTransaction(c *gin.Context) {
	transaction, err := getTransactionResource(c)
	if err != nil {
		return
	}

	models.DB.Delete(&transaction)

	c.JSON(http.StatusNoContent, gin.H{})
}

// getTransactionResource verifies that the request URI is valid for the transaction and returns it.
func getTransactionResource(c *gin.Context) (models.Transaction, error) {
	var transaction models.Transaction

	budget, err := getBudgetResource(c)
	if err != nil {
		return models.Transaction{}, err
	}

	accountID, err := httputil.ParseID(c, "transactionId")
	if err != nil {
		return models.Transaction{}, err
	}

	err = models.DB.First(&transaction, &models.Transaction{
		TransactionCreate: models.TransactionCreate{
			BudgetID: budget.ID,
		},
		Model: models.Model{
			ID: accountID,
		},
	}).Error
	if err != nil {
		httputil.FetchErrorHandler(c, err)
		return models.Transaction{}, err
	}

	return transaction, nil
}
