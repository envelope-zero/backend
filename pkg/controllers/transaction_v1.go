package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/envelope-zero/backend/v3/pkg/database"
	"github.com/envelope-zero/backend/v3/pkg/httperrors"
	"github.com/envelope-zero/backend/v3/pkg/httputil"
	"github.com/envelope-zero/backend/v3/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
)

// Transaction is the API v1 representation of a Transaction in EZ.
type Transaction struct {
	models.Transaction
	Links struct {
		Self string `json:"self" example:"https://example.com/api/v1/transactions/d430d7c3-d14c-4712-9336-ee56965a6673"` // The transaction itself
	} `json:"links"` // Links for the transaction
}

// links generates HATEOAS links for the transaction.
func (t *Transaction) links(c *gin.Context) {
	// Set links
	t.Links.Self = fmt.Sprintf("%s/v1/transactions/%s", c.GetString(string(database.ContextURL)), t.ID)
}

type TransactionListResponse struct {
	Data []Transaction `json:"data"` // List of transactions
}

type TransactionResponse struct {
	Data Transaction `json:"data"` // Data for the transaction
}

func (co Controller) getTransaction(c *gin.Context, id uuid.UUID) (Transaction, bool) {
	transactionModel, ok := getResourceByIDAndHandleErrors[models.Transaction](c, co, id)
	if !ok {
		return Transaction{}, false
	}

	transaction := Transaction{
		Transaction: transactionModel,
	}

	transaction.links(c)
	return transaction, true
}

type TransactionQueryFilter struct {
	Date                  time.Time       `form:"date" filterField:"false"`              // Exact date. Time is ignored.
	FromDate              time.Time       `form:"fromDate" filterField:"false"`          // From this date. Time is ignored.
	UntilDate             time.Time       `form:"untilDate" filterField:"false"`         // Until this date. Time is ignored.
	Amount                decimal.Decimal `form:"amount"`                                // Exact amount
	AmountLessOrEqual     decimal.Decimal `form:"amountLessOrEqual" filterField:"false"` // Amount less than or equal to this
	AmountMoreOrEqual     decimal.Decimal `form:"amountMoreOrEqual" filterField:"false"` // Amount more than or equal to this
	Note                  string          `form:"note" filterField:"false"`              // Note contains this string
	BudgetID              string          `form:"budget"`                                // ID of the budget
	SourceAccountID       string          `form:"source"`                                // ID of the source account
	DestinationAccountID  string          `form:"destination"`                           // ID of the destination account
	EnvelopeID            string          `form:"envelope"`                              // ID of the envelope
	Reconciled            bool            `form:"reconciled"`                            // DEPRECATED. Do not use, this field does not work as intended. See https://github.com/envelope-zero/backend/issues/528. Use reconciledSource and reconciledDestination instead.
	ReconciledSource      bool            `form:"reconciledSource"`                      // Is the transaction reconciled in the source account?
	ReconciledDestination bool            `form:"reconciledDestination"`                 // Is the transaction reconciled in the destination account?
	AccountID             string          `form:"account" filterField:"false"`           // ID of either source or destination account
}

func (f TransactionQueryFilter) ToCreate(c *gin.Context) (models.TransactionCreate, bool) {
	budgetID, ok := httputil.UUIDFromString(c, f.BudgetID)
	if !ok {
		return models.TransactionCreate{}, false
	}

	sourceAccountID, ok := httputil.UUIDFromString(c, f.SourceAccountID)
	if !ok {
		return models.TransactionCreate{}, false
	}

	destinationAccountID, ok := httputil.UUIDFromString(c, f.DestinationAccountID)
	if !ok {
		return models.TransactionCreate{}, false
	}

	envelopeID, ok := httputil.UUIDFromString(c, f.EnvelopeID)
	if !ok {
		return models.TransactionCreate{}, false
	}

	// If the envelopeID is nil, use an actual nil, not uuid.Nil
	var eID *uuid.UUID
	if envelopeID != uuid.Nil {
		eID = &envelopeID
	}

	return models.TransactionCreate{
		Amount:                f.Amount,
		BudgetID:              budgetID,
		SourceAccountID:       sourceAccountID,
		DestinationAccountID:  destinationAccountID,
		EnvelopeID:            eID,
		Reconciled:            f.Reconciled,
		ReconciledSource:      f.ReconciledSource,
		ReconciledDestination: f.ReconciledDestination,
	}, true
}

// RegisterTransactionRoutes registers the routes for transactions with
// the RouterGroup that is passed.
func (co Controller) RegisterTransactionRoutes(r *gin.RouterGroup) {
	// Root group
	{
		r.OPTIONS("", co.OptionsTransactionList)
		r.GET("", co.GetTransactions)
		r.POST("", co.CreateTransaction)
	}

	// Transaction with ID
	{
		r.OPTIONS("/:transactionId", co.OptionsTransactionDetail)
		r.GET("/:transactionId", co.GetTransaction)
		r.PATCH("/:transactionId", co.UpdateTransaction)
		r.DELETE("/:transactionId", co.DeleteTransaction)
	}
}

// OptionsTransactionList returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Transactions
//	@Success		204
//	@Router			/v1/transactions [options]
func (co Controller) OptionsTransactionList(c *gin.Context) {
	httputil.OptionsGetPost(c)
}

// OptionsTransactionDetail returns the allowed HTTP methods
//
//	@Summary		Allowed HTTP verbs
//	@Description	Returns an empty response with the HTTP Header "allow" set to the allowed HTTP verbs
//	@Tags			Transactions
//	@Success		204
//	@Failure		400				{object}	httperrors.HTTPError
//	@Failure		404				{object}	httperrors.HTTPError
//	@Failure		500				{object}	httperrors.HTTPError
//	@Param			transactionId	path		string	true	"ID formatted as string"
//	@Router			/v1/transactions/{transactionId} [options]
func (co Controller) OptionsTransactionDetail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("transactionId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	var t models.Transaction
	err = co.DB.First(&t, id).Error
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	httputil.OptionsGetPatchDelete(c)
}

// CreateTransaction creates a new transaction
//
//	@Summary		Create transaction
//	@Description	Creates a new transaction
//	@Tags			Transactions
//	@Produce		json
//	@Success		201			{object}	TransactionResponse
//	@Failure		400			{object}	httperrors.HTTPError
//	@Failure		404			{object}	httperrors.HTTPError
//	@Failure		500			{object}	httperrors.HTTPError
//	@Param			transaction	body		models.TransactionCreate	true	"Transaction"
//	@Router			/v1/transactions [post]
//	@Deprecated		true
func (co Controller) CreateTransaction(c *gin.Context) {
	var transactionCreate models.TransactionCreate

	if err := httputil.BindData(c, &transactionCreate); err != nil {
		return
	}

	transaction := models.Transaction{
		TransactionCreate: transactionCreate,
	}

	transaction, err := co.createTransaction(c, transaction)
	if !err.Nil() {
		c.JSON(err.Status, err.Body())
		return
	}

	transactionObject, ok := co.getTransaction(c, transaction.ID)
	if !ok {
		return
	}

	c.JSON(http.StatusCreated, TransactionResponse{Data: transactionObject})
}

// GetTransactions returns transactions filtered by the query parameters
//
//	@Summary		Get transactions
//	@Description	Returns a list of transactions
//	@Tags			Transactions
//	@Produce		json
//	@Success		200	{object}	TransactionListResponse
//	@Failure		400	{object}	httperrors.HTTPError
//	@Failure		500	{object}	httperrors.HTTPError
//	@Router			/v1/transactions [get]
//	@Param			date					query	string	false	"Date of the transaction. Ignores exact time, matches on the day of the RFC3339 timestamp provided."
//	@Param			fromDate				query	string	false	"Transactions at and after this date. Ignores exact time, matches on the day of the RFC3339 timestamp provided."
//	@Param			untilDate				query	string	false	"Transactions before and at this date. Ignores exact time, matches on the day of the RFC3339 timestamp provided."
//	@Param			amount					query	string	false	"Filter by amount"
//	@Param			amountLessOrEqual		query	string	false	"Amount less than or equal to this"
//	@Param			amountMoreOrEqual		query	string	false	"Amount more than or equal to this"
//	@Param			note					query	string	false	"Filter by note"
//	@Param			budget					query	string	false	"Filter by budget ID"
//	@Param			account					query	string	false	"Filter by ID of associated account, regardeless of source or destination"
//	@Param			source					query	string	false	"Filter by source account ID"
//	@Param			destination				query	string	false	"Filter by destination account ID"
//	@Param			envelope				query	string	false	"Filter by envelope ID"
//	@Param			reconciled				query	bool	false	"DEPRECATED. Filter by reconcilication state"
//	@Param			reconciledSource		query	bool	false	"Reconcilication state in source account"
//	@Param			reconciledDestination	query	bool	false	"Reconcilication state in destination account"
func (co Controller) GetTransactions(c *gin.Context) {
	var filter TransactionQueryFilter
	if err := c.Bind(&filter); err != nil {
		httperrors.InvalidQueryString(c)
		return
	}

	// Get the fields set in the filter
	queryFields, setFields := httputil.GetURLFields(c.Request.URL, filter)

	// Convert the QueryFilter to a Create struct
	create, ok := filter.ToCreate(c)
	if !ok {
		return
	}

	var query *gorm.DB
	query = co.DB.Order("date(date) DESC").Where(&models.Transaction{
		TransactionCreate: create,
	}, queryFields...)

	// Filter for the transaction being at the same date
	if !filter.Date.IsZero() {
		date := time.Date(filter.Date.Year(), filter.Date.Month(), filter.Date.Day(), 0, 0, 0, 0, time.UTC)
		query = query.Where("transactions.date >= date(?)", date).Where("transactions.date < date(?)", date.AddDate(0, 0, 1))
	}

	if !filter.FromDate.IsZero() {
		query = query.Where("transactions.date >= date(?)", time.Date(filter.FromDate.Year(), filter.FromDate.Month(), filter.FromDate.Day(), 0, 0, 0, 0, time.UTC))
	}

	if !filter.UntilDate.IsZero() {
		query = query.Where("transactions.date < date(?)", time.Date(filter.UntilDate.Year(), filter.UntilDate.Month(), filter.UntilDate.Day()+1, 0, 0, 0, 0, time.UTC))
	}

	if filter.AccountID != "" {
		accountID, ok := httputil.UUIDFromString(c, filter.AccountID)
		if !ok {
			return
		}

		query = query.Where(co.DB.Where(&models.Transaction{
			TransactionCreate: models.TransactionCreate{
				SourceAccountID: accountID,
			},
		}).Or(&models.Transaction{
			TransactionCreate: models.TransactionCreate{
				DestinationAccountID: accountID,
			},
		}))
	}

	if !filter.AmountLessOrEqual.IsZero() {
		query = query.Where("transactions.amount <= ?", filter.AmountLessOrEqual)
	}

	if !filter.AmountMoreOrEqual.IsZero() {
		query = query.Where("transactions.amount >= ?", filter.AmountMoreOrEqual)
	}

	if filter.Note != "" {
		query = query.Where("note LIKE ?", fmt.Sprintf("%%%s%%", filter.Note))
	} else if slices.Contains(setFields, "Note") {
		query = query.Where("note = ''")
	}

	var transactions []models.Transaction
	if !queryWithRetry(c, query.Find(&transactions)) {
		return
	}

	transactionObjects := make([]Transaction, 0)
	for _, t := range transactions {
		transactionObject, ok := co.getTransaction(c, t.ID)
		if !ok {
			return
		}

		transactionObjects = append(transactionObjects, transactionObject)
	}

	c.JSON(http.StatusOK, TransactionListResponse{Data: transactionObjects})
}

// GetTransaction returns a specific transaction
//
//	@Summary		Get transaction
//	@Description	Returns a specific transaction
//	@Tags			Transactions
//	@Produce		json
//	@Success		200				{object}	TransactionResponse
//	@Failure		400				{object}	httperrors.HTTPError
//	@Failure		404				{object}	httperrors.HTTPError
//	@Failure		500				{object}	httperrors.HTTPError
//	@Param			transactionId	path		string	true	"ID formatted as string"
//	@Router			/v1/transactions/{transactionId} [get]
func (co Controller) GetTransaction(c *gin.Context) {
	id, err := uuid.Parse(c.Param("transactionId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	var t models.Transaction
	err = co.DB.First(&t, id).Error
	if err != nil {
		httperrors.Handler(c, err)
		return
	}

	tObject, ok := co.getTransaction(c, t.ID)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, TransactionResponse{Data: tObject})
}

// UpdateTransaction updates a specific transaction
//
//	@Summary		Update transaction
//	@Description	Updates an existing transaction. Only values to be updated need to be specified.
//	@Tags			Transactions
//	@Accept			json
//	@Produce		json
//	@Success		200				{object}	TransactionResponse
//	@Failure		400				{object}	httperrors.HTTPError
//	@Failure		404				{object}	httperrors.HTTPError
//	@Failure		500				{object}	httperrors.HTTPError
//	@Param			transactionId	path		string						true	"ID formatted as string"
//	@Param			transaction		body		models.TransactionCreate	true	"Transaction"
//	@Router			/v1/transactions/{transactionId} [patch]
func (co Controller) UpdateTransaction(c *gin.Context) {
	id, err := uuid.Parse(c.Param("transactionId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	transaction, ok := getResourceByIDAndHandleErrors[models.Transaction](c, co, id)
	if !ok {
		return
	}

	updateFields, err := httputil.GetBodyFields(c, models.TransactionCreate{})
	if err != nil {
		return
	}

	var data models.Transaction
	if err := httputil.BindData(c, &data.TransactionCreate); err != nil {
		return
	}

	// If the amount set via the API request is not existent or
	// is 0, we use the old amount
	if data.Amount.IsZero() {
		data.Amount = transaction.Amount
	}

	// Check the source account
	sourceAccountID := transaction.SourceAccountID
	if data.SourceAccountID != uuid.Nil {
		sourceAccountID = data.SourceAccountID
	}
	sourceAccount, ok := getResourceByIDAndHandleErrors[models.Account](c, co, sourceAccountID)

	if !ok {
		return
	}

	// Check the destination account
	destinationAccountID := transaction.DestinationAccountID
	if data.DestinationAccountID != uuid.Nil {
		destinationAccountID = data.DestinationAccountID
	}
	destinationAccount, ok := getResourceByIDAndHandleErrors[models.Account](c, co, destinationAccountID)

	if !ok {
		return
	}

	// Check the transaction that is set
	if !co.checkTransactionAndHandleErrors(c, data, sourceAccount, destinationAccount) {
		return
	}

	if !queryWithRetry(c, co.DB.Model(&transaction).Select("", updateFields...).Updates(data)) {
		return
	}

	transactionObject, ok := co.getTransaction(c, transaction.ID)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, TransactionResponse{Data: transactionObject})
}

// DeleteTransaction deletes a specific transaction
//
//	@Summary		Delete transaction
//	@Description	Deletes a transaction
//	@Tags			Transactions
//	@Success		204
//	@Failure		400				{object}	httperrors.HTTPError
//	@Failure		404				{object}	httperrors.HTTPError
//	@Failure		500				{object}	httperrors.HTTPError
//	@Param			transactionId	path		string	true	"ID formatted as string"
//	@Router			/v1/transactions/{transactionId} [delete]
func (co Controller) DeleteTransaction(c *gin.Context) {
	id, err := uuid.Parse(c.Param("transactionId"))
	if err != nil {
		httperrors.InvalidUUID(c)
		return
	}

	transaction, ok := getResourceByIDAndHandleErrors[models.Transaction](c, co, id)
	if !ok {
		return
	}

	if !queryWithRetry(c, co.DB.Delete(&transaction)) {
		return
	}

	c.JSON(http.StatusNoContent, gin.H{})
}
