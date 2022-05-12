package controllers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/envelope-zero/backend/internal/httputil"
	"github.com/envelope-zero/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type TransactionListResponse struct {
	Data []Transaction `json:"data"`
}

type TransactionResponse struct {
	Data Transaction `json:"data"`
}

type Transaction struct {
	models.Transaction
	Links TransactionLinks `json:"links"`
}

type TransactionLinks struct {
	Self string `json:"self" example:"https://example.com/api/v1/transactions/1741"`
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
// @Router       /v1/transactions [options]
func OptionsTransactionList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary      Allowed HTTP verbs
// @Description  Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags         Transactions
// @Success      204
// @Param        transactionId  path  uint64  true  "ID of the transaction"
// @Router       /v1/transactions/{transactionId} [options]
func OptionsTransactionDetail(c *gin.Context) {
	httputil.OptionsGetPatchDelete(c)
}

// @Summary      Create transaction
// @Description  Create a new transaction
// @Tags         Transactions
// @Produce      json
// @Success      201  {object}  TransactionResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500          {object}  httputil.HTTPError
// @Param        transaction  body      models.TransactionCreate  true  "Transaction"
// @Router       /v1/transactions [post]
func CreateTransaction(c *gin.Context) {
	var transaction models.Transaction

	if err := httputil.BindData(c, &transaction); err != nil {
		return
	}

	// Check if the budget that the transaction shoud belong to exists
	_, err := getBudgetResource(c, transaction.BudgetID)
	if err != nil {
		return
	}

	if !decimal.Decimal.IsPositive(transaction.Amount) {
		httputil.NewError(c, http.StatusBadRequest, errors.New("The transaction amount must be positive"))
		return
	}

	models.DB.Create(&transaction)

	transactionObject, _ := getTransactionObject(c, transaction.ID)
	c.JSON(http.StatusCreated, TransactionResponse{Data: transactionObject})
}

// @Summary      Get all transactions
// @Description  Returns all transactions for a specific budget
// @Tags         Transactions
// @Produce      json
// @Success      200  {object}  TransactionListResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500  {object}  httputil.HTTPError
// @Router       /v1/transactions [get]
func GetTransactions(c *gin.Context) {
	var transactions []models.Transaction

	models.DB.Find(&transactions)

	// When there are no resources, we want an empty list, not null
	// Therefore, we use make to create a slice with zero elements
	// which will be marshalled to an empty JSON array
	transactionObjects := make([]Transaction, 0)
	for _, transaction := range transactions {
		o, _ := getTransactionObject(c, transaction.ID)
		transactionObjects = append(transactionObjects, o)
	}

	c.JSON(http.StatusOK, TransactionListResponse{Data: transactionObjects})
}

// @Summary      Get transaction
// @Description  Returns a transaction by its ID
// @Tags         Transactions
// @Produce      json
// @Success      200  {object}  TransactionResponse
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500            {object}  httputil.HTTPError
// @Param        transactionId  path      uint64  true  "ID of the transaction"
// @Router       /v1/transactions/{transactionId} [get]
func GetTransaction(c *gin.Context) {
	id, err := httputil.ParseID(c, "transactionId")
	if err != nil {
		return
	}

	transactionObject, err := getTransactionObject(c, id)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, TransactionResponse{Data: transactionObject})
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
// @Param        transactionId  path      uint64                    true  "ID of the transaction"
// @Param        transaction    body      models.TransactionCreate  true  "Transaction"
// @Router       /v1/transactions/{transactionId} [patch]
func UpdateTransaction(c *gin.Context) {
	id, err := httputil.ParseID(c, "transactionId")
	if err != nil {
		return
	}

	transaction, err := getTransactionResource(c, id)
	if err != nil {
		return
	}

	var data models.Transaction
	if err := httputil.BindData(c, &data); err != nil {
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
	transactionObject, _ := getTransactionObject(c, id)
	c.JSON(http.StatusOK, TransactionResponse{Data: transactionObject})
}

// @Summary      Delete a transaction
// @Description  Deletes an existing transaction
// @Tags         Transactions
// @Success      204
// @Failure      400  {object}  httputil.HTTPError
// @Failure      404
// @Failure      500            {object}  httputil.HTTPError
// @Param        transactionId  path      uint64  true  "ID of the transaction"
// @Router       /v1/transactions/{transactionId} [delete]
func DeleteTransaction(c *gin.Context) {
	id, err := httputil.ParseID(c, "transactionId")
	if err != nil {
		return
	}

	transaction, err := getTransactionResource(c, id)
	if err != nil {
		return
	}

	models.DB.Delete(&transaction)

	c.JSON(http.StatusNoContent, gin.H{})
}

// getTransactionResource verifies that the request URI is valid for the transaction and returns it.
func getTransactionResource(c *gin.Context, id uint64) (models.Transaction, error) {
	var transaction models.Transaction

	err := models.DB.First(&transaction, &models.Transaction{
		Model: models.Model{
			ID: id,
		},
	}).Error
	if err != nil {
		httputil.FetchErrorHandler(c, err)
		return models.Transaction{}, err
	}

	return transaction, nil
}

func getTransactionObject(c *gin.Context, id uint64) (Transaction, error) {
	resource, err := getTransactionResource(c, id)
	if err != nil {
		return Transaction{}, err
	}

	return Transaction{
		resource,
		getTransactionLinks(c, id),
	}, nil
}

// getTransactionLinks returns a TransactionLinks struct.
//
// This function is only needed for getTransactionObject as we cannot create an instance of Transaction
// with mixed named and unnamed parameters.
func getTransactionLinks(c *gin.Context, id uint64) TransactionLinks {
	url := httputil.RequestPathV1(c) + fmt.Sprintf("/transactions/%d", id)

	return TransactionLinks{
		Self: url,
	}
}
