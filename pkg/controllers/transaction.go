package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/envelope-zero/backend/internal/database"
	"github.com/envelope-zero/backend/pkg/httputil"
	"github.com/envelope-zero/backend/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	Self string `json:"self" example:"https://example.com/api/v1/transactions/d430d7c3-d14c-4712-9336-ee56965a6673"`
}

type TransactionQueryFilter struct {
	Date                 time.Time       `form:"date"`
	Amount               decimal.Decimal `form:"amount"`
	Note                 string          `form:"note"`
	BudgetID             string          `form:"budget"`
	SourceAccountID      string          `form:"source"`
	DestinationAccountID string          `form:"destination"`
	EnvelopeID           string          `form:"envelope"`
	Reconciled           bool            `form:"reconciled"`
}

func (f TransactionQueryFilter) ToCreate(c *gin.Context) (models.TransactionCreate, error) {
	budgetID, err := httputil.UUIDFromString(c, f.BudgetID)
	if err != nil {
		return models.TransactionCreate{}, err
	}

	sourceAccountID, err := httputil.UUIDFromString(c, f.SourceAccountID)
	if err != nil {
		return models.TransactionCreate{}, err
	}

	destinationAccountID, err := httputil.UUIDFromString(c, f.DestinationAccountID)
	if err != nil {
		return models.TransactionCreate{}, err
	}

	envelopeID, err := httputil.UUIDFromString(c, f.EnvelopeID)
	if err != nil {
		return models.TransactionCreate{}, err
	}

	// If the envelopeID is nil, use an actual nil, not uuid.Nil
	var eID *uuid.UUID = nil
	if envelopeID != uuid.Nil {
		eID = &envelopeID
	}

	return models.TransactionCreate{
		Date:                 f.Date,
		Amount:               f.Amount,
		Note:                 f.Note,
		BudgetID:             budgetID,
		SourceAccountID:      sourceAccountID,
		DestinationAccountID: destinationAccountID,
		EnvelopeID:           eID,
		Reconciled:           f.Reconciled,
	}, nil
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

// @Summary     Allowed HTTP verbs
// @Description Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags        Transactions
// @Success     204
// @Router      /v1/transactions [options]
func OptionsTransactionList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// @Summary     Allowed HTTP verbs
// @Description Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
// @Tags        Transactions
// @Success     204
// @Param       transactionId path string true "ID formatted as string"
// @Router      /v1/transactions/{transactionId} [options]
func OptionsTransactionDetail(c *gin.Context) {
	p, err := uuid.Parse(c.Param("transactionId"))
	if err != nil {
		httputil.ErrorInvalidUUID(c)
		return
	}

	_, err = getTransactionObject(c, p)
	if err != nil {
		return
	}
	httputil.OptionsGetPatchDelete(c)
}

// @Summary     Create transaction
// @Description Creates a new transaction
// @Tags        Transactions
// @Produce     json
// @Success     201 {object} TransactionResponse
// @Failure     400 {object} httputil.HTTPError
// @Failure     404
// @Failure     500         {object} httputil.HTTPError
// @Param       transaction body     models.TransactionCreate true "Transaction"
// @Router      /v1/transactions [post]
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

	// Check the source account
	_, err = getAccountResource(c, transaction.SourceAccountID)
	if err != nil {
		return
	}

	// Check the destination account
	_, err = getAccountResource(c, transaction.DestinationAccountID)
	if err != nil {
		return
	}

	// Check the envelope ID only if it is set. (This will always evaluate to true for incoming and outgoing transactions,
	// but for transfers, can evaluate to false
	if transaction.EnvelopeID != nil {
		_, err = getEnvelopeResource(c, *transaction.EnvelopeID)
		if err != nil {
			return
		}
	}

	if !decimal.Decimal.IsPositive(transaction.Amount) {
		httputil.NewError(c, http.StatusBadRequest, errors.New("The transaction amount must be positive"))
		return
	}

	database.DB.Create(&transaction)

	transactionObject, _ := getTransactionObject(c, transaction.ID)
	c.JSON(http.StatusCreated, TransactionResponse{Data: transactionObject})
}

// @Summary     Get transactions
// @Description Returns a list of transactions
// @Tags        Transactions
// @Produce     json
// @Success     200 {object} TransactionListResponse
// @Failure     400 {object} httputil.HTTPError
// @Failure     404
// @Failure     500 {object} httputil.HTTPError
// @Router      /v1/transactions [get]
// @Param       date        query time.Time       false "Filter by date"
// @Param       amount      query decimal.Decimal false "Filter by amount"
// @Param       note        query string          false "Filter by note"
// @Param       budget      query string          false "Filter by budget ID"
// @Param       source      query string          false "Filter by source account ID"
// @Param       destination query string          false "Filter by destination account ID"
// @Param       envelope    query string          false "Filter by envelope ID"
// @Param       reconciled  query bool            false "Filter by reconcilication state"
func GetTransactions(c *gin.Context) {
	var filter TransactionQueryFilter
	if err := c.Bind(&filter); err != nil {
		httputil.ErrorInvalidQueryString(c)
		return
	}

	// Get the fields set in the filter
	queryFields := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	create, err := filter.ToCreate(c)
	if err != nil {
		return
	}

	var transactions []models.Transaction
	database.DB.Order("date(date) DESC").Where(&models.Transaction{
		TransactionCreate: create,
	}, queryFields...).Find(&transactions)

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

// @Summary     Get transaction
// @Description Returns a specific transaction
// @Tags        Transactions
// @Produce     json
// @Success     200 {object} TransactionResponse
// @Failure     400 {object} httputil.HTTPError
// @Failure     404
// @Failure     500           {object} httputil.HTTPError
// @Param       transactionId path     string true "ID formatted as string"
// @Router      /v1/transactions/{transactionId} [get]
func GetTransaction(c *gin.Context) {
	p, err := uuid.Parse(c.Param("transactionId"))
	if err != nil {
		httputil.ErrorInvalidUUID(c)
		return
	}

	transactionObject, err := getTransactionObject(c, p)
	if err != nil {
		return
	}

	c.JSON(http.StatusOK, TransactionResponse{Data: transactionObject})
}

// @Summary     Update transaction
// @Description Updates an existing transaction. Only values to be updated need to be specified.
// @Tags        Transactions
// @Accept      json
// @Produce     json
// @Success     200 {object} TransactionResponse
// @Failure     400 {object} httputil.HTTPError
// @Failure     404
// @Failure     500           {object} httputil.HTTPError
// @Param       transactionId path     string                   true "ID formatted as string"
// @Param       transaction   body     models.TransactionCreate true "Transaction"
// @Router      /v1/transactions/{transactionId} [patch]
func UpdateTransaction(c *gin.Context) {
	p, err := uuid.Parse(c.Param("transactionId"))
	if err != nil {
		httputil.ErrorInvalidUUID(c)
		return
	}

	transaction, err := getTransactionResource(c, p)
	if err != nil {
		return
	}

	updateFields, err := httputil.GetBodyFields(c, models.TransactionCreate{})
	if err != nil {
		return
	}

	var data models.Transaction
	if err := httputil.BindData(c, &data); err != nil {
		return
	}

	// If the amount set via the API request is not existent or
	// is 0, we use the old amount
	if data.Amount.IsZero() {
		data.Amount = transaction.Amount
	}

	if !decimal.Decimal.IsPositive(data.Amount) {
		httputil.NewError(c, http.StatusBadRequest, errors.New("The transaction amount must positive"))
		return
	}

	// Check the source account
	sourceAccountID := transaction.SourceAccountID
	if data.SourceAccountID != uuid.Nil {
		sourceAccountID = data.SourceAccountID
	}
	sourceAccount, err := getAccountResource(c, sourceAccountID)
	if err != nil {
		return
	}

	// Check the destination account
	destinationAccountID := transaction.DestinationAccountID
	if data.DestinationAccountID != uuid.Nil {
		destinationAccountID = data.DestinationAccountID
	}
	destinationAccount, err := getAccountResource(c, destinationAccountID)
	if err != nil {
		return
	}

	// Check if the transaction is a transfer. If yes, the envelope can be empty.
	//
	// Check that the Envelope ID is set for incoming and outgoing transactions
	if sourceAccount.External || destinationAccount.External && data.EnvelopeID == nil {
		httputil.NewError(c, http.StatusBadRequest, errors.New("For incoming and outgoing transactions, an envelope is required"))
		return
	}

	err = database.DB.Model(&transaction).Select("", updateFields...).Updates(data).Error
	if err != nil {
		httputil.ErrorHandler(c, err)
		return
	}

	transactionObject, _ := getTransactionObject(c, p)
	c.JSON(http.StatusOK, TransactionResponse{Data: transactionObject})
}

// @Summary     Delete transaction
// @Description Deletes a transaction
// @Tags        Transactions
// @Success     204
// @Failure     400 {object} httputil.HTTPError
// @Failure     404
// @Failure     500           {object} httputil.HTTPError
// @Param       transactionId path     string true "ID formatted as string"
// @Router      /v1/transactions/{transactionId} [delete]
func DeleteTransaction(c *gin.Context) {
	p, err := uuid.Parse(c.Param("transactionId"))
	if err != nil {
		httputil.ErrorInvalidUUID(c)
		return
	}

	transaction, err := getTransactionResource(c, p)
	if err != nil {
		return
	}

	database.DB.Delete(&transaction)

	c.JSON(http.StatusNoContent, gin.H{})
}

// getTransactionResource verifies that the request URI is valid for the transaction and returns it.
func getTransactionResource(c *gin.Context, id uuid.UUID) (models.Transaction, error) {
	if id == uuid.Nil {
		err := errors.New("No transaction ID specified")
		httputil.NewError(c, http.StatusBadRequest, err)
		return models.Transaction{}, err
	}

	var transaction models.Transaction

	err := database.DB.First(&transaction, &models.Transaction{
		Model: models.Model{
			ID: id,
		},
	}).Error
	if err != nil {
		httputil.ErrorHandler(c, err)
		return models.Transaction{}, err
	}

	return transaction, nil
}

func getTransactionObject(c *gin.Context, id uuid.UUID) (Transaction, error) {
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
func getTransactionLinks(c *gin.Context, id uuid.UUID) TransactionLinks {
	url := fmt.Sprintf("%s/v1/transactions/%s", c.GetString("baseURL"), id)

	return TransactionLinks{
		Self: url,
	}
}
